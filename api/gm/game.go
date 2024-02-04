package gm

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"gofishing-plate/api"
	"gofishing-plate/dao"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type onlineReq struct {
	SubId   int    `json:"subId"`
	Curdate string `json:"day"`
}

// 查询在线人数 /api/admin/online
func queryOnline(c *Context, req any) (any, error) {
	args := req.(*onlineReq)
	online, _ := dao.QueryOnline(args.SubId, args.Curdate)

	games := map[int]string{}
	for _, rowId := range config.Rows("Room") {
		var subId int
		var roomName string
		config.Scan("Room", rowId, "RoomID,RoomName", &subId, &roomName)
		games[subId] = roomName
	}

	const roomPointNum = 6 * 24
	curdate, _ := config.ParseTime(args.Curdate)
	onlineSet := map[int][]dao.RoomOnline{}
	onlineSet[0] = make([]dao.RoomOnline, roomPointNum)
	for _, point := range online {
		if _, ok := onlineSet[point.SubId]; !ok && point.Online > 0 {
			onlineSet[point.SubId] = make([]dao.RoomOnline, roomPointNum)
		}
	}

	for subId := range onlineSet {
		for k := range onlineSet[subId] {
			roomName := "总人数"
			if subId != 0 {
				roomName = games[subId]
			}
			onlineSet[subId][k] = dao.RoomOnline{
				RoomName: roomName,
				Time:     curdate.Add(time.Duration(k) * 600 * time.Second).Format("15:04"),
			}
		}
	}

	// 10分钟一个点
	for _, point := range online {
		if _, ok := onlineSet[point.SubId]; !ok {
			continue
		}
		point.RoomName = games[point.SubId]

		createTime, _ := config.ParseTime(point.CreateTime)
		secs := int(createTime.Sub(curdate).Seconds()) / 600 * 600

		if onlineSet[point.SubId][secs/600].Online == 0 {
			onlineSet[0][secs/600].Online += point.Online
			onlineSet[point.SubId][secs/600].Online = point.Online
		}
	}
	var onlinePoints []dao.RoomOnline
	for _, roomPoints := range onlineSet {
		onlinePoints = append(onlinePoints, roomPoints...)
	}
	sort.Slice(onlinePoints, func(i, j int) bool {
		return onlinePoints[i].Time < onlinePoints[j].Time
	})

	return M{
		"data":  onlinePoints,
		"games": games,
	}, nil
}

type itemLogReq struct {
	Uid       int      `json:"uid"`
	ItemId    int      `json:"itemId"`
	Way       string   `json:"way"`
	TimeRange []string `json:"deadline"`
}

// 查询物品日志 /api/admin/itemLog
func queryItemLog(c *Context, req any) (any, error) {
	args := req.(*itemLogReq)
	itemLogs, _ := dao.QueryItemLogs(args.Uid, args.ItemId, args.Way, args.TimeRange, 1000)

	itemWayMap := map[string]string{}
	itemWayList := dao.GetConfigTableItemLog()
	for _, itemWay := range itemWayList {
		itemWay.Way = "sys." + itemWay.Way
		itemWayMap[itemWay.Way] = itemWay.Name
	}

	itemMap := map[int]*dao.ItemRow{}
	itemList := dao.GetConfigTableItem()
	for _, item := range itemList {
		itemMap[item.ShopID] = item
	}

	for _, itemLog := range itemLogs {
		if item, ok := itemMap[itemLog.ItemId]; ok {
			itemLog.ItemName = item.ShopTitle
		}
		if name, ok := itemWayMap[itemLog.Way]; ok {
			itemLog.Way = name
		}
	}

	return M{
		"data":  itemLogs,
		"ways":  itemWayList,
		"items": itemList,
	}, nil
}

type payOrderReq struct {
	Uid       int
	OrderId   string
	TimeRange []string
	Result    []string

	Current  int
	PageSize int
}

// 查询物品日志 /api/admin/payOrder
func queryPayOrder(c *Context, req any) (any, error) {
	args := req.(*payOrderReq)
	payOrders, total, sum, _ := dao.QueryPayOrder(args.OrderId, args.Uid, args.Result, args.TimeRange, args.Current, args.PageSize)

	return M{
		"data":  payOrders,
		"total": total,
		"sum":   sum,
	}, nil
}

type configTableReq struct {
	Name   string
	Method string
	Table  [][]string
}

// 查询/更新配置表 /api/admin/configTable
func handleConfigTable(c *Context, req any) (any, error) {
	args := req.(*configTableReq)
	if args.Method == "update" {
		var content string
		for _, row := range args.Table {
			for k := range row {
				row[k] = strings.ReplaceAll(row[k], "\t", "    ")

			}
			if content != "" {
				content += "\n"
			}
			content += strings.Join(row, "\t")
		}

		err := dao.UpdateConfigTable(args.Name, content)
		if err != nil {
			return nil, err
		}

		log.Debugf("update config table %s success", args.Name)
		cmd.Forward("*", "FUNC_EffectConfigTable", cmd.M{"Tables": []string{args.Name}})
		return M{"Code": 0, "Msg": "ok"}, nil
	}
	table, err := dao.QueryConfigTable(args.Name)
	if err != nil {
		return nil, err
	}

	return M{
		"Table": table.Table,
	}, nil
}

type regUserReq struct {
	Uid       int
	OpenId    string
	ChanId    string
	TimeRange []string
	Method    string
	Current   int
	PageSize  int
}

// 注册玩家 /api/admin/regUser
func handleRegUser(c *Context, req any) (any, error) {
	args := req.(*regUserReq)
	switch args.Method {
	case "query":
		user, err := dao.GetRegUserInfo(args.Uid)
		return M{
			"User": user,
		}, err
	case "delete":
		buf, err := cmd.Request("hall", "FUNC_DeleteAccount", cmd.M{"Uid": args.Uid})
		return json.RawMessage(buf), err
	}
	data, total, err := dao.QueryRegUser(args.Uid, args.ChanId, args.OpenId, args.TimeRange, args.Current, args.PageSize)
	if err != nil {
		return nil, err
	}

	return M{
		"data":  data,
		"total": total,
	}, nil
}

type addItemsReq struct {
	Uid   int
	Items []struct {
		Id  int
		Num int64
	}
}

// 发放补偿 /api/admin/addItems
func addItems(c *Context, req any) (any, error) {
	args := req.(*addItemsReq)
	api.SendMsg(args.Uid, "FUNC_AddItems", M{
		"Uid":   args.Uid,
		"Items": args.Items,
		"Way":   "gm_deal",
	})

	return M{
		"Code": 0,
		"Msg":  "ok",
	}, nil
}

type chanReq struct {
	ChanId    string
	DateRange []string
}

// 渠道数据 /api/admin/chan
func queryChan(c *Context, req any) (any, error) {
	args := req.(*chanReq)
	resp, err := dao.QueryChan(args.ChanId, args.DateRange)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type maintainReq struct {
	Action    string
	TimeRange []string
	Content   string
	AllowList string
}

// 停机维护 /api/admin/maintain
func handleMaintain(c *Context, req any) (any, error) {
	args := req.(*maintainReq)
	if args.Action == "update" {
		args.TimeRange = append(args.TimeRange, "", "")
		err := dao.UpdateMaintain(args.TimeRange[0], args.TimeRange[1], args.Content, args.AllowList)
		if err != nil {
			return nil, err
		}
		cmd.Route("hall", "FUNC_UpdateMaintain", cmd.M{})
		return cmd.M{}, nil
	}
	resp, err := dao.QueryMaintain()
	if err != nil {
		return nil, err
	}
	timeRange := []string{}
	if resp.StartTime != "" {
		timeRange = []string{resp.StartTime, resp.EndTime}
	}
	return cmd.M{
		"TimeRange": timeRange,
		"Content":   resp.Content,
		"AllowList": resp.AllowList,
		"IP":        c.ClientIP(),
	}, nil
}

type mailReq struct {
	Action        string
	SendTimeRange []string
	Current       int
	PageSize      int
	RecvUsers     []int

	dao.Mail
}

// 邮件 /api/admin/mail
func handleMail(c *Context, req any) (any, error) {
	args := req.(*mailReq)
	if args.Action == "add" {
		cmd.Route("hall", "FUNC_SendMail", cmd.M{
			"Users": args.RecvUsers,
			"Mail":  args.Mail,
		})
		return cmd.M{}, nil
	}
	mails, total, err := dao.QueryMails(args.RecvId, args.Type, args.SendTimeRange, args.Current, args.PageSize)
	if err != nil {
		return nil, err
	}
	return cmd.M{
		"data":  mails,
		"total": total,
	}, nil
}
