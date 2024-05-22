package ticker

import (
	"time"

	"github.com/guogeer/quasar/utils"

	"gofishing-plate/internal"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

func tick12h() {
	generateGoldRank()
}

func generateGoldRank()

func init() {
	// 农场项目的建设度排行榜功能
	log.Infof("config.xml node ProductName:%s", internal.Config().ProductName)
	if internal.Config().ProductName == "farm" {
		startTime, _ := config.ParseTime("2021-01-01 00:00:00")
		utils.NewPeriodTimer(tick12h, startTime, 12*time.Hour)
	}
}
