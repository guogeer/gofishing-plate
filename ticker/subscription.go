// 谷歌订阅
package ticker

import (
	"database/sql"
	"time"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

// 向游戏内的玩家发送消息
func SendMsg(uid int, msgId string, body any) error {
	user, err := dao.GetRegUserInfo(uid)
	log.Debugf("sendMsg player %d msgId %s ServerName %s error %v", uid, msgId, user.ServerName, err)
	if err != nil {
		return err
	}

	if user.ServerName == "" {
		user.ServerName = "hall"
	}

	cmd.Route(user.ServerName, msgId, body)
	return nil
}

// 延长订阅期。可用于补偿或订阅体验
func batchDeferPurchaseSubscription() {
	for i := 0; i < 99; i++ {
		order, err := dao.DeferPurchaseSubscriptionOrder()
		if err != nil {
			if err != sql.ErrNoRows {
				log.Errorf("dao.DeferPurchaseSubscriptionOrder error: %v", err)
			}
			return
		}
		d := time.Duration(order.DeferDays) * 24 * time.Hour
		expiryTimeMillis, err := internal.DeferPurchaseSubscription(order.PackageName, order.ProductId, order.PurchaseToken, order.ExpireMillis+d.Milliseconds(), order.ExpireMillis)
		if err != nil {
			log.Errorf("purchasesSubscriptionsService defer error: %v", err)
			return
		}

		dao.UpdatePurchaseSubscriptionOrderExpiryTime(order.Id, expiryTimeMillis)
		log.Infof("defer purchase subscription order id %d NewExpiryTimeMillis %d", order.Id, expiryTimeMillis)

		// 通知到游戏内
		SendMsg(order.UId, "FUNC_UpdatePurchaseScription", cmd.M{
			"UId":      order.UId,
			"ExpireTs": expiryTimeMillis / 1000,
		})
	}
}

func handlePurchaseSubscription() {
	batchDeferPurchaseSubscription()
}
