package api

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type clientVersionArgs struct {
	ChanId  string
	Version string
	Uid     int
	OpenId  string
	Bundles []*dao.ClientBundle
}

func matchClientVersion(chanId, version, clientIP, openId string) *dao.ClientVersion {
	// 所有相同的匹配渠道号的版本
	clientVersions, _ := dao.QueryAllClientVersion()
	// 按照版本号排序
	sort.Slice(clientVersions, func(i, j int) bool {
		return compareClientVersion(clientVersions[i].Version, clientVersions[j].Version) <= 0
	})

	// 过滤白名单
	var matchVersions []*dao.ClientVersion
	for _, clientVersion := range clientVersions {
		if clientVersion.ChanId == chanId {
			ret := compareClientVersion(version, clientVersion.Version)
			// 白名单
			if clientVersion.Whitelist != "" {
				if !((openId != "" && strings.Contains(clientVersion.Whitelist, openId)) ||
					strings.Contains(clientVersion.Whitelist, clientIP)) {
					continue
				}
			}
			// 预发布
			if clientVersion.IsPrerelease && ret != 0 {
				continue
			}
			if !clientVersion.IsPrerelease && ret >= 0 {
				continue
			}
			matchVersions = append(matchVersions, clientVersion)
		}
	}

	cv := &dao.ClientVersion{
		Version: version,
		ChanId:  chanId,
	}
	if len(matchVersions) > 0 {
		cv = matchVersions[0]
	}
	if cv.Type == "" {
		cv.Type = "newest"
	}
	return cv
}

var clientVersionRe = regexp.MustCompile(`[0-9]+`)

// 版本格式：1.3.7.100_r 1.3.7.100 1.3.7.100_d
func compareClientVersion(v1, v2 string) int {
	nums := clientVersionRe.FindAllString(v1, 4)
	nums = append(nums, "0", "0", "0", "0")
	v1 = fmt.Sprintf("%08s%08s%08s%08s", nums[0], nums[1], nums[2], nums[3])

	nums = clientVersionRe.FindAllString(v2, 4)
	nums = append(nums, "0", "0", "0", "0")
	v2 = fmt.Sprintf("%08s%08s%08s%08s", nums[0], nums[1], nums[2], nums[3])
	if v1 > v2 {
		return 1
	}
	if v1 < v2 {
		return -1
	}
	return 0
}

func matchClientBundles(uid int, ip string, queryBundles []*dao.ClientBundle) ([]*dao.ClientBundle, error) {
	dbBundles, err := dao.QueryClientBundle()
	if err != nil {
		return nil, err
	}

	var matchBundles []*dao.ClientBundle
	for _, queryBundle := range queryBundles {
		matchBundle := &dao.ClientBundle{Version: queryBundle.Version, BundleName: queryBundle.BundleName}
		for _, dbBundle := range dbBundles {
			if !strings.Contains(","+dbBundle.BundleName+",", queryBundle.BundleName) {
				continue
			}
			if compareClientVersion(matchBundle.Version, dbBundle.Version) >= 0 {
				continue
			}
			if dbBundle.AllowList != "" {
				allowList := "," + dbBundle.AllowList + ","
				if !strings.Contains(allowList, ip) && !strings.Contains(allowList, strconv.Itoa(uid)) {
					continue
				}
			}
			matchBundle = &dao.ClientBundle{Version: dbBundle.Version, BundleName: queryBundle.BundleName, Url: dbBundle.Url}
		}
		matchBundles = append(matchBundles, matchBundle)
	}
	return matchBundles, nil
}

type clientVersionResponse struct {
	*dao.ClientVersion

	TablesUrl string
	Bundles   []*dao.ClientBundle
	Maintain  *clientMaintain `json:",omitempty"`
	LoginAddr string          `json:",omitempty"` // 登陆服地址。empiregame2020.com:8080
}

type clientMaintain struct {
	StartTs int64
	EndTs   int64
	Content string
}

func checkMaintain(uid int, ip string) (*clientMaintain, error) {
	maintain, err := dao.QueryMaintain()
	if err != nil {
		return nil, err
	}
	startTime, _ := config.ParseTime(maintain.StartTime)
	endTime, _ := config.ParseTime(maintain.EndTime)
	if time.Now().After(endTime) {
		return nil, nil
	}

	if maintain.AllowList != "" {
		allowList := "," + maintain.AllowList + ","
		if strings.Contains(allowList, ip) || strings.Contains(allowList, strconv.Itoa(uid)) {
			return nil, nil
		}
	}
	return &clientMaintain{
		StartTs: startTime.Unix(),
		EndTs:   endTime.Unix(),
		Content: maintain.Content,
	}, nil
}

// 检测客户端版本
func checkClientVersionAndConfig(c *Context, in any) (any, error) {
	args := in.(*clientVersionArgs)
	log.Debugf("checkClientVersionAndConfig,%v,%v,%v", args.ChanId, args.Version, args.Uid)
	clientVersion := matchClientVersion(args.ChanId, args.Version, c.ClientIP(), args.OpenId)
	tablesUrl := getNewestConfigTableUrl()
	bundles, err := matchClientBundles(args.Uid, c.ClientIP(), args.Bundles)
	if err != nil {
		return nil, err
	}
	maintain, err := checkMaintain(args.Uid, c.ClientIP())
	if err != nil {
		return nil, err
	}

	return &clientVersionResponse{
		ClientVersion: clientVersion,
		TablesUrl:     tablesUrl,
		Bundles:       bundles,
		Maintain:      maintain,
		LoginAddr:     internal.Config().Server("login").Addr,
	}, nil
}
