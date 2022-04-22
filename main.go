package main

import (
	"github.com/cgghui/bt_site_cluster/bt"
	"github.com/cgghui/bt_site_cluster/kernel"
	"github.com/cgghui/bt_site_cluster_program_api/core"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	core.Start()

	var (
		option []bt.Option
		err    error
	)
	if err = kernel.LoadBtPanelConfig(&option); err != nil {
		panic(err)
	}

	SiteList := make([]core.SiteConfig, 0)
	for j, opt := range option {
		var siteList []core.SiteConfig
		if err = kernel.LoadSiteConfig(opt.GetAddress(), &siteList); err != nil {
			log.Printf("加载站点文件失败，Error: %v", err)
			continue
		}
		for i := range siteList {
			siteList[i].BtO = &option[j]
		}
		SiteList = append(SiteList, siteList...)
	}
	for i, site := range SiteList {
		if !site.Open {
			continue
		}
		core.SiteCollectChannel <- &SiteList[i]
	}
	WaitQuitSignal()
	log.Println("Byte.")
}

func WaitQuitSignal() {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
