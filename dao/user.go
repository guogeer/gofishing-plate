package dao

import (
	"gofishing-plate/internal/pb"
	// "github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/config"
)

type UserInfo struct {
	UId        int
	Nickname   string
	ChanId     string
	Sex        int
	Icon       string
	OS         string
	NetMode    string
	PhoneBrand string
	SubId      int
	IP         string
	ServerName string
	OpenId     string
	CreateTime string
	AllOpenId  []string
	VIP        int
	RoomName   string

	SuperPowers int64       // SP总数
	Stat        *pb.StatBin `json:",omitempty"`
}

func GetRegUserInfo(uid int) (*UserInfo, error) {
	info := &UserInfo{
		UId:  uid,
		Stat: &pb.StatBin{},
	}
	gameDB.QueryRow("select account_info,create_time from user_info where uid=?", uid).Scan(JSON(info), &info.CreateTime)
	rs, _ := gameDB.Query("select open_id from user_plate where uid=?", uid)
	for rs != nil && rs.Next() {
		rs.Scan(&info.OpenId)
		info.AllOpenId = append(info.AllOpenId, info.OpenId)
	}

	gameDB.QueryRow("select bin from user_bin where uid=? and `class`=?", uid, "stat").Scan(PB(info.Stat))
	for _, rowId := range config.Rows("super_power_up") {
		id, _ := config.Int("super_power_up", rowId, "ItemId")
		if itemStat, ok := info.Stat.Items[int32(id)]; ok {
			info.SuperPowers += itemStat.CopyNum
		}
	}
	info.RoomName, _ = config.String("Room", info.SubId, "RoomName")
	return info, nil
}

func QueryRegUser(uid int, chanId, openId string, timeRange []string, current, pageSize int) ([]*UserInfo, int, error) {
	var total int
	var users []*UserInfo
	var params []any

	if openId != "" {
		gameDB.QueryRow("select uid from user_plate where open_id=?", openId).Scan(&uid)
	}

	where := " where 1 = 1"
	if uid > 0 {
		where += " and uid=?"
		params = append(params, uid)
	}
	if chanId != "" {
		where += " and chan_id=?"
		params = append(params, chanId)
	}

	if len(timeRange) > 1 {
		where += " and create_time between ? and ?"
		params = append(params, timeRange[0], timeRange[1])
	}
	gameDB.QueryRow("select count(*) from user_info"+where, params...).Scan(&total)

	limit := " order by uid desc limit ?,?"
	params = append(params, (current-1)*pageSize, pageSize)
	rs, err := gameDB.Query("select uid from user_info"+where+limit, params...)
	if err != nil {
		return users, total, err
	}

	var queryUsers []int
	for rs.Next() {
		var queryUId int
		rs.Scan(&queryUId)
		queryUsers = append(queryUsers, queryUId)
	}
	for _, queryUId := range queryUsers {
		info, _ := GetRegUserInfo(queryUId)
		users = append(users, info)
	}
	return users, total, nil
}
