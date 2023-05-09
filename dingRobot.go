package dingSend

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"dingSend/common"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type DingRobot struct {
	RobotId            string `gorm:"primaryKey;foreignKey:RobotId" json:"robot_id"` //机器人的token
	Type               string `json:"type"`                                          //机器人类型，1为企业内部机器人，2为自定义webhook机器人
	TypeDetail         string `json:"type_detail"`                                   //具体机器人类型
	ChatBotUserId      string `json:"chat_bot_user_id"`                              //加密的机器人id，该字段无用
	Secret             string `json:"secret"`                                        //如果是自定义成机器人， 则存在此字段
	DingUserID         string `json:"ding_user_id"`                                  // 机器人所属用户id
	UserName           string `json:"user_name"`                                     //机器人所属用户名
	ChatId             string `json:"chat_id"`                                       //机器人所在的群聊chatId
	OpenConversationID string `json:"open_conversation_id"`                          //机器人所在的群聊openConversationID
	Name               string `json:"name"`                                          //机器人的名称
}
type ParamChat struct {
	Token     string   `json:"token"`
	RobotCode string   `json:"robotCode"`
	UserIds   []string `json:"userIds"`
	MsgKey    string   `json:"msgKey"`
	MsgParam  string   `json:"msgParam"`
}
type ParamCronTask struct {
	MsgText     *common.MsgText     `json:"msg_text"`
	MsgLink     *common.MsgLink     `json:"msg_link"`
	MsgMarkDown *common.MsgMarkDown `json:"msg_mark_down"`
	RobotId     string              `json:"robot_id" binding:"required"` //使用机器人的robot_id来确定机器人
}
type ResponseSendMessage struct {
	DingResponseCommon
}
type DingResponseCommon struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

//钉钉机器人单聊
func (t *DingRobot) hmacSha256(stringToSign string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (r *DingRobot) ChatSendMessage(p *ParamChat) error {
	var client *http.Client
	var request *http.Request
	var resp *http.Response
	var body []byte
	URL := "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
	client = &http.Client{Transport: &http.Transport{ //对客户端进行一些配置
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}, Timeout: time.Duration(time.Second * 5)}
	//此处是post请求的请求题，我们先初始化一个对象
	b := struct {
		MsgParam  string   `json:"msgParam"`
		MsgKey    string   `json:"msgKey"`
		RobotCode string   `json:"robotCode"`
		UserIds   []string `json:"userIds"`
	}{MsgParam: fmt.Sprintf("{       \"content\": \"%s\"   }", p.MsgParam),
		MsgKey:    p.MsgKey,
		RobotCode: r.RobotId,
		UserIds:   p.UserIds,
	}
	//然后把结构体对象序列化一下
	bodymarshal, err := json.Marshal(&b)
	if err != nil {
		return nil
	}
	//再处理一下
	reqBody := strings.NewReader(string(bodymarshal))
	//然后就可以放入具体的request中的
	request, err = http.NewRequest(http.MethodPost, URL, reqBody)
	if err != nil {
		return nil
	}

	if err != nil {
		return err
	}
	request.Header.Set("x-acs-dingtalk-access-token", p.Token)
	request.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(request)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body) //把请求到的body转化成byte[]
	if err != nil {
		return nil
	}
	h := struct {
		Code                      string   `json:"code"`
		Message                   string   `json:"message"`
		ProcessQueryKey           string   `json:"processQueryKey"`
		InvalidStaffIdList        []string `json:"invalidStaffIdList"`
		FlowControlledStaffIdList []string `json:"flowControlledStaffIdList"`
	}{}
	//把请求到的结构反序列化到专门接受返回值的对象上面
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil
	}
	if h.Code != "" {
		return errors.New(h.Message)
	}
	// 此处举行具体的逻辑判断，然后返回即可

	return nil
}
func (t *DingRobot) getURL() string {
	url := "https://oapi.dingtalk.com/robot/send?access_token=" + t.RobotId //拼接token路径
	timestamp := time.Now().UnixNano() / 1e6                                //以毫秒为单位
	//formatTimeStr := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, t.Secret)
	sign := t.hmacSha256(stringToSign, t.Secret)
	url = fmt.Sprintf("%s&timestamp=%d&sign=%s", url, timestamp, sign) //把timestamp和sign也拼接在一起
	return url
}
func (t *DingRobot) getURLV2() string {
	url := "https://oapi.dingtalk.com/robot/send?access_token=" + t.RobotId //拼接token路径
	return url
}
func (t *DingRobot) SendMessage(p *ParamCronTask) error {
	b := []byte{}
	//我们需要在文本，链接，markdown三种其中的一个
	if p.MsgText.Msgtype == "text" {
		msg := map[string]interface{}{}
		atMobileStringArr := make([]string, len(p.MsgText.At.AtMobiles))
		for i, atMobile := range p.MsgText.At.AtMobiles {
			atMobileStringArr[i] = atMobile.AtMobile
		}
		atUserIdStringArr := make([]string, len(p.MsgText.At.AtUserIds))
		for i, AtuserId := range p.MsgText.At.AtUserIds {
			atUserIdStringArr[i] = AtuserId.AtUserId
		}
		msg = map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": p.MsgText.Text.Content,
			},
		}
		if p.MsgText.At.IsAtAll {
			msg["at"] = map[string]interface{}{
				"isAtAll": p.MsgText.At.IsAtAll,
			}
		} else {
			msg["at"] = map[string]interface{}{
				"atMobiles": atMobileStringArr, //字符串切片类型
				"atUserIds": atUserIdStringArr,
				"isAtAll":   p.MsgText.At.IsAtAll,
			}
		}
		b, _ = json.Marshal(msg)

	} else if p.MsgLink.Msgtype == "link" {
		//直接序列化
		b, _ = json.Marshal(p.MsgLink)
	} else if p.MsgMarkDown.Msgtype == "markdown" {
		msg := map[string]interface{}{}
		atMobileStringArr := make([]string, len(p.MsgMarkDown.At.AtMobiles))
		for i, atMobile := range p.MsgMarkDown.At.AtMobiles {
			atMobileStringArr[i] = atMobile.AtMobile
		}
		msg = map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": p.MsgMarkDown.MarkDown.Title,
				"text":  p.MsgMarkDown.MarkDown.Text,
			},
		}
		if p.MsgText.At.IsAtAll {
			msg["at"] = map[string]interface{}{
				"isAtAll": p.MsgText.At.IsAtAll,
			}
		} else {
			msg["at"] = map[string]interface{}{
				"atMobiles": atMobileStringArr, //字符串切片类型
				"isAtAll":   p.MsgText.At.IsAtAll,
			}
		}
		b, _ = json.Marshal(msg)
	} else {
		msg := map[string]interface{}{}
		atMobileStringArr := make([]string, len(p.MsgText.At.AtMobiles))
		for i, atMobile := range p.MsgText.At.AtMobiles {
			atMobileStringArr[i] = atMobile.AtMobile
		}
		atUserIdStringArr := make([]string, len(p.MsgText.At.AtUserIds))
		for i, AtuserId := range p.MsgText.At.AtUserIds {
			atUserIdStringArr[i] = AtuserId.AtUserId
		}
		msg = map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": p.MsgText.Text.Content,
			},
		}
		if p.MsgText.At.IsAtAll {
			msg["at"] = map[string]interface{}{
				"isAtAll": p.MsgText.At.IsAtAll,
			}
		} else {
			msg["at"] = map[string]interface{}{
				"atMobiles": atMobileStringArr, //字符串切片类型
				"atUserIds": atUserIdStringArr,
				"isAtAll":   p.MsgText.At.IsAtAll,
			}
		}
		b, _ = json.Marshal(msg)
	}

	var resp *http.Response
	var err error
	if t.Type == "1" || t.Secret == "" {
		resp, err = http.Post(t.getURLV2(), "application/json", bytes.NewBuffer(b))
	} else {
		resp, err = http.Post(t.getURL(), "application/json", bytes.NewBuffer(b))
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	date, err := ioutil.ReadAll(resp.Body)
	r := ResponseSendMessage{}
	err = json.Unmarshal(date, &r)
	if err != nil {
		return err
	}
	if r.Errcode != 0 {
		fmt.Println(r.Errmsg)
		return errors.New(r.Errmsg)
	}

	return nil
}
