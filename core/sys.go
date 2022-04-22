package core

import (
	"context"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/cgghui/bt_site_cluster/bt"
	"github.com/cgghui/bt_site_cluster/kernel"
	"github.com/cgghui/bt_site_cluster_collect/collect"
	_ "github.com/cgghui/bt_site_cluster_collect/target/nbtimes_net"
	_ "github.com/cgghui/bt_site_cluster_collect/target/techsir_com"
	_ "github.com/cgghui/bt_site_cluster_collect/target/v2_sohu_com"
	"github.com/cgghui/bt_site_cluster_program_api/base"
	_ "github.com/cgghui/bt_site_cluster_program_api/z-blog"
	"log"
	"strings"
	"sync"
	"time"
)

var ErrProgramNotUndefined = errors.New("err program not undefined")

type collectAction struct {
	name string
	s    *SiteConfig
	c    *Category
	api  base.ProgramAPI
	wg   *sync.WaitGroup
}

func (c *collectAction) run() {
	defer c.wg.Done()
	sd := collect.GetStandard(c.name)
	if sd == nil {
		log.Printf("【%s】采集名称未定义 Name: %s", c.s.BindDomain[0], c.name)
		return
	}
	// 标签
	for _, tag := range c.c.Collect.Cate {
		for _, p := range c.c.Collect.Page { // 按页进行采集
			c.s.collect(c.api, sd, tag, p, c.c)
		}
	}
}

var SiteCollectChannel = make(chan *SiteConfig)
var CollectActionChannel = make(chan *collectAction)

func Start() {
	for i := 0; i < 50; i++ {
		go func(No int) {
			for site := range SiteCollectChannel {
				site.CollectAction()
				log.Printf("thread No[%d] work[s] Done, site: %s", No, site.BindDomain[0])
			}
		}(i + 1)
	}
	for i := 0; i < 50; i++ {
		go func(No int) {
			for ca := range CollectActionChannel {
				ca.run()
				log.Printf("thread No[%d] work[c] Done, site: [%s] %s -> %s", No, ca.c.Name, ca.name, ca.s.BindDomain[0])
			}
		}(i + 1)
	}
}

type SiteConfig struct {
	kernel.SiteConfig
	base.ProgramBaseInfo
	base.SiteSetting
	Category     []Category `json:"category"`
	SiteRootPath string     `json:"site_root_path"`
	Username     string     `json:"login_username"`
	Password     string     `json:"login_password"`
	Open         bool       `json:"open"`
	BtO          *bt.Option
	BtS          *bt.Session
}

type Category struct {
	base.Category
	Collect CategoryCollect `json:"collect"`
}

type CategoryCollect struct {
	Name    []string          `json:"name"`
	Page    []int             `json:"page"`
	Cate    []collect.Tag     `json:"cate"`
	Contain []CategoryContain `json:"contain"`
}

type CategoryContain struct {
	Word string `json:"word"`
	Num  int    `json:"occ"`
}

// InContain 判断文章内容是否包含必要的关键词 包含返回true
func (c *CategoryCollect) InContain(art *collect.Article) bool {
	for _, cc := range c.Contain {
		if strings.Count(art.Content, cc.Word) >= cc.Num {
			return true
		}
	}
	return false
}

// Login 登录站点
func (s *SiteConfig) Login() (base.ProgramAPI, error) {
	function := base.GetProgram(s.ProgramName)
	if function == nil {
		return nil, ErrProgramNotUndefined
	}
	return function(s.Username, s.Password, s.ProgramBaseInfo)
}

// CollectAction 采集动作
func (s *SiteConfig) CollectAction() {
	// 创建宝塔登录会话
	if s.BtO.GetLoginSession() == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		ss, err := s.BtO.Login(ctx)
		cancel()
		if err != nil {
			log.Printf("登录宝塔失败 Error: %v", err)
			return
		}
		s.BtO.SetLoginSession(ss)
	}
	// 登录站点
	api, err := s.Login()
	if err != nil {
		log.Printf("【%s】登录%s失败 Error: %v", s.BindDomain[0], s.ProgramName, err)
		return
	}
	if err = api.Init(); err != nil {
		log.Printf("【%s】初始化站点信息失败 Error: %v", s.BindDomain[0], err)
		return
	}
	if err = api.SiteSetting(&s.SiteSetting); err != nil {
		log.Printf("【%s】设定站点基本信息失败 Error: %v", s.BindDomain[0], err)
		return
	}
	wg := sync.WaitGroup{}
	for i, category := range s.Category {
		// 尝试获取分类，分类不存在时，尝试创建分类
		if err = api.CategoryGet(&category.Category); err != nil {
			if err = api.CategoryNew(&category.Category); err != nil {
				log.Printf("【%s】无法创建分类[%s] Error: %v", s.BindDomain[0], category.Category.Name, err)
				continue
			}
		}
		if err = api.CategoryGet(&category.Category); err != nil {
			log.Printf("【%s】无法获取分类[%s] Error: %v", s.BindDomain[0], category.Category.Name, err)
			continue
		}
		for _, name := range category.Collect.Name {
			act := &collectAction{
				wg:   &wg,
				name: name,
				s:    s,
				c:    &s.Category[i],
				api:  api,
			}
			CollectActionChannel <- act
			wg.Add(1)
		}
	}
	wg.Wait()
}

func (s *SiteConfig) collect(api base.ProgramAPI, sd collect.Standard, tag collect.Tag, page int, cc *Category) {
	list, err := sd.ArticleList(tag, page)
	if err != nil {
		log.Printf("【%s】【%s】采集文章列表页 Error: %v", s.BindDomain[0], sd.Name(), err)
		return
	}
	for i := range list {
		art := &list[i]
		info := base.Article{Title: art.Title}
		_ = api.ArticleGet(&info)
		if info.ID != "" {
			continue // 已经发布
		}
		if err = sd.ArticleDetail(&list[i]); err != nil {
			log.Printf("【%s】【%s】《%s》采集文章详细失败 Error: %v", s.BindDomain[0], sd.Name(), art.Title, err)
			continue
		}
		if !cc.Collect.InContain(&list[i]) {
			log.Printf("【%s】【%s】《%s》文章内容未包含有效关键词", s.BindDomain[0], sd.Name(), art.Title)
			continue
		}
		info = base.Article{
			Union:    base.Union{ID: "0", Type: "0"},
			Title:    art.Title,
			Content:  art.Content,
			Alias:    "",
			Tag:      nil,
			Cate:     &cc.Category,
			Status:   "0",
			Template: "single",
			AuthorID: "1",
			PostTime: art.PostTime,
			IsTop:    "0",
			IsLock:   "0",
			Intro:    "",
		}
		// 处理tag
		if len(art.Tag) > 0 {
			if info.Tag == nil {
				info.Tag = make([]string, 0)
			}
			for _, tg := range art.Tag {
				v := base.Tag{}
				if err = api.TagGet(&v); err == base.TagUndefinedErr {
					_ = api.TagNew(&base.Tag{
						Union:     base.Union{ID: "0", Type: "0"},
						Name:      tg.Name,
						Alias:     tg.Tag,
						Template:  "index",
						Intro:     "",
						AddNavbar: "0",
					})
				} else {
					tg.Name = v.Name
					tg.Tag = v.Alias
				}
				info.Tag = append(info.Tag, tg.Name)
			}
			info.Tag = StrSliceDistinct(info.Tag)
			doc, _ := goquery.NewDocumentFromReader(strings.NewReader(info.Content))
			doc.Find(collect.TagClass).Each(func(_ int, k *goquery.Selection) {
				k.SetAttr("href", "/tags-"+k.AttrOr(collect.TagAttrValue, "")+".html")
				k.SetAttr("target", "_blank")
				k.SetText(k.AttrOr(collect.TagAttrName, ""))
				k.RemoveAttr(collect.TagAttrValue)
				k.RemoveAttr(collect.TagAttrName)
			})
			info.Content, _ = doc.Html()
		}
		if err = api.ArticleNew(&info); err != nil {
			log.Printf("【%s】【%s】《%s》文章入库失败 采集文章入库失败 Error: %v", s.BindDomain[0], sd.Name(), list[i].Title, err)
			continue
		}
		for _, local := range art.LocalImages {
			collect.UploadImage(s.BtO.GetLoginSession(), s.SiteRootPath, local)
		}
		log.Printf("【%s】【%s】《%s》文章入库成功 标签[%s] 图片[%d]张", s.BindDomain[0], sd.Name(), art.Title, strings.Join(info.Tag, ","), len(art.LocalImages))
	}
}

func StrSliceDistinct(s []string) []string {
	distinct := make(map[string]struct{})
	for _, v := range s {
		distinct[v] = struct{}{}
	}
	i := 0
	for k := range distinct {
		s[i] = k
		i++
	}
	return s[:len(distinct)]
}
