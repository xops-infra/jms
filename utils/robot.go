package utils

// https://open.dingtalk.com/document/orgapp/custom-robots-send-group-messages
import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type RobotClient struct {
	requestBuilder requestBuilder
}

func NewRobotClient() *RobotClient {
	return &RobotClient{
		requestBuilder: newRequestBuilder(),
	}
}

type TextBody struct {
	Content string `json:"content"`
}

type AtBody struct {
	IsAtAll   bool     `json:"isAtAll"`
	AtMobiles []string `json:"atMobiles"`
	AtUserIDS []string `json:"atUserIds"`
}

type LinkBody struct {
	MessageUrl string `json:"messageUrl"`
	Title      string `json:"title"`
	PicUrl     string `json:"picUrl"`
	Text       string `json:"text"`
}

type MarkDownBody struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type BtnBody struct {
	ActionURL string `json:"actionURL"`
	Title     string `json:"title"`
}

type ActionCardBody struct {
	HideAvatar     string    `json:"hideAvatar"`
	BtnOrientation string    `json:"btnOrientation"`
	Single         string    `json:"singleURL"`
	SingleTitle    string    `json:"singleTitle"`
	Title          string    `json:"title"`
	Text           string    `json:"text"`
	Btns           []BtnBody `json:"btns"`
}

type FeedCard struct {
	Links []LinkBody `json:"links"`
}

type MessageContent struct {
	MsgType    string         `json:"msgtype"`
	Text       TextBody       `json:"text,omitempty"`
	At         AtBody         `json:"at,omitempty"`
	Link       LinkBody       `json:"link,omitempty"`
	MarkDown   MarkDownBody   `json:"markdown,omitempty"`
	ActionCard ActionCardBody `json:"actionCard,omitempty"`
}

type SendMessageRequest struct {
	AccessToken    string         `json:"access_token"`
	Sign           string         `json:"sign"`
	MessageContent MessageContent `json:"message_content"`
}

func (c *RobotClient) SendMessage(ctx context.Context, req *SendMessageRequest) error {
	var (
		resp     LowApiError
		robotUrl string
	)

	if req.Sign == "" {
		robotUrl = fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", req.AccessToken)
	} else {
		timestamp := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
		toSign := timestamp + "\n" + req.Sign
		//使用HMAC-SHA256算法，以机器人的Secret为key，对待签名的字符串进行加密
		hmacRes := hmac.New(sha256.New, []byte(req.Sign))
		hmacRes.Write([]byte(toSign))

		//将计算出的签名，使用Base64进行编码
		signature := base64.StdEncoding.EncodeToString(hmacRes.Sum(nil))
		//UrlEncode
		urlEncodedSignature := url.QueryEscape(signature)
		robotUrl = fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s&timestamp=%s&sign=%s", req.AccessToken, timestamp, urlEncodedSignature)
	}
	build, err := c.requestBuilder.build(context.Background(), http.MethodPost, robotUrl, req.MessageContent)
	if err != nil {
		return err
	}
	err = c.requestBuilder.sendRequest(build, &resp)
	if err != nil {
		return err
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("%s", resp.ErrMsg)
	}
	return nil
}
