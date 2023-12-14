package dao

import (
	"math"
	"sort"
	"time"

	"gofishing-plate/internal"
	"gofishing-plate/internal/pb"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

func round2(f float64, n int) float64 {
	multi := math.Pow10(n)
	return math.Round(f*multi) / multi
}

type ChanData struct {
	RegUser        int     // 注册用户数
	ActiveUser     int     // 活跃用户数
	FirstPayUser   int     // 首充用户数
	FirstPayNum    float64 // 首充金额
	PayUser        int     // 付费用户数
	PayNum         float64 // 付费金额
	OldPayUser     int     // 老用户付费人数
	OldPayNum      float64 // 老用户付费金额
	RegPayUser     int     // 注册用户付费数
	RegPayNum      float64 // 新用户付费
	WatchAdNum     int     // 看广告次数
	WatchAdUser    int     // 看广告人数
	PlayRoundCount int     // 局数

	ActiveDay2  int // 2留
	ActiveDay3  int // 3留
	ActiveDay7  int // 7留
	ActiveDay15 int // 15留
	ActiveDay30 int // 30留
}

func (chanData *ChanData) activePtr(dayNum int) *int {
	switch dayNum {
	case 2:
		return &chanData.ActiveDay2
	case 3:
		return &chanData.ActiveDay3
	case 7:
		return &chanData.ActiveDay7
	case 15:
		return &chanData.ActiveDay15
	case 30:
		return &chanData.ActiveDay30
	}
	return new(int)
}

type clientChanData struct {
	*ChanData

	Date       string
	ChanId     string
	OldPayUser int
	OldPayNum  float64

	RegPayPer      float64
	ActiveARPU     float64
	PayARPU        float64
	ActiveDay2Per  float64
	ActiveDay3Per  float64
	ActiveDay7Per  float64
	ActiveDay15Per float64
	ActiveDay30Per float64
}

type ChanResult struct {
	Chans map[string]*ChanData
}

// 生成渠道数据
func GenerateChan(curDate string) {
	curTime, _ := config.ParseTime(curDate)
	nextDate := curTime.Add(24 * time.Hour).Format(internal.ShortDateFmt)
	dayKey := int32(10000*curTime.Year() + 100*int(curTime.Month()) + curTime.Day())
	log.Debugf("query channel data on %s key %d", curDate, dayKey)

	chans := map[string]*ChanData{}
	rs, err := gameDB.Query("select chan_id,count(*) from user_info where create_time>=? and create_time<? group by chan_id", curDate, nextDate)
	if err != nil {
		log.Error("query user info error: ", err)
	}
	for rs != nil && rs.Next() {
		var chanId string
		var regNum int
		rs.Scan(&chanId, &regNum)
		chans[chanId] = &ChanData{RegUser: regNum}
	}

	rs, err = gameDB.Query("select b.bin,u.create_time,u.chan_id from user_info u left join user_bin b on u.uid=b.uid where b.update_time>=? and b.`class`=?", curDate, "stat")
	if err != nil {
		log.Error("query user stat error: ", err)
	}

	for rs.Next() {
		var chanId, regTimeStr string

		statData := &pb.StatBin{}
		err = rs.Scan(PB(statData), &regTimeStr, &chanId)

		regDate := regTimeStr[:len(internal.ShortDateFmt)]
		if _, ok := chans[chanId]; !ok {
			chans[chanId] = &ChanData{}
		}
		chanData := chans[chanId]
		dayStat := &pb.UserDayStat{}
		if stat, ok := statData.Day[dayKey]; ok {
			dayStat = stat
		}
		// 活跃用户数
		if !dayStat.IsEnter {
			continue
		}
		chanData.ActiveUser += 1
		// 首次付费用户数
		if dayStat.RealFirstPayNum > 0 {
			chanData.FirstPayUser += 1
			chanData.FirstPayNum += dayStat.RealFirstPayNum
		}

		// 付费用户数/金额
		if dayStat.RealPayNum > 0 {
			chanData.PayUser += 1
			chanData.PayNum += dayStat.RealPayNum
		}
		// 付费新用户数/金额
		if regDate == curDate && dayStat.RealPayNum > 0 {
			chanData.RegPayUser += 1
			chanData.RegPayNum += dayStat.RealPayNum
		}
		// 广告人数/次数
		if dayStat.WatchAdNum > 0 {
			chanData.WatchAdNum += int(dayStat.WatchAdNum)
			chanData.WatchAdUser += 1
		}
		regTime, _ := config.ParseTime(regDate)
		dayNum := int(curTime.Sub(regTime).Hours()/24) + 1
		// 更新留存
		activePtr := chanData.activePtr(dayNum)
		*activePtr += 1
		chanData.PlayRoundCount += int(dayStat.PlayRoundCount)
		chans[chanId] = chanData
	}

	// 更新留存数据
	// 字段格式：ActiveDay2, ActiveyDay7...
	for _, dayNum := range []int{2, 3, 7, 15, 30} {
		lastDate := curTime.Add(time.Duration(-dayNum+1) * 24 * time.Hour).Format(internal.ShortDateFmt)

		oldRes := &ChanResult{}
		manageDB.QueryRow("select data from report_day_data where `curdate`=? and `name`=?", lastDate, "chan").Scan(JSON(oldRes))
		for id, oldChan := range oldRes.Chans {
			if newChan, ok := chans[id]; ok {
				newActivePtr, oldActivePtr := newChan.activePtr(dayNum), oldChan.activePtr(dayNum)
				*newActivePtr, *oldActivePtr = 0, *newActivePtr
			}
		}
		manageDB.Exec("update report_day_data set data=? where `curdate`=? and `name`=?", JSON(oldRes), lastDate, "chan")
	}

	res := &ChanResult{Chans: chans}
	manageDB.Exec("delete from report_day_data where `curdate`=? and `name`=?", curDate, "chan")
	manageDB.Exec("insert ignore report_day_data(`curdate`,`name`,`data`) values(?,?,?)", curDate, "chan", JSON(res))
}

type ChanResponse struct {
	TotalRegUser          int
	TotalPlayNum          int
	TotalActiveUser       int
	TotalPayUser          int
	TotalFirstPayUser     int
	TotalRegPayUser       int
	TotalRegPayNum        float64
	TotalRegPayPer        float64
	TotalOldPayUser       int
	TotalOldPayNum        float64
	TotalPayNum           float64
	TotalActiveARPU       float64
	TotalPayARPU          float64
	TotalActiveWatchAdPer float64
	TotalActiveWatchAdAvg float64
	TotalWatchAdUser      int
	TotalWatchAdNum       int
	Data                  []clientChanData `json:"data"`
	ChanList              map[string]string
}

// 查询渠道数据
func QueryChan(chanId string, dateRange []string) (*ChanResponse, error) {
	params := []any{"chan"}

	where := " where `name`=? and 1=1"
	if len(dateRange) > 1 {
		where += " and curdate>=? and curdate<=?"
		params = append(params, dateRange[0], dateRange[1])
	}

	rs, err := manageDB.Query("select curdate,data from report_day_data"+where, params...)
	if err != nil {
		return nil, err
	}

	chanSet := map[string]bool{}
	chans := []clientChanData{}
	for rs.Next() {
		var date string
		var res ChanResult
		err = rs.Scan(&date, JSON(&res))
		for id, c := range res.Chans {
			chanSet[id] = true
			if chanId == "" || chanId == id {
				chans = append(chans, clientChanData{
					ChanData: c,
					ChanId:   id,
					Date:     date,
				})
			}
		}
	}

	sort.Slice(chans, func(i, j int) bool {
		if chans[i].Date > chans[j].Date {
			return true
		}
		return chans[i].Date == chans[j].Date && chans[i].ChanId < chans[j].ChanId
	})

	resp := &ChanResponse{Data: chans, ChanList: map[string]string{}}
	for i := range chans {
		c := &chans[i]
		c.OldPayNum = c.PayNum - c.RegPayNum
		c.OldPayUser = c.PayUser - c.RegPayUser

		if c.RegUser > 0 {
			c.RegPayPer = round2(float64(c.RegPayUser)/float64(c.RegUser)*100, 2)

			c.ActiveDay2Per = round2(float64(c.ActiveDay2)/float64(c.RegUser)*100, 2)
			c.ActiveDay3Per = round2(float64(c.ActiveDay3)/float64(c.RegUser)*100, 2)
			c.ActiveDay7Per = round2(float64(c.ActiveDay7)/float64(c.RegUser)*100, 2)
			c.ActiveDay15Per = round2(float64(c.ActiveDay15)/float64(c.RegUser)*100, 2)
			c.ActiveDay30Per = round2(float64(c.ActiveDay30)/float64(c.RegUser)*100, 2)
		}

		if c.ActiveUser > 0 {
			c.ActiveARPU = c.PayNum / float64(c.ActiveUser)
		}
		if c.PayUser > 0 {
			c.PayARPU = c.PayNum / float64(c.PayUser)
		}

		resp.TotalRegUser += c.RegUser
		resp.TotalPlayNum += c.PlayRoundCount
		resp.TotalActiveUser += c.ActiveUser
		resp.TotalPayUser += c.PayUser
		resp.TotalFirstPayUser += c.FirstPayUser
		resp.TotalRegPayUser += c.RegPayUser
		resp.TotalRegPayNum += c.RegPayNum
		resp.TotalOldPayUser += c.OldPayUser
		resp.TotalOldPayNum += c.OldPayNum
		resp.TotalPayNum += c.PayNum
		resp.TotalWatchAdUser += c.WatchAdUser
		resp.TotalWatchAdNum += c.WatchAdNum
	}
	if resp.TotalRegUser > 0 {
		resp.TotalRegPayPer = round2(float64(resp.TotalRegPayUser)/float64(resp.TotalRegUser)*100, 2)
	}
	if resp.TotalActiveUser > 0 {
		resp.TotalActiveARPU = round2(float64(resp.TotalPayNum)/float64(resp.TotalActiveUser), 2)
		resp.TotalActiveWatchAdPer = round2(float64(resp.TotalWatchAdUser)/float64(resp.TotalActiveUser)*100, 2)
		resp.TotalActiveWatchAdAvg = round2(float64(resp.TotalWatchAdNum)/float64(resp.TotalActiveUser), 2)
	}
	if resp.TotalPayUser > 0 {
		resp.TotalPayARPU = round2(float64(resp.TotalPayNum)/float64(resp.TotalPayUser), 2)
	}
	for name := range chanSet {
		resp.ChanList[name] = name
	}
	return resp, err
}
