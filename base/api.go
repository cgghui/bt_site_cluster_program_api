package base

import (
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var LoginFailErr = errors.New("登录失败")
var SiteSettingErr = errors.New("设置站点失败")
var ArticleGetErr = errors.New("获取文章失败")
var ArticleNewErr = errors.New("新建文章失败")
var ArticleDelErr = errors.New("删除文章失败")
var CategoryNewErr = errors.New("新建分类失败")
var CategoryGetErr = errors.New("获取分类失败")
var CategoryDelErr = errors.New("删除分类失败")
var TagNewErr = errors.New("新建标签失败")
var TagDelErr = errors.New("删除标签失败")
var TagUndefinedErr = errors.New("无法找到标签")
var NavbarNewErr = errors.New("新建导航失败")

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36"

type SiteSetting struct {
	SiteName        string `json:"site_title"`
	SubSiteName     string `json:"site_sub_title"`
	SiteKeywords    string `json:"site_keywords"`
	SiteDescription string `json:"site_description"`
}

// Union 通用结构体
type Union struct {
	ID   string
	Type string
}

// Article 文章
type Article struct {
	Union
	Title    string    `json:"title"`     // 标题
	Content  string    `json:"content"`   // 正文
	Alias    string    `json:"alias"`     // 别名
	Tag      []string  `json:"tag"`       // 标签
	Cate     *Category `json:"cate"`      // 分类
	Status   string    `json:"status"`    // 状态		0 公开	1 草稿	2 审核
	Template string    `json:"template"`  // 内容模板
	AuthorID string    `json:"author_id"` // 作者id
	PostTime time.Time `json:"post_time"` // 发布时间
	IsTop    string    `json:"is_top"`    // 置顶		0 无	1 全局	2 首页	3 分类
	IsLock   string    `json:"is_lock"`   // 评论		0 允许	2 禁止
	Intro    string    `json:"intro"`     // 摘要
}

// Category 分类
type Category struct {
	Union
	Name        string `json:"name"`         // 名称
	Alias       string `json:"alias"`        // 别名
	Order       string `json:"order"`        // 排序
	ParentID    int    `json:"parent_id"`    // 父级ID
	Template    string `json:"template"`     // 模板 首页及列表页
	LogTemplate string `json:"log_template"` // 模板 文章页（单页）
	Intro       string `json:"intro"`        // 简述
	AddNavbar   string `json:"add_navbar"`   // 是否为导航，或是否在导航显示	0 不显示		1 显示
}

// Navbar 导航
type Navbar struct {
	Href   string `json:"href"`   // 链接
	Title  string `json:"title"`  // 描述
	Text   string `json:"text"`   // 文本
	Target string `json:"target"` // 新窗
	Sub    string `json:"sub"`    // 二级
	Ico    string `json:"ico"`    // 图标（class属性值）
}

// Tag 标签
type Tag struct {
	Union
	Name      string `json:"name"`
	Alias     string `json:"alias"`
	Template  string `json:"template"`
	Intro     string `json:"intro"`
	AddNavbar string `json:"add_navbar"`
}

type ProgramAPI interface {

	// Init 初始化
	Init() error

	// SiteSetting 设置站点
	SiteSetting(*SiteSetting) error

	// ArticleNew 新建或修改文章
	ArticleNew(*Article) error

	// ArticleGet 获取文章
	ArticleGet(*Article) error

	// ArticleDel 删除文章 必须指定 Article.ID
	ArticleDel(*Article) error

	// CategoryGet 获取分类
	CategoryGet(*Category) error

	// CategoryNew 创建或修改分类
	CategoryNew(*Category) error

	// CategoryDel 删除文章 必须指定 Category.ID
	CategoryDel(*Category) error

	// NavbarNew 创建或修改导航
	NavbarNew(*Navbar) error

	// TagNew 创建或修改标签
	TagNew(*Tag) error

	// TagGet 获取tag
	TagGet(*Tag) error

	// TagDel 删除标签
	TagDel(*Tag) error
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

type ProgramBaseInfo struct {
	HomeURL       string `json:"home_url"`       // 主站 http://blog.isolezvoscombles.com/
	BackstagePath string `json:"backstage_path"` // 后台路径 zb_system/
	LoginPath     string `json:"login_path"`     // 登录路径 cmd.php?act=verify
}

// LoginFunc 登录
type LoginFunc func(u, p string, info ProgramBaseInfo) (ProgramAPI, error)

var programList = make(map[string]LoginFunc)
var plMutex = &sync.Mutex{}

func RegisterProgram(name string, f LoginFunc) {
	plMutex.Lock()
	defer plMutex.Unlock()
	programList[name] = f
}

func GetProgram(name string) LoginFunc {
	plMutex.Lock()
	defer plMutex.Unlock()
	if _, ok := programList[name]; ok {
		return programList[name]
	}
	return nil
}
