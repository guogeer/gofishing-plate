package dao

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gofishing-plate/internal"

	"github.com/guogeer/quasar/config"
)

type ItemLog struct {
	Id       int    `json:"id"`
	Uid      int    `json:"uid"`
	ItemId   int    `json:"itemId"`
	ItemName string `json:"itemName"`
	ItemNum  int    `json:"itemNum"`
	Balance  int    `json:"balance"`
	Way      string `json:"way"`
	Guid     string `json:"guid"`
	Params   string `json:"params"`
	Deadline string `json:"deadline"`
}

// 查询物品日志
func QueryItemLogs(uid, itemId int, way string, dateRange []string, limit int) ([]*ItemLog, error) {
	today := time.Now().Format(internal.ShortDateFmt)
	if len(dateRange) < 2 {
		dateRange = []string{
			time.Now().Add(30 * 24 * time.Hour).Format(internal.LongDateFmt),
			time.Now().Format(internal.LongDateFmt),
		}
	}
	endTime, _ := config.ParseTime(dateRange[1])

	var lastStartDate string
	var itemLogs []*ItemLog
	for i := 0; i < 30 && len(itemLogs) < limit; i++ {
		dayStartDate := endTime.Add(-time.Duration(i) * 24 * time.Hour).Format(internal.ShortDateFmt)
		if dayStartDate < dateRange[0] && len(dateRange[0]) > len(internal.ShortDateFmt) {
			dayStartDate = dateRange[0][:len(internal.ShortDateFmt)]
		}
		if lastStartDate == dayStartDate {
			break
		}
		lastStartDate = dayStartDate

		dayEndTime := dayStartDate + " 23:59:59"
		if dayEndTime > dateRange[1] {
			dayEndTime = dateRange[1]
		}

		where := " where 1=1"
		if uid > 0 {
			where = where + " and uid=" + strconv.Itoa(uid)
		}
		if itemId > 0 {
			where = where + " and item_id=" + strconv.Itoa(itemId)
		}
		if way != "" {
			where = where + fmt.Sprintf(" and way='%s'", way)
		}
		where = where + fmt.Sprintf(" and deadline between '%s' and '%s'", dayStartDate, dayEndTime)

		tableName := "item_log_" + strings.ReplaceAll(dayStartDate, "-", "")
		if dayStartDate == today {
			tableName = "item_log"
		}

		limit := fmt.Sprintf(" limit %d", limit-len(itemLogs))
		rs, err := gameDB.Query("select id,uid,item_id,item_num,balance,way,guid,params,deadline from " + tableName + where + " order by id desc " + limit)
		if err != nil {
			continue
		}

		for rs.Next() {
			itemLog := &ItemLog{}
			rs.Scan(&itemLog.Id, &itemLog.Uid, &itemLog.ItemId, &itemLog.ItemNum, &itemLog.Balance, &itemLog.Way, &itemLog.Guid, &itemLog.Params, &itemLog.Deadline)
			itemLogs = append(itemLogs, itemLog)
		}
	}
	return itemLogs, nil
}

type RoomOnline struct {
	Time       string `json:"time"`
	SubId      int    `json:"subId"`
	Online     int    `json:"online"`
	RoomName   string `json:"roomName"`
	CreateTime string `json:"-"`
}

// 查询在线人数
func QueryOnline(subId int, curdate string) ([]RoomOnline, error) {
	rs, err := manageDB.Query("select sub_id,user_num,create_time from game_online where (sub_id=? or ?=0) and create_time between ? and ?", subId, subId, curdate, curdate+" 23:59:59")
	if err != nil {
		return nil, err
	}

	var points []RoomOnline
	for rs.Next() {
		point := RoomOnline{}
		rs.Scan(&point.SubId, &point.Online, &point.CreateTime)
		points = append(points, point)
	}
	return points, err
}

func AddOnline(data []*RoomOnline) error {
	for _, point := range data {
		manageDB.Exec("insert into game_online(sub_id,user_num) values(?,?)", point.SubId, point.Online)
	}
	return nil
}

// 邮件
type Mail struct {
	Id            int64
	Type          int
	SendId        int
	RecvId        int
	Title         string
	Body          string
	Reward        string
	Status        int
	SendTime      string
	ClientVersion string   // 指定版本
	LoginTime     []string // 上次登陆时间
	EffectTime    []string // 有效时间
	RegTime       []string // 用户注册时间
}

// 查询邮件
func QueryMails(recvId, typ int, sendTimeRange []string, current, pageSize int) ([]Mail, int, error) {
	where := " where 1=1"
	params := []any{}
	if recvId > 0 {
		where += " and recv_uid=?"
		params = append(params, recvId)
	}
	where += " and `type`=?"
	params = append(params, typ)
	if len(sendTimeRange) > 1 {
		where += " and send_time between ? and ?"
		params = append(params, sendTimeRange[0], sendTimeRange[1])
	}
	where += " order by id desc limit ?,?"
	params = append(params, (current-1)*pageSize, pageSize)

	rs, err := gameDB.Query("select id,recv_uid,`type`,`data`,`status`,send_time from mail"+where, params...)
	if err != nil {
		return nil, 0, err
	}

	var mails []Mail
	for rs.Next() {
		mail := Mail{}
		rs.Scan(&mail.Id, &mail.RecvId, &mail.Type, JSON(&mail), &mail.Status, &mail.SendTime)
		mails = append(mails, mail)
	}

	var total int
	gameDB.QueryRow("select count(*) from mail").Scan(&total)
	return mails, total, nil
}
