package dao

import (
	"encoding/json"

	"gofishing-plate/internal/pb"
)

// 查询字典
func QueryDictValue(key string) (string, error) {
	var value string
	manageDB.QueryRow("select `value` from dict where `key`=?", key).Scan(&value)
	return value, nil
}

func UpdateDictValue(key, value string) error {
	manageDB.Exec("insert ignore into dict(`key`,`value`) values(?,?)", key, value)
	manageDB.Exec("update dict set `value`=? where `key`=?", value, key)
	return nil
}

// 停机维护
func QueryMaintain() (*pb.Maintain, error) {
	maintain := &pb.Maintain{}
	value, _ := QueryDictValue("maintain")
	json.Unmarshal([]byte(value), maintain)
	return maintain, nil
}

func UpdateMaintain(startTime, endTime, content, allowList string) error {
	buf, _ := json.Marshal(&pb.Maintain{
		StartTime: startTime,
		EndTime:   endTime,
		Content:   content,
		AllowList: allowList,
	})
	return UpdateDictValue("maintain", string(buf))
}
