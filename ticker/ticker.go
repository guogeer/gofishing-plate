package ticker

import (
	"time"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/util"
)

func tick1d() {
	curTime := time.Now().Add(-24 * time.Hour)
	curDate := curTime.Format(internal.ShortDateFmt)

	dao.GenerateChan(curDate) // 生成渠道数据
}

func tick10m() {
	// handlePurchaseSubscription()
}

func init() {
	startTime, _ := config.ParseTime("2021-01-01 00:10:00")
	util.NewPeriodTimer(tick1d, startTime, 24*time.Hour)
	util.NewPeriodTimer(tick10m, startTime, 10*time.Minute)
}
