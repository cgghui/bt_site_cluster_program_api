package z_blog

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/cgghui/bt_site_cluster_program_api/base"
	"github.com/cgghui/cgghui"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ZBlog struct {
	HomeURL       string // 主站 http://blog.isolezvoscombles.com/
	BackstagePath string // 后台路径 zb_system/
	LoginFile     string // 登录路径 cmd.php?act=verify
}

type ZBlogSession struct {
	zb           ZBlog
	cookie       string
	cookieValues []*http.Cookie
	csrfS        string
	csrfT        time.Time
}

var Client = &http.Client{
	Timeout: 6 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// Login 登录
func (z ZBlog) Login(username, password string) (base.ProgramAPI, error) {
	param := url.Values{}
	param.Set("edtUserName", username)
	param.Set("edtPassWord", password)
	param.Set("btnPost", "登录")
	param.Set("username", username)
	param.Set("password", cgghui.MD5(password))
	param.Set("savedate", "1")
	req, err := http.NewRequest(http.MethodPost, z.HomeURL+z.BackstagePath+z.LoginFile, strings.NewReader(param.Encode()))
	req.Header.Add("User-Agent", base.UserAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 302 {
		return nil, base.LoginFailErr
	}
	s := &ZBlogSession{zb: z, cookie: ""}
	c := make([]string, 0)
	for _, cookie := range resp.Cookies() {
		c = append(c, cookie.Name+"="+cookie.Value)
	}
	s.cookie = strings.Join(c, "; ")
	s.cookieValues = resp.Cookies()
	return s, nil
}

// GetCSRF 获取CSRF
func (s *ZBlogSession) GetCSRF() string {
	if s.csrfS != "" && time.Now().Before(s.csrfT) {
		return s.csrfS
	}
	req, err := s.NewRequest(http.MethodGet, "admin/index.php", nil)
	if err != nil {
		return ""
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		return ""
	}
	csrf := doc.Find(`meta[name="csrfToken"]`).AttrOr("content", "")
	if csrf != "" {
		s.csrfT = time.Now().Add(time.Minute)
		s.csrfS = csrf
	}
	return csrf
}

// ParamCSRF URL参数
func (s *ZBlogSession) ParamCSRF(uri, act string, params ...url.Values) string {
	var param url.Values
	if len(params) > 0 {
		param = params[0]
		if param == nil {
			param = url.Values{}
		}
	} else {
		param = url.Values{}
	}
	param.Set("csrfToken", s.GetCSRF())
	if act != "" {
		param.Set("act", act)
	}
	return uri + "?" + base.UrlQueryBuild(param)
}

// ArticleNew 新建或修改文章
func (s *ZBlogSession) ArticleNew(a *base.Article) error {
	art := url.Values{}
	art.Add("ID", a.ID)
	art.Add("Type", a.Type)
	art.Add("Title", a.Title)
	art.Add("Content", a.Content)
	art.Add("Alias", a.Alias)
	art.Add("Tag", strings.Join(a.Tag, ","))
	art.Add("CateID", a.Cate.ID)
	art.Add("Status", a.Status)
	art.Add("Template", a.Template)
	art.Add("AuthorID", a.AuthorID)
	art.Add("PostTime", a.PostTime.Format("2006-01-02 15:04:05"))
	art.Add("IsTop", a.IsTop)
	art.Add("IsLock", a.IsLock)
	art.Add("Intro", a.Intro)
	req, err := s.NewRequest(http.MethodPost, s.ParamCSRF("cmd.php", "ArticlePst"), art)
	if err != nil {
		return err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}
	if bytes.Contains(body, checkNewArticleSuccess) {
		return nil
	}
	return base.ArticleNewErr
}

// ArticleDel 删除文章
func (s *ZBlogSession) ArticleDel(a *base.Article) error {
	if a.ID == "0" || a.ID == "" {
		return errors.New("请指定文章的id")
	}
	param := url.Values{}
	param.Set("id", a.ID)
	req, err := s.NewRequest(http.MethodGet, s.ParamCSRF("cmd.php", "ArticleDel", param), nil)
	if err != nil {
		return err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == 302 {
		return nil
	}
	return base.ArticleDelErr
}

// SiteSetting 站点设置
func (s *ZBlogSession) SiteSetting(ss *base.SiteSetting) error {
	param := url.Values{}
	param.Set("ZC_BLOG_NAME", ss.SiteName)
	param.Set("ZC_BLOG_SUBNAME", ss.SubSiteName)
	req, err := s.NewRequest(http.MethodPost, s.ParamCSRF("cmd.php", "SettingSav"), param)
	if err != nil {
		return err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == 302 {
		return nil
	}
	return base.SiteSettingErr
}

// CategoryNew 新建或修改分类
func (s *ZBlogSession) CategoryNew(c *base.Category) error {
	cate := url.Values{}
	cate.Add("ID", c.ID)
	cate.Add("Type", c.Type)
	cate.Add("Name", c.Name)
	cate.Add("Alias", c.Alias)
	cate.Add("Order", c.Order)
	cate.Add("Template", c.Template)
	cate.Add("LogTemplate", c.LogTemplate)
	cate.Add("Intro", c.Intro)
	cate.Add("AddNavbar", c.AddNavbar)
	req, err := s.NewRequest(http.MethodPost, s.ParamCSRF("cmd.php", "CategoryPst"), cate)
	if err != nil {
		return err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == 302 {
		return nil
	}
	return base.CategoryNewErr
}

// CategoryDel 删除分类
func (s *ZBlogSession) CategoryDel(c *base.Category) error {
	if c.ID == "0" || c.ID == "" {
		return errors.New("请指定分类的id")
	}
	param := url.Values{}
	param.Set("id", c.ID)
	req, err := s.NewRequest(http.MethodGet, s.ParamCSRF("cmd.php", "CategoryDel", param), nil)
	if err != nil {
		return err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == 302 {
		return nil
	}
	return base.CategoryDelErr
}

var StatusCodeNot200Err = errors.New("status code not 200")

func (s *ZBlogSession) NavbarGet() ([]*base.Navbar, error) {
	param := url.Values{}
	param.Set("edit", "navbar")
	req, err := s.NewRequestHome(http.MethodGet, s.ParamCSRF("zb_users/plugin/LinksManage/main.php", "", param), nil)
	if err != nil {
		return nil, err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return nil, StatusCodeNot200Err
	}
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		return nil, err
	}
	data := make([]*base.Navbar, 0)
	doc.Find("#LinksManageList tr").Each(func(_ int, tr *goquery.Selection) {
		if tr.Find("td").Length() != 6 {
			return
		}
		data = append(data, &base.Navbar{
			Href:   tr.Find(`input[name="href[]"]`).AttrOr("value", ""),
			Title:  tr.Find(`input[name="title[]"]`).AttrOr("value", ""),
			Text:   tr.Find(`input[name="text[]"]`).AttrOr("value", ""),
			Target: tr.Find(`input[name="target[]"]`).AttrOr("value", ""),
			Sub:    tr.Find(`input[name="sub[]"]`).AttrOr("value", ""),
			Ico:    tr.Find(`input[name="ico[]"]`).AttrOr("value", ""),
		})
	})
	return data, nil
}

func (s *ZBlogSession) NavbarNew(n *base.Navbar) error {
	navList, err := s.NavbarGet()
	if err != nil {
		navList = make([]*base.Navbar, 0)
	}
	navList = append(navList, n)
	param := url.Values{}
	param.Set("ID", "1")
	param.Set("Source", "system")
	param.Set("Name", "导航栏")
	param.Set("FileName", "navbar")
	param.Set("HtmlID", "divNavBar")
	param.Set("IsHideTitle", "")
	param.Set("tree", "1")
	param.Set("stay", "1")
	for _, nav := range navList {
		param.Add("href[]", nav.Href)
		param.Add("title[]", nav.Title)
		param.Add("text[]", nav.Text)
		param.Add("target[]", nav.Target)
		param.Add("sub[]", nav.Sub)
		param.Add("ico[]", nav.Ico)
	}
	param.Add("href[]", "")
	param.Add("title[]", "链接描述")
	param.Add("text[]", "链接文本")
	param.Add("target[]", "")
	param.Add("sub[]", "")
	param.Add("ico[]", "")
	var req *http.Request
	req, err = s.NewRequestHome(http.MethodPost, s.ParamCSRF("zb_users/plugin/LinksManage/main.php", "save"), param)
	if err != nil {
		return err
	}
	var resp *http.Response
	req.Header.Add("Referer", s.zb.HomeURL+"zb_users/plugin/LinksManage/main.php?edit=navbar")
	if resp, err = s.RequestAction(req); err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == 302 {
		return nil
	}
	return base.NavbarNewErr
}

func (s *ZBlogSession) NewRequestHome(method, uri string, param url.Values) (*http.Request, error) {
	var body io.Reader
	if param == nil {
		body = nil
	} else {
		body = strings.NewReader(base.UrlQueryBuild(param))
	}
	req, err := http.NewRequest(method, s.zb.HomeURL+uri, body)
	if err != nil {
		return req, err
	}
	req.Header.Add("User-Agent", base.UserAgent)
	req.Header.Add("Cookie", s.cookie)
	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}
	return req, err
}

// NewRequest 发起请求
func (s *ZBlogSession) NewRequest(method, uri string, param url.Values) (*http.Request, error) {
	return s.NewRequestHome(method, s.zb.BackstagePath+"/"+uri, param)
}

func (s *ZBlogSession) RequestAction(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	if resp, err = Client.Do(req); err != nil {
		return nil, err
	}
	if len(resp.Cookies()) > 0 {
		for _, c := range resp.Cookies() {
			has := false
			for i, cv := range s.cookieValues {
				if cv.Name == c.Name {
					s.cookieValues[i] = c
					has = true
					break
				}
			}
			if !has {
				s.cookieValues = append(s.cookieValues, c)
			}
		}
		c := make([]string, 0)
		for _, cookie := range resp.Cookies() {
			c = append(c, cookie.Name+"="+cookie.Value)
		}
		s.cookie = strings.Join(c, "; ")
	}
	return resp, err
}

var checkNewArticleSuccess = []byte("cmd.php%3Fact%3DArticleMng")
