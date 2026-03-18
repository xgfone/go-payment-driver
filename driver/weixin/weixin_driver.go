// Copyright 2026 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package weixin

import (
	"bytes"
	"cmp"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"github.com/xgfone/go-payment-driver/builder"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-toolkit/jsonx"
	"github.com/xgfone/go-toolkit/random"
	"github.com/xgfone/go-toolkit/runtimex"
	"github.com/xgfone/go-toolkit/unsafex"
)

var defaultCurrency = "CNY"

func registerBuilder(scene string, linktype driver.LinkType, newf func(_Driver) driver.Driver) {
	metadata := driver.NewMetadata(Type, scene).WithLinkType(linktype)
	builder.Register(builder.New[*Config](newf, metadata))
}

func newDriver(c Config, b builder.Builder) (d _Driver, err error) {
	if err = c.init(); err != nil {
		return
	}

	c.Prikey = strings.ReplaceAll(c.Prikey, `\n`, "\n")
	prikey, err := utils.LoadPrivateKey(c.Prikey)
	if err != nil {
		err = fmt.Errorf("fail to load the private key of the weixin merchant: %w", err)
		return
	}

	var verifier auth.Verifier
	opts := make([]core.ClientOption, 0, 1)
	if c.PubKeyId != "" {
		c.PubKey = strings.ReplaceAll(c.PubKey, `\n`, "\n")
		pubkey, _err := utils.LoadPublicKey(c.PubKey)
		if _err != nil {
			err = fmt.Errorf("fail to load the public key of the weixin merchant: %w", err)
			return
		}

		opts = append(opts, option.WithWechatPayPublicKeyAuthCipher(c.Mchid, c.Certsn, prikey, c.PubKeyId, pubkey))
		verifier = verifiers.NewSHA256WithRSAPubkeyVerifier(c.PubKeyId, *pubkey)
	} else {
		opts = append(opts, option.WithWechatPayAutoAuthCipher(c.Mchid, c.Certsn, prikey, c.Apikey))
		verifier = verifiers.NewSHA256WithRSAVerifier(downloader.MgrInstance().GetCertificateVisitor(c.Mchid))
	}

	client, err := core.NewClient(context.Background(), opts...)
	if err != nil {
		err = fmt.Errorf("fail to new a weixin payment client: %w", err)
		return
	}

	handler, err := notify.NewRSANotifyHandler(c.Apikey, verifier)
	if err != nil {
		err = fmt.Errorf("fail to init the callback notify handler: %w", err)
		return
	}

	d = _Driver{config: c, client: client, handler: handler}
	d.metadata = b.Metadata()
	return
}

type _Driver struct {
	config   Config
	client   *core.Client
	handler  *notify.Handler
	metadata driver.Metadata
}

func (d *_Driver) LinkInfo(paylink, currency string) driver.LinkInfo {
	return driver.LinkInfo{PayLink: paylink, Currency: currency}
}

func (d *_Driver) CheckCreateTradeRequest(r *driver.CreateTradeRequest) (err error) {
	if r.Share && r.TradeAmount < 10 {
		return driver.ErrTooSmallTradeAmount
	}

	r.TradeCurrency = cmp.Or(r.TradeCurrency, defaultCurrency)
	if r.TradeCurrency != "CNY" {
		return errors.New("trade currency is not CNY")
	}

	return
}

/// ----------------------------------------------------------------------- ///

func (d *_Driver) Metadata() driver.Metadata {
	return d.metadata
}

func (d *_Driver) ParseTradeCallbackRequest(ctx context.Context, r *http.Request) (info driver.TradeInfo, err error) {
	var trans payments.Transaction
	_, err = d.handler.ParseNotifyRequest(ctx, r, &trans)
	if err == nil {
		info = d.parsePayRequest(&trans)
	}
	return
}

func (d *_Driver) ParseRefundCallbackRequest(ctx context.Context, r *http.Request) (info driver.RefundInfo, err error) {
	var resp refunddomestic.Refund
	_req, err := d.handler.ParseNotifyRequest(ctx, r, &resp)
	if err == nil {
		info.RefundReason = _req.Summary
		info = d.parseRefundRequest(&resp)
	}
	return
}

func (d *_Driver) SendTradeCallbackResponse(_ context.Context, w http.ResponseWriter, err error) {
	d.sendCallbackResponse(w, err)
}

func (d *_Driver) SendRefundCallbackResponse(_ context.Context, w http.ResponseWriter, err error) {
	d.sendCallbackResponse(w, err)
}

func (d *_Driver) sendCallbackResponse(w http.ResponseWriter, err error) {
	if err != nil {
		data := map[string]string{"code": "FAIL", "message": err.Error()}
		w.(interface{ JSON(int, any) }).JSON(500, data)
	} else {
		w.WriteHeader(200)
	}
}

// If the trade has been fully refunded, return ErrTradeRefundedFully.
// If the balance is insufficient, return ErrBalanceInsufficient.
// If it's not allowed to refund the trade, return ErrUnallowed.
func (d *_Driver) RefundTrade(ctx context.Context, r driver.RefundTradeRequest) (info driver.RefundInfo, err error) {
	var rundaccount *refunddomestic.ReqFundsAccount
	switch r.FundAccount {
	case "":
	case "AVAILABLE":
		rundaccount = refunddomestic.REQFUNDSACCOUNT_AVAILABLE.Ptr()
	default:
		err = fmt.Errorf("unsupported FundAccount '%s', only support 'AVAILABLE'", r.FundAccount)
		return
	}

	svc := refunddomestic.RefundsApiService{Client: d.client}
	resp, result, err := svc.Create(ctx, refunddomestic.CreateRequest{
		TransactionId: nil,
		OutTradeNo:    &r.TradeNo,
		OutRefundNo:   &r.RefundNo,
		NotifyUrl:     &r.CallbackUrl,
		Reason:        &r.RefundReason,
		FundsAccount:  rundaccount,

		Amount: &refunddomestic.AmountReq{
			Refund:   &r.RefundAmount,   // 本次退款的总金额
			Total:    &r.TradeAmount,    // 原订单支付的总金额
			Currency: &r.RefundCurrency, // 目前只支持人民币：CNY
		},
	})
	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}
	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			switch {
			case e.Code == "NOT_ENOUGH":
				err = driver.ErrBalanceInsufficient.WithMessage(e.Message)

			case e.Code == "RULE_LIMIT":
				err = driver.ErrUnallowed.WithReasonf("%s: %s", e.Code, e.Message)

			case e.Code == "INVALID_REQUEST" && strings.Contains(e.Message, "已全额退款"):
				err = driver.ErrTradeRefundedFully

			default:
				err = fmt.Errorf("%s: %s", e.Code, e.Message)
			}
		}
		return
	}

	info = d.parseRefundRequest(resp)
	if info.TradeNo == "" {
		info.TradeNo = r.TradeNo
	}
	return
}

func (d *_Driver) QueryRefund(ctx context.Context, query driver.QueryRefundRequest) (info driver.RefundInfo, ok bool, err error) {
	svc := refunddomestic.RefundsApiService{Client: d.client}
	resp, result, err := svc.QueryByOutRefundNo(ctx, refunddomestic.QueryByOutRefundNoRequest{
		OutRefundNo: &query.RefundNo,
	})
	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	switch e := err.(type) {
	case nil:
	case *core.APIError:
		if e.StatusCode == 404 {
			err = nil
		}
		return
	default:
		return
	}

	info = d.parseRefundRequest(resp)
	ok = true
	return
}

func (d *_Driver) parsePayRequest(trans *payments.Transaction) (info driver.TradeInfo) {
	if trans == nil {
		return
	}

	// SUCCESS：支付成功
	// REFUND：转入退款
	// NOTPAY：未支付
	// CLOSED：已关闭
	// REVOKED：已撤销（仅付款码支付会返回）
	// USERPAYING：用户支付中（仅付款码支付会返回）
	// PAYERROR：支付失败（仅付款码支付会返回）
	info.ChannelStatus = runtimex.Indirect(trans.TradeState)
	switch info.ChannelStatus {
	case "REFUND":
		info.IsRefunded = true
		fallthrough

	case "SUCCESS":
		info.TaskStatus = driver.TaskStatusSuccess
		info.TradeNo = runtimex.Indirect(trans.OutTradeNo)
		info.ChannelTradeNo = runtimex.Indirect(trans.TransactionId)

		if trans.SuccessTime != nil {
			info.PaidAt, _ = time.Parse(time.RFC3339, *trans.SuccessTime)
		}

		if trans.Payer != nil {
			info.PayerId = runtimex.Indirect(trans.Payer.Openid)
		}

		if trans.Amount != nil {
			info.TradeAmount = runtimex.Indirect(trans.Amount.Total)
			info.TradeCurrency = runtimex.Indirect(trans.Amount.Currency)

			info.PaidAmount = runtimex.Indirect(trans.Amount.PayerTotal)
			info.PaidCurrency = runtimex.Indirect(trans.Amount.PayerCurrency)
		}

		var data ChannelData
		data.BankType = runtimex.Indirect(trans.BankType)
		if _len := len(trans.PromotionDetail); _len > 0 {
			data.Promotions = make([]Promotion, _len)
			for i := range trans.PromotionDetail {
				pd := &trans.PromotionDetail[i]
				data.Promotions[i] = Promotion{
					Name:     runtimex.Indirect(pd.Name),
					Type:     runtimex.Indirect(pd.Type),
					Scope:    runtimex.Indirect(pd.Scope),
					StockId:  runtimex.Indirect(pd.StockId),
					CouponId: runtimex.Indirect(pd.CouponId),
					Currency: runtimex.Indirect(pd.Currency),
					Amount:   runtimex.Indirect(pd.Amount),

					WechatpayContribute: runtimex.Indirect(pd.WechatpayContribute),
					MerchantContribute:  runtimex.Indirect(pd.MerchantContribute),
					OtherContribute:     runtimex.Indirect(pd.OtherContribute),
				}
			}
		}
		if data.BankType != "" || len(data.Promotions) > 0 {
			info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 24)
		}

	case "USERPAYING", "NOTPAY":
		info.TaskStatus = driver.TaskStatusProcessing

	case "CLOSED", "REVOKED":
		info.TaskStatus = driver.TaskStatusClosed

	case "PAYERROR":
		info.TaskStatus = driver.TaskStatusFailure

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return
}

func (d *_Driver) parseRefundRequest(r *refunddomestic.Refund) (info driver.RefundInfo) {
	if r == nil {
		return
	}

	channelData := ChannelData{UserAccount: runtimex.Indirect(r.UserReceivedAccount)}
	channelDataStr, _ := jsonx.MarshalStringWithCap(channelData, 32)

	info = driver.RefundInfo{
		TradeNo:    runtimex.Indirect(r.OutTradeNo),
		RefundNo:   runtimex.Indirect(r.OutRefundNo),
		RefundedAt: runtimex.Indirect(r.SuccessTime),

		ChannelTradeNo:  runtimex.Indirect(r.TransactionId),
		ChannelRefundNo: runtimex.Indirect(r.RefundId),
		ChannelStatus:   string(runtimex.Indirect(r.Status)),
		ChannelData:     channelDataStr,
	}

	info.TradeAmount = runtimex.Indirect(runtimex.Indirect(r.Amount).Total)
	info.RefundAmount = runtimex.Indirect(runtimex.Indirect(r.Amount).PayerRefund)

	// SUCCESS: 退款成功
	// CLOSED: 退款关闭
	// PROCESSING: 退款处理中
	// ABNORMAL: 退款异常，退款到银行发现用户的卡作废或者冻结了，导致原路退款银行卡失败，
	switch info.ChannelStatus {
	case "SUCCESS":
		info.TaskStatus = driver.TaskStatusSuccess

	case "PROCESSING":
		info.TaskStatus = driver.TaskStatusProcessing

	case "CLOSED":
		info.TaskStatus = driver.TaskStatusClosed

	case "":
		if info.RefundedAt.IsZero() {
			info.TaskStatus = driver.TaskStatusUnknown
		} else {
			info.TaskStatus = driver.TaskStatusSuccess
		}

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return
}

type Signature struct {
	Nonce    string `json:"nonceStr"`
	UnixTime string `json:"timeStamp"`
	SignType string `json:"signType"`
	PaySign  string `json:"paySign"`
}

func (d *_Driver) getSign(prepayid string) (sign Signature, err error) {
	prikeyblock, _ := pem.Decode(unsafex.Bytes(d.config.Prikey))
	if prikeyblock == nil {
		err = errors.New("invalid weixin merchant private key")
		return
	}

	prikey, err := x509.ParsePKCS1PrivateKey(prikeyblock.Bytes)
	if err != nil {
		if key, err2 := x509.ParsePKCS8PrivateKey(prikeyblock.Bytes); err2 != nil {
			err = fmt.Errorf("invalid weixin merchant private key: %w", err)
			return
		} else {
			prikey = key.(*rsa.PrivateKey)
		}
	}

	buf := bytes.NewBuffer(nil)
	buf.Grow(128)

	buf.WriteString(d.config.Appid)
	buf.WriteByte('\n')

	sign.UnixTime = strconv.FormatInt(time.Now().Unix(), 10)
	buf.WriteString(sign.UnixTime)
	buf.WriteByte('\n')

	sign.Nonce = random.String(16, random.AlphaNumCharset)
	buf.WriteString(sign.Nonce)
	buf.WriteByte('\n')

	buf.WriteString("prepay_id=")
	buf.WriteString(prepayid)
	buf.WriteByte('\n')

	hash := sha256.Sum256(buf.Bytes())
	data, err := rsa.SignPKCS1v15(rand.Reader, prikey, crypto.SHA256, hash[:])
	if err != nil {
		err = fmt.Errorf("fail to generate sign by rsa: %w", err)
		return
	}

	sign.PaySign = base64.StdEncoding.EncodeToString(data)
	sign.SignType = "RSA"
	return
}
