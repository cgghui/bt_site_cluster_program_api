package z_blog

import (
	"fmt"
	"github.com/cgghui/bt_site_cluster_program_api/base"
	"testing"
)

func TestZBlog_Login(t *testing.T) {

	z := &ZBlog{
		HomeURL:       "http://blog.isolezvoscombles.com/",
		BackstagePath: "zb_system/",
		LoginFile:     "cmd.php?act=verify",
	}

	s, err := z.Login("admin", "BeA3UJTRDALFsQNJ")
	if err != nil {
		t.Fatal(err)
	}

	_ = s.NavbarNew(&base.Navbar{
		Href:   "https://www.google.com",
		Title:  "Google",
		Text:   "Google",
		Target: "",
		Sub:    "",
		Ico:    "",
	})

	v := s.(*ZBlogSession)
	csrf := v.GetCSRF()
	fmt.Println(csrf)
	//_ = s.ArticleNew(&base.Article{
	//	Title:    "测试文章",
	//	Content:  "测试文章",
	//	Alias:    "ta",
	//	Tag:      []string{"1", "2", "3", "4"},
	//	Cate:     base.Category{Union: base.Union{ID: "1"}},
	//	Status:   "0",
	//	Template: "single",
	//	AuthorID: "1",
	//	PostTime: time.Now(),
	//	IsTop:    "0",
	//	IsLock:   "0",
	//	Intro:    "测试",
	//})
	//_ = s.ArticleDel(&base.Article{Union: base.Union{ID: "8"}})
	//_ = s.SiteSetting(&base.SiteSetting{SiteName: "ccccc", SubSiteName: "bbbbb"})
	//_ = s.CategoryNew(&base.Category{
	//	Union:       base.Union{ID: "6", Type: "0"},
	//	Name:        "TEST",
	//	Alias:       "TEST",
	//	Order:       "1",
	//	ParentID:    0,
	//	Template:    "index",
	//	LogTemplate: "single",
	//	Intro:       "TEST",
	//	AddNavbar:   "0",
	//})
	//_ = s.CategoryDel(&base.Category{Union: base.Union{ID: "6"}})
}
