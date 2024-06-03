package dao

import (
	"gofishing-plate/internal/pb"
	"strconv"
	"strings"

	// "github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/config"
)

type UserInfo struct {
	Uid        int      `json:"uid,omitempty"`
	Nickname   string   `json:"nickname,omitempty"`
	ChanId     string   `json:"chanId,omitempty"`
	Sex        int      `json:"sex,omitempty"`
	Icon       string   `json:"icon,omitempty"`
	OS         string   `json:"os,omitempty"`
	NetMode    string   `json:"netMode,omitempty"`
	PhoneBrand string   `json:"phoneBrand,omitempty"`
	SubId      int      `json:"subId,omitempty"`
	IP         string   `json:"ip,omitempty"`
	ServerId   string   `json:"serverId,omitempty"`
	OpenId     string   `json:"openId,omitempty"`
	CreateTime string   `json:"createTime,omitempty"`
	AllOpenId  []string `json:"allOpenId,omitempty"`
	VIP        int      `json:"vip,omitempty"`
	RoomName   string   `json:"roomName,omitempty"`

	Stat *pb.StatBin `json:"stat,omitempty"`
}

func GetRegUserInfo(uid int) (*UserInfo, error) {
	info := &UserInfo{
		Uid:  uid,
		Stat: &pb.StatBin{},
	}
	var gameLocation string
	gameDB.QueryRow("select chan_id,server_location,nickname,sex,icon,create_time from user_info where id=?", uid).Scan(
		&info.ChanId, &gameLocation, &info.Nickname, &info.Sex, &info.Icon, &info.CreateTime)

	values := strings.SplitN(gameLocation, ":", 2)
	if len(values) > 1 {
		info.ServerId = values[0]
		info.SubId, _ = strconv.Atoi(values[1])
	}

	rs, _ := gameDB.Query("select open_id from user_plate where uid=?", uid)
	for rs != nil && rs.Next() {
		rs.Scan(&info.OpenId)
		info.AllOpenId = append(info.AllOpenId, info.OpenId)
	}

	gameDB.QueryRow("select bin from user_bin where uid=? and `class`=?", uid, "stat").Scan(PB(info.Stat))
	info.RoomName, _ = config.String("room", info.SubId, "roomName")
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
		where += " and id=?"
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
		var queryUid int
		rs.Scan(&queryUid)
		queryUsers = append(queryUsers, queryUid)
	}
	for _, queryUid := range queryUsers {
		info, _ := GetRegUserInfo(queryUid)
		users = append(users, info)
	}
	return users, total, nil
}
