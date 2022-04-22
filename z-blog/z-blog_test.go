package z_blog

import (
	"fmt"
	"github.com/cgghui/bt_site_cluster_program_api/base"
	"testing"
)

func TestZBlog_Login(t *testing.T) {

	z := base.ProgramBaseInfo{
		HomeURL:       "http://blog.isolezvoscombles.com/",
		BackstagePath: "zb_system/",
		LoginPath:     "cmd.php?act=verify",
	}

	s, err := Login("admin", "YPVCX6HqGDN15XCZ", z)
	if err != nil {
		t.Fatal(err)
	}

	var art = base.Article{Title: "遥望网络蹭元宇宙概念，星期六屡次“跨界”失败创巨额亏损"}

	_ = s.ArticleGet(&art)

	//var cate = base.Category{
	//	Name: "未命名a",
	//}
	//_ = s.CategoryGet(&cate)
	//
	//tag := base.Tag{Name: "京东"}
	//_ = s.TagGet(&tag)
	fmt.Println(art)
}
