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

func init() {
	base.RegisterProgram("z-blog", Login)
}

type ZBlogSession struct {
	zb           base.ProgramBaseInfo
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
func Login(username, password string, z base.ProgramBaseInfo) (base.ProgramAPI, error) {
	param := url.Values{}
	param.Set("edtUserName", username)
	param.Set("edtPassWord", password)
	param.Set("btnPost", "登录")
	param.Set("username", username)
	param.Set("password", cgghui.MD5(password))
	param.Set("savedate", "1")
	req, err := http.NewRequest(http.MethodPost, z.HomeURL+z.BackstagePath+z.LoginPath, strings.NewReader(param.Encode()))
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

var ErrOpenRewriteFail = errors.New("open rewrite fail")

func (s *ZBlogSession) Init() error {
	// open rewrite
	param := url.Values{}
	param.Set("name", "STACentre")
	req, err := s.NewRequest(http.MethodGet, s.ParamCSRF("cmd.php", "PluginEnb", param), nil)
	if err != nil {
		return err
	}
	var resp *http.Response
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 302 {
		return ErrOpenRewriteFail
	}
	//
	param = url.Values{}
	param.Set("install", "STACentre")
	req, err = s.NewRequest(http.MethodGet, s.ParamCSRF("cmd.php", "PluginMng", param), nil)
	if err != nil {
		return err
	}
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 302 {
		return ErrOpenRewriteFail
	}
	//
	param = url.Values{}
	param.Set("csrfToken", s.GetCSRF())
	param.Set("reset", "")
	param.Set("ZC_STATIC_MODE", "REWRITE")
	param.Set("ZC_ARTICLE_REGEX", "{%host%}post/{%id%}.html")
	param.Set("ZC_PAGE_REGEX", "{%host%}{%id%}.html")
	param.Set("ZC_INDEX_REGEX", "{%host%}page_{%page%}.html")
	param.Set("ZC_CATEGORY_REGEX", "{%host%}category-{%id%}_{%page%}.html")
	param.Set("ZC_TAGS_REGEX", "{%host%}tags-{%alias%}_{%page%}.html")
	param.Set("radioZC_TAGS_REGEX", "{%host%}tags-{%alias%}_{%page%}.html")
	param.Set("ZC_DATE_REGEX", "{%host%}date-{%date%}_{%page%}.html")
	param.Set("ZC_AUTHOR_REGEX", "{%host%}date-{%date%}_{%page%}.html")
	req, err = s.NewRequestHome(http.MethodPost, s.ParamCSRF("zb_users/plugin/STACentre/main.php", ""), param)
	if err != nil {
		return err
	}
	if resp, err = Client.Do(req); err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 302 {
		return ErrOpenRewriteFail
	}
	return nil
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

func (s *ZBlogSession) ArticleGet(a *base.Article) error {
	art := url.Values{}
	art.Add("category", "")
	art.Add("status", "")
	art.Add("search", a.Title)
	req, err := s.NewRequestHome(http.MethodPost, s.ParamCSRF("zb_system/admin/index.php", "ArticleMng"), art)
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
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		return err
	}
	doc.Find(".table_striped tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		if strings.TrimSpace(tr.Find("td").Eq(3).Text()) == a.Title {
			a.ID = tr.Find("td").Eq(0).Text()
			return false
		}
		return true
	})
	return base.ArticleGetErr
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
	if c.ID == "" {
		cate.Add("ID", "0")
	} else {
		cate.Add("ID", c.ID)
	}
	if c.Type == "" {
		cate.Add("Type", "0")
	} else {
		cate.Add("Type", c.Type)
	}
	cate.Add("Name", c.Name)
	cate.Add("Alias", c.Alias)
	cate.Add("Order", c.Order)
	if c.Template == "" {
		cate.Add("Template", "index")
	} else {
		cate.Add("Template", c.Template)
	}
	if c.LogTemplate == "" {
		cate.Add("LogTemplate", "single")
	} else {
		cate.Add("LogTemplate", c.LogTemplate)
	}
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

// CategoryGet 查找分类
func (s *ZBlogSession) CategoryGet(c *base.Category) error {
	req, err := s.NewRequestHome(http.MethodGet, s.ParamCSRF("zb_system/admin/index.php", "CategoryMng"), nil)
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
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		return err
	}
	doc.Find(".tableBorder-thcenter tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		name := strings.TrimSpace(tr.Find("td").Eq(2).Text())
		if name == c.Name {
			c.ID = strings.TrimSpace(tr.Find("td").Eq(0).Text())
			c.Order = strings.TrimSpace(tr.Find("td").Eq(1).Text())
			c.Alias = strings.TrimSpace(tr.Find("td").Eq(3).Text())

			return false
		}
		return true
	})
	if c.ID != "" {
		return nil
	}
	return base.CategoryGetErr
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

// NavbarList 导航
func (s *ZBlogSession) NavbarList() ([]*base.Navbar, error) {
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

// NavbarNew 创建导航
func (s *ZBlogSession) NavbarNew(n *base.Navbar) error {
	navList, err := s.NavbarList()
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

var duplicateTag = []byte("标签名称重复")

func (s *ZBlogSession) TagNew(t *base.Tag) error {
	param := url.Values{}
	param.Set("ID", t.ID)
	param.Set("Type", t.Type)
	param.Set("Name", t.Name)
	param.Set("Alias", t.Alias)
	param.Set("Template", t.Template)
	param.Set("Intro", t.Intro)
	param.Set("AddNavbar", t.AddNavbar)
	var req *http.Request
	var err error
	req, err = s.NewRequest(http.MethodPost, s.ParamCSRF("cmd.php", "TagPst"), param)
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
	if b, _ := ioutil.ReadAll(resp.Body); bytes.Contains(b, duplicateTag) {
		return nil
	}
	return base.TagNewErr
}

func (s *ZBlogSession) TagGet(t *base.Tag) error {
	param := url.Values{}
	param.Set("search", t.Name)
	var req *http.Request
	var err error
	req, err = s.NewRequestHome(http.MethodPost, s.ParamCSRF("zb_system/admin/index.php", "TagMng"), param)
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
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		return err
	}
	tr := doc.Find(".table_striped tr")
	if tr.Length() == 1 {
		return base.TagUndefinedErr
	}
	td := tr.Eq(1).Find("td")
	t.ID = td.Eq(0).Text()
	t.Name = td.Eq(1).Text()
	t.Alias = td.Eq(2).Text()
	return nil
}

func (s *ZBlogSession) TagDel(t *base.Tag) error {
	var err error
	if err = s.TagGet(t); err != nil {
		return err
	}
	param := url.Values{}
	param.Set("id", t.ID)
	var req *http.Request
	req, err = s.NewRequest(http.MethodGet, s.ParamCSRF("cmd.php", "TagDel", param), nil)
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
		return base.TagDelErr
	}
	return nil
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
