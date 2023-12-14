package dao

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

// gm管理后台账号
type GmAccount struct {
	Id       int
	Username string
	Password string // md5加密后的密码
	Menus    []string
	Comment  string
}

func GetGmAccount(account string) (*GmAccount, error) {
	info := &GmAccount{}
	err := manageDB.QueryRow("select id,account,`password`,menus,`comment` from gm_account where account=?", account).Scan(&info.Id, &info.Username, &info.Password, JSON(&info.Menus), &info.Comment)
	return info, err
}

func QueryGmAccount() ([]*GmAccount, error) {
	var accounts []*GmAccount

	rows, _ := manageDB.Query("select id,`account`,`password`,menus,`comment` from gm_account order by id desc")
	for rows != nil && rows.Next() {
		info := &GmAccount{}
		rows.Scan(&info.Id, &info.Username, &info.Password, JSON(&info.Menus), &info.Comment)
		accounts = append(accounts, info)
	}
	return accounts, nil
}

func UpdateGmAccount(info *GmAccount) error {
	_, err := manageDB.Exec("update gm_account set account=?,menus=?,`comment`=? where id=?",
		info.Username, JSON(info.Menus), info.Comment, info.Id)
	if err != nil {
		return err
	}
	if strings.Index(info.Password, "*") < 0 {
		sum := md5.Sum([]byte(info.Password))
		password := hex.EncodeToString(sum[:])
		_, err = manageDB.Exec("update gm_account set `password`=? where id=?",
			password, info.Id)
	}
	return err
}

func DeleteGmAccount(id int) error {
	_, err := manageDB.Exec("delete from gm_account where id=?", id)
	return err
}

func AddGmAccount(info *GmAccount) (int64, error) {
	sum := md5.Sum([]byte(info.Password))
	password := hex.EncodeToString(sum[:])
	rs, err := manageDB.Exec("insert into gm_account(`account`, `password`, `menus`, `comment`, `create_time`) VALUES (?, md5(?), ?, ?, now())",
		info.Username, password, JSON(info.Menus), info.Comment)
	if err != nil {
		return -1, err
	}
	return rs.LastInsertId()
}
