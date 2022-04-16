package base

import (
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"
)

var LoginFailErr = errors.New("登录失败")
var ArticleNewErr = errors.New("新建文章失败")
var ArticleDelErr = errors.New("删除文章失败")
var SiteSettingErr = errors.New("设置站点失败")
var CategoryNewErr = errors.New("新建分类失败")
var CategoryDelErr = errors.New("删除分类失败")
var NavbarNewErr = errors.New("新建导航失败")

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36"

type SWITCH string

type SiteSetting struct {
	SiteName    string // 网站标题
	SubSiteName string // 网站副标题
}

// Union 通用结构体
type Union struct {
	ID   string
	Type string
}

// Article 文章
type Article struct {
	Union
	Title    string    // 标题
	Content  string    // 正文
	Alias    string    // 别名
	Tag      []string  // 标签
	Cate     Category  // 分类
	Status   string    // 状态		0 公开	1 草稿	2 审核
	Template string    // 内容模板
	AuthorID string    // 作者id
	PostTime time.Time // 发布时间
	IsTop    string    // 置顶		0 无	1 全局	2 首页	3 分类
	IsLock   string    // 评论		0 允许	2 禁止
	Intro    string    // 摘要
}

// Category 分类
type Category struct {
	Union
	Name        string // 名称
	Alias       string // 别名
	Order       string // 排序
	ParentID    int    // 父级ID
	Template    string // 模板 首页及列表页
	LogTemplate string // 模板 文章页（单页）
	Intro       string // 简述
	AddNavbar   string // 是否为导航，或是否在导航显示	0 不显示		1 显示
}

type Navbar struct {
	Href   string `json:"href"`   // 链接
	Title  string `json:"title"`  // 描述
	Text   string `json:"text"`   // 文本
	Target string `json:"target"` // 新窗
	Sub    string `json:"sub"`    // 二级
	Ico    string `json:"ico"`    // 图标（class属性值）
}

type ProgramAPI interface {

	// ArticleNew 新建或修改文章
	// 新建时，指定 Article.ID 为 0
	// 修改时，指定 Article.ID 为 被修改文章的id
	// 必要时可在法内部对 Article 进行更新
	ArticleNew(*Article) error

	// ArticleDel 删除文章 必须指定 Article.ID
	ArticleDel(*Article) error

	// SiteSetting 设置站点
	SiteSetting(*SiteSetting) error

	// CategoryNew 创建或修改分类
	// 新建时，指定 Category.ID 为 0
	// 修改时，指定 Category.ID 为 被修改文章的id
	// 必要时可在法内部对 Category 进行更新
	CategoryNew(*Category) error

	// CategoryDel 删除文章 必须指定 Category.ID
	CategoryDel(*Category) error

	// NavbarNew 创建或修改导航
	// 必要时可在法内部对 Navbar 进行更新
	NavbarNew(*Navbar) error
}

func UrlQueryBuild(value url.Values) string {
	param := make([]string, 0)
	ak := make([]string, 0)
	as := 0
	for k, v := range value {
		if len(v) == 1 {
			param = append(param, url.QueryEscape(k)+"="+url.QueryEscape(v[0]))
			continue
		}
		ak = append(ak, k)
		if l := len(v); l > as {
			as = l
		}
	}
	sort.Strings(ak)
	for i := 0; i < as; i++ {
		for _, k := range ak {
			param = append(param, url.QueryEscape(k)+"="+url.QueryEscape(value[k][i]))
		}
	}
	return strings.Join(param, "&")
}
