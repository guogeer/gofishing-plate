package dao

import (
// "github.com/guogeer/quasar/log"
)

type ClientBundle struct {
	Id         int `json:",omitempty"`
	Version    string
	BundleName string
	AllowList  string `json:",omitempty"`
	Url        string
	UpdateTime string `json:",omitempty"`
}

type ClientVersion struct {
	clientVersionParams

	Id      int    `json:"id,omitempty"` //服务器唯一id
	Version string `json:",omitempty"`   // 版本 1.12.123.1234_r,1.12.123.1234_d,1.12.123.1234
	ChanId  string `json:",omitempty"`
}

type clientVersionParams struct {
	Reward       string // 奖励 [[1102,1000],[1104,2000]]
	Body         string
	Type         string
	Whitelist    string // 白名单
	AllowIps     string // IP白名单
	Bundles      []ClientBundle
	LastTime     string
	ChangeLog    string
	Url          string
	IsPrerelease bool // 预发布版本
}

// 版本管理
func QueryAllClientVersion() ([]*ClientVersion, error) {
	var versions []*ClientVersion

	rows, _ := manageDB.Query("select id, chan_id,version,json_value,last_time from gm_client_version")
	for rows != nil && rows.Next() {
		v := &ClientVersion{}
		rows.Scan(&v.Id, &v.ChanId, &v.Version, JSON(v), &v.LastTime)
		versions = append(versions, v)
	}
	return versions, nil
}

func UpdateClientVersion(v *ClientVersion) {
	manageDB.Exec("update gm_client_version set chan_id=?,version=?,last_time=now(),json_value=? where id=?",
		v.ChanId, v.Version, JSON(v.clientVersionParams), v.Id)
}

func DeleteClientVersion(id int) {
	manageDB.Exec("DELETE FROM gm_client_version WHERE id = ?", id)
}

func AddClientVersion(v *ClientVersion) error {
	_, err := manageDB.Exec("INSERT INTO gm_client_version (`chan_id`, `version`, `json_value`, `last_time`) VALUES (?, ?, ?,now())",
		v.ChanId, v.Version, JSON(v))
	return err
}

// 分包资源
func QueryClientBundle() ([]*ClientBundle, error) {
	var bundles []*ClientBundle

	rows, _ := manageDB.Query("select id,`bundle_name`,version,allow_list,url,update_time from gm_client_bundle order by id desc")
	for rows != nil && rows.Next() {
		b := &ClientBundle{}
		rows.Scan(&b.Id, &b.BundleName, &b.Version, &b.AllowList, &b.Url, &b.UpdateTime)
		bundles = append(bundles, b)
	}
	return bundles, nil
}

func UpdateClientBundle(bundle *ClientBundle) error {
	_, err := manageDB.Exec("update gm_client_bundle set bundle_name=?,version=?,update_time=now(),allow_list=?,url=? where id=?",
		bundle.BundleName, bundle.Version, bundle.AllowList, bundle.Url, bundle.Id)
	return err
}

func DeleteClientBundle(id int) error {
	_, err := manageDB.Exec("delete from gm_client_bundle where id=?", id)
	return err
}

func AddClientBundle(bundle *ClientBundle) (int64, error) {
	rs, err := manageDB.Exec("insert into gm_client_bundle(`bundle_name`, `version`, `allow_list`, `url`, `update_time`) VALUES (?, ?, ?, ?, now())",
		bundle.BundleName, bundle.Version, bundle.AllowList, bundle.Url)
	if err != nil {
		return -1, err
	}
	return rs.LastInsertId()
}
