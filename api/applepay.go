package api

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/golang-jwt/jwt/v4"
	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const (
	applePayTestURL = "https://sandbox.itunes.apple.com/verifyReceipt"
	applePayURL     = "https://buy.itunes.apple.com/verifyReceipt"
)

type applePayArgs struct {
	Uid           int
	ShopId        int
	Receipt       string
	ClientOrderId int
}

type applePayReceipt struct {
	ProductId             string `json:"product_id"`
	TransactionId         string `json:"transaction_id"`
	OriginalTransactionId string `json:"original_transaction_id"`
	ExpiresDate           string `json:"expires_date"`
	BundleId              string `json:"bid"`
}

type applePayResponse struct {
	Status             int              `json:"status"`
	AutoRenewProductId string           `json:"auto_renew_product_id"`
	Receipt            *applePayReceipt `json:"receipt"`
	LatestReceiptInfo  *applePayReceipt `json:"latest_receipt_info"`
}

func addAppleOrder(c *Context, data any) (any, error) {
	args := data.(*applePayArgs)
	response, err := sendReceipt(applePayURL, args.Receipt)
	if err != nil {
		return nil, err
	}
	/*
	   {
	       "receipt":{
	           "original_purchase_date_pst":"2021-12-30 01:53:03 America\/Los_Angeles",
	           "purchase_date_ms":"1640857983393",
	           "unique_identifier":"00008101-0002492C3E92001E",
	           "original_transaction_id":"1000000940639443",
	           "bvrs":"1.0",
	           "transaction_id":"1000000940639443",
	           "quantity":"1",
	           "in_app_ownership_type":"PURCHASED",
	           "unique_vendor_identifier":"41B3AD9D-E222-4B92-B278-66D81C05D074",
	           "item_id":"1601788333",
	           "original_purchase_date":"2021-12-30 09:53:03 Etc\/GMT",
	           "is_in_intro_offer_period":"false",
	           "product_id":"bingo.town.free.wild.journey.99",
	           "purchase_date":"2021-12-30 09:53:03 Etc\/GMT",
	           "is_trial_period":"false",
	           "purchase_date_pst":"2021-12-30 01:53:03 America\/Los_Angeles",
	           "bid":"bingo.town.free.wild.journey",
	           "original_purchase_date_ms":"1640857983393"
	       },
	       "status":0
	   }
	*/
	orderStatus := dao.PayOrderFinish
	if response.Status == 21007 {
		orderStatus = dao.PayOrderTest
		response, err = sendReceipt(applePayTestURL, args.Receipt)
		if err != nil {
			return nil, err
		}
	}

	if response.Status != 0 {
		return nil, errors.New("apple pay error code:  " + strconv.Itoa(response.Status))
	}

	itemId := args.ShopId
	matchReceipt := response.Receipt
	if itemId == 0 {
		matchRows := config.FilterRows("Shop", "AppleShopId", matchReceipt.ProductId)
		for _, rowId := range matchRows {
			config.Scan("Shop", rowId, "ID", &itemId)
		}

	}
	configShopId, _ := config.String("Shop", itemId, "AppleShopId")
	if configShopId != matchReceipt.ProductId {
		return nil, errors.New("shopId not match receipt-data product_id")
	}

	price, _ := config.Float("Shop", itemId, "ShopsPrice")
	// 付费订阅
	if response.AutoRenewProductId != "" {
		if response.LatestReceiptInfo == nil {
			return nil, errors.New("empty latest_receipt_info")
		}
		matchReceipt = response.LatestReceiptInfo

		expireMillis, _ := strconv.ParseInt(matchReceipt.ExpiresDate, 10, 64)
		subOrder := &internal.SimpleSubscriptionPurchase{
			ExpiryTimeMillis: expireMillis,
			DevelopPayload:   fmt.Sprintf("%d:%d:%d", args.Uid, args.ShopId, internal.GetClientEnv()),
			OrderId:          matchReceipt.TransactionId,
			SubOrderId:       matchReceipt.OriginalTransactionId,
			IsTest:           orderStatus == dao.PayOrderTest,
			PaySDK:           "apple",
		}
		notifyPurchaseSubscription(matchReceipt.BundleId, matchReceipt.ProductId, "", subOrder)
		return cmd.M{"code": 0, "ClientOrderId": args.ClientOrderId}, nil
	}

	// 增加订单
	order := &dao.PayOrder{
		OrderId:          matchReceipt.TransactionId,
		Result:           int32(orderStatus),
		ExchangeCurrency: "USD",
		ExchangePrice:    price,
		PaySDK:           "apple",
		ItemId:           itemId,
		BuyUid:           args.Uid,
	}
	if err := addPayOrder(order); err != nil {
		return nil, err
	}
	return cmd.M{"code": 0, "ClientOrderId": args.ClientOrderId}, nil
}

func notifyAppleSubscription(ctx *Context) {
	rawData, _ := ctx.GetRawData()
	log.Debugf("notify apple subscription %s", rawData)

	err := handleAppleSubscription(rawData)
	response := cmd.M{"code": 0, "msg": "ok"}
	if err != nil {
		response = cmd.M{"code": 1, "msg": err.Error()}
	}
	ctx.JSON(http.StatusOK, response)
}

func getClaimValue(claim jwt.MapClaims, key string) string {
	if v, ok := claim[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func handleAppleSubscription(rawData []byte) error {
	var response subscriptionResponseV2
	if err := json.Unmarshal([]byte(rawData), &response); err != nil {
		return err
	}

	token, err := jwt.Parse(response.SignedPayload, checkToken)
	if err != nil {
		return err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return errors.New("invalid token")
	}
	data := claims["data"].(map[string]any)
	signedTransactionToken, err := jwt.Parse(data["signedTransactionInfo"].(string), checkToken)
	if err != nil {
		return err
	}
	if !(ok && signedTransactionToken.Valid) {
		return errors.New("invalid token")
	}
	signedTransactionClaims := signedTransactionToken.Claims.(jwt.MapClaims)
	/*signedRenewalToken, err := jwt.Parse(data["signedRenewalInfo"].(string), checkToken)
	if err != nil {
		return err
	}*/

	originOrderId := getClaimValue(signedTransactionClaims, "originalTransactionId")
	originOrder, err := dao.QueryPurchaseSubscriptionOrder(originOrderId)
	if err != nil {
		return err
	}

	shopId := 0
	shopRows := config.FilterRows("Shop", "AppleShopId", originOrder.ProductId)
	for _, rowId := range shopRows {
		config.Scan("Shop", rowId, "Id", &shopId)
	}
	if shopId == 0 {
		return errors.New("invalid shop item: " + originOrder.ProductId)
	}

	// {uid}:{item_id}:{env}
	bundleId := getClaimValue(signedTransactionClaims, "bundleId")
	expireDate, _ := strconv.ParseFloat(getClaimValue(signedTransactionClaims, "expiresDate"), 10)
	orderId := getClaimValue(signedTransactionClaims, "transactionId")
	notifyPurchaseSubscription(bundleId, originOrder.ProductId, "", &internal.SimpleSubscriptionPurchase{
		ExpiryTimeMillis: int64(expireDate),
		DevelopPayload:   fmt.Sprintf("%d:%d:%d", originOrder.Uid, shopId, internal.GetClientEnv()),
		OrderId:          orderId,
		SubOrderId:       originOrderId,
		IsTest:           data["environment"] == "Sandbox",
		PaySDK:           "apple",
	})
	return nil
}

func sendReceipt(url, receipt string) (*applePayResponse, error) {
	buf, err := json.Marshal(cmd.M{
		"receipt-data": receipt,
		"password":     internal.Config().ApplePayPassword,
	})
	if err != nil {
		return nil, err
	}
	body, err := internal.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}
	var response applePayResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func checkToken(token *jwt.Token) (any, error) {
	// header alg: ES256
	if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
		return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
	}
	// header x5c: ["A","B","C"]
	x5c, ok := token.Header["x5c"]
	if !ok {
		return nil, errors.New("header x5c need set")
	}

	x5cArr, ok := x5c.([]any)
	if !ok {
		return nil, errors.New("header x5c is array")
	}

	var x5cArrStr []string
	for _, v := range x5cArr {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("header x5c is string array")
		}
		x5cArrStr = append(x5cArrStr, s)
	}
	// 校验证书有效性，并返回公钥
	publicKey, err := checkCertificates(x5cArrStr)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}

type subscriptionResponseV2 struct {
	SignedPayload string `json:"signedPayload"`
}

func checkCertificates(x5cField []string) (any, error) {
	var pems []string
	for _, x5c := range x5cField {
		pemData := "-----BEGIN CERTIFICATE-----\n"
		for i := 0; i < len(x5c); i += 64 {
			end := i + 64
			if end > len(x5c) {
				end = len(x5c)
			}
			pemData += x5c[i:end] + "\n"
		}
		pemData += "-----END CERTIFICATE-----"
		pems = append(pems, pemData)
	}

	var certs []*x509.Certificate
	// https://www.apple.com/certificateauthority/AppleRootCA-G3.cer
	// openssl x509 -inform der -in AppleRootCA-G3.cer -out AppleRootCA-G3.pem
	rootPem, err := os.ReadFile("configs/AppleRootCA-G3.pem")
	if err != nil {
		return nil, err
	}
	pems = append(pems, string(rootPem))

	for _, pemData := range pems {
		// Parse PEM block
		var block *pem.Block
		if block, _ = pem.Decode([]byte(pemData)); block == nil {
			return nil, errors.New("invalid pem format")
		}

		// Parse the key
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, errors.New("invalid x5c")
	}
	// 校验证书链
	for i := 0; i+1 < len(certs); i++ {
		if err := certs[i].CheckSignatureFrom(certs[i+1]); err != nil {
			return nil, err
		}
	}

	/*publicKey, err := jwt.ParseECPublicKeyFromPEM([]byte(cer))
	if err != nil {
		return err
	}*/
	return certs[0].PublicKey, nil
}

// 订阅返回
// {"expiration_intent":"1", "auto_renew_product_id":"bingo_apple_goldmembership1", "is_in_billing_retry_period":"1", "latest_expired_receipt_info":{"original_purchase_date_pst":"2022-01-05 02:30:16 America/Los_Angeles", "quantity":"1", "unique_vendor_identifier":"E8EB1228-5DB3-4CDE-887D-87E9DB483D00", "bvrs":"1.0", "expires_date_formatted":"2022-01-06 11:38:46 Etc/GMT", "is_in_intro_offer_period":"false", "purchase_date_ms":"1641468946000", "expires_date_formatted_pst":"2022-01-06 03:38:46 America/Los_Angeles", "is_trial_period":"false", "item_id":"1602489001", "unique_identifier":"00008101-0002492C3E92001E", "original_transaction_id":"1000000943022077", "subscription_group_identifier":"20913639", "app_item_id":"1570373069", "transaction_id":"1000000943901585", "in_app_ownership_type":"PURCHASED", "web_order_line_item_id":"1000000070734104", "purchase_date":"2022-01-06 11:35:46 Etc/GMT", "product_id":"bingo_apple_goldmembership1", "expires_date":"1641469126000", "original_purchase_date":"2022-01-05 10:30:16 Etc/GMT", "purchase_date_pst":"2022-01-06 03:35:46 America/Los_Angeles", "bid":"bingo.town.free.wild.journey", "original_purchase_date_ms":"1641378616000"}, "receipt":{"original_purchase_date_pst":"2022-01-05 02:30:16 America/Los_Angeles", "quantity":"1", "unique_vendor_identifier":"44A6C261-0B21-421E-9BB0-724CF734DAD5", "bvrs":"1.0", "expires_date_formatted":"2022-01-06 10:59:55 Etc/GMT", "is_in_intro_offer_period":"false", "purchase_date_ms":"1641466615000", "expires_date_formatted_pst":"2022-01-06 02:59:55 America/Los_Angeles", "is_trial_period":"false", "item_id":"1602489001", "unique_identifier":"00008101-0002492C3E92001E", "original_transaction_id":"1000000943022077", "subscription_group_identifier":"20913639", "transaction_id":"1000000943871220", "in_app_ownership_type":"PURCHASED", "web_order_line_item_id":"1000000070675204", "purchase_date":"2022-01-06 10:56:55 Etc/GMT", "product_id":"bingo_apple_goldmembership1", "expires_date":"1641466795000", "original_purchase_date":"2022-01-05 10:30:16 Etc/GMT", "purchase_date_pst":"2022-01-06 02:56:55 America/Los_Angeles", "bid":"bingo.town.free.wild.journey", "original_purchase_date_ms":"1641378616000"}, "auto_renew_status":0, "status":21006}
