package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	mathRand "math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// 抓包获取session_token 替换
	token = ""
	// 最大分数 1:1
	maxScore = 11000
	// 活动id
	activeId = 1000005
)

func main() {
	i := 0
	for {
		i++
		fmt.Println("第", i, "次")
		isRun := create()
		if !isRun {
			break
		}
	}
}

type (
	Create struct {
		ActivityId int    `json:"activity_id"`
		GamePkId   string `json:"game_pk_id"`
	}

	CreateResponse struct {
		Data struct {
			ActivityId        int    `json:"activity_id"`
			GameId            string `json:"game_id"`
			GamerHeadUrl      string `json:"gamer_head_url"`
			GamerNickName     string `json:"gamer_nick_name"`
			OpponentHeadUrl   string `json:"opponent_head_url"`
			OpponentNickName  string `json:"opponent_nick_name"`
			OpponentPlayScore int    `json:"opponent_play_score"`
			PlayScript        struct {
				DragonBoat2023PlayScript struct {
					BonusPropSpeedPerSecond int `json:"bonus_prop_speed_per_second"`
					GameDurationTimeMs      int `json:"game_duration_time_ms"`
					MaxSpeedPerSecond       int `json:"max_speed_per_second"`
					MinSpeedPerSecond       int `json:"min_speed_per_second"`
					TotalLength             int `json:"total_length"`
					Tracks                  []struct {
						Props []struct {
							DurationTimeMs int    `json:"duration_time_ms"`
							Length         int    `json:"length"`
							Position       int    `json:"position"`
							PropId         string `json:"prop_id"`
							RoleId         string `json:"role_id"`
							Score          int    `json:"score,omitempty"`
							TrackId        int    `json:"track_id"`
						} `json:"props"`
						TrackArrangeType string `json:"track_arrange_type"`
					} `json:"tracks"`
				} `json:"dragon_boat_2023_play_script"`
				GameConfigType string `json:"game_config_type"`
			} `json:"play_script"`
		} `json:"data"`
		Errcode int `json:"errcode"`
		Graphid int `json:"graphid"`
	}
)

func create() bool {
	go batchReport()
	createUrl := fmt.Sprintf("%s?session_token=%s", "https://payapp.weixin.qq.com/coupon-center-activity/game/create", token)

	c := &Create{
		ActivityId: activeId,
		GamePkId:   "",
	}

	payload, _ := json.Marshal(c)
	httpReq, _ := http.NewRequest(http.MethodPost, createUrl, bytes.NewReader(payload))
	httpReq.Header = header()

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return true
	}

	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return true
	}

	response := new(CreateResponse)
	err = json.Unmarshal(res, response)

	var scoreItems []ReportScoreItems
	var score int
	for _, track := range response.Data.PlayScript.DragonBoat2023PlayScript.Tracks {
		for _, prop := range track.Props {
			score += prop.Score
			if prop.Score == 0 {
				continue
			}

			scoreItems = append(scoreItems, ReportScoreItems{
				PropId:           prop.PropId,
				AwardScore:       prop.Score,
				FetchTimestampMs: time.Now().UnixMilli(),
			})
		}
	}

	fmt.Println("当前游戏分数：", score)
	if score < maxScore {
		return true
	}

	time.Sleep(30 * time.Second)
	report(response, scoreItems, score)
	return false
}

func batchReport() {
	batchReportUrl := fmt.Sprintf("%s?session_token=%s", "https://payapp.weixin.qq.com/coupon-center-report/statistic/batchreport", token)

	payload := fmt.Sprintf(`{"device":"DEVICE_IOS","device_platform":"ios","device_system":"iOS 16.5","device_brand":"iPhone","device_model":"iPhone 13<iPhone14,5>","wechat_version":"8.0.38","wxa_sdk_version":"2.32.2","wxa_custom_version":"6.30.8","source_scene":"1007","event_list":[{"event_code":"ResultPageExposure","intval1":1,"event_target":"%d","logid":"enqizk2niisv86tq80"}],"sid":"e5849bc7c67e75d13175e5930d032af8","coutom_version":"6.30.8"}`, activeId)
	httpReq, _ := http.NewRequest(http.MethodPost, batchReportUrl, strings.NewReader(payload))
	httpReq.Header = header()
	httpReq.Header.Set("mpm-model", "iPhone 13<iPhone14,5>")
	httpReq.Header.Set("mpm-system", "iOS 16.5")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	fmt.Println("batchReport res == ", string(res))
}

type (
	Report struct {
		ActivityId          int                 `json:"activity_id"`
		GameId              string              `json:"game_id"`
		GameReportScoreInfo GameReportScoreInfo `json:"game_report_score_info"`
	}

	GameReportScoreInfo struct {
		ScoreItems []ReportScoreItems `json:"score_items"`
		GameScore  int                `json:"game_score"`
		TotalScore int                `json:"total_score"`
	}

	ReportScoreItems struct {
		PropId           string `json:"prop_id"`
		AwardScore       int    `json:"award_score"`
		FetchTimestampMs int64  `json:"fetch_timestamp_ms"`
	}
)

func report(cr *CreateResponse, items []ReportScoreItems, score int) {
	go batchReport()
	if score == 0 {
		fmt.Println("分数太低")
		return
	}
	reportUrl := fmt.Sprintf("%s?session_token=%s", "https://payapp.weixin.qq.com/coupon-center-activity/game/report", token)

	r := &Report{
		ActivityId: activeId,
		GameId:     cr.Data.GameId,
		GameReportScoreInfo: GameReportScoreInfo{
			ScoreItems: items,
			GameScore:  score,
			TotalScore: score,
		},
	}

	payload, _ := json.Marshal(r)

	httpReq, _ := http.NewRequest(http.MethodPost, reportUrl, bytes.NewReader(payload))
	httpReq.Header = header()

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {

		return
	}

	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	fmt.Println("report res == ", string(res))
	getScore(cr.Data.GameId)
}

func getScore(gameId string) {
	val := url.Values{}
	val.Set("session_token", token)
	val.Set("active_id", strconv.Itoa(activeId))
	val.Set("game_id", gameId)
	val.Set("sid", "e5849bc7c67e75d13175e5930d032af8")
	val.Set("coutom_version", "6.30.6")

	scoreUrl := fmt.Sprintf("%s?%s", "https://payapp.weixin.qq.com/coupon-center-activity/game/get", val.Encode())

	resp, err := http.Get(scoreUrl)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {

		return
	}

	fmt.Println("getScore res == ", string(res))

	awardObtain(gameId)
}

type AwardObtain struct {
	ActivityId     int    `json:"activity_id"`
	GameId         string `json:"game_id"`
	ObtainOrnament bool   `json:"obtain_ornament"`
	RequestId      string `json:"request_id"`
	Sid            string `json:"sid"`
	CoutomVersion  string `json:"coutom_version"`
}

func awardObtain(gameId string) {
	awardObtainUrl := fmt.Sprintf("%s?session_token=%s", "https://payapp.weixin.qq.com/coupon-center-activity/award/obtain", token)

	requestId := fmt.Sprintf("%s%s", "osd2L5XcCO62Rhuave602-qZzmy0_lj51xsaf_", randLetterStr(4))
	a := &AwardObtain{
		ActivityId:     activeId,
		GameId:         gameId,
		ObtainOrnament: true,
		RequestId:      requestId,
		Sid:            "e5849bc7c67e75d13175e5930d032af8",
		CoutomVersion:  "6.30.8",
	}
	//a.RequestId = "osd2L5XcCO62Rhuave602-qZzmy0_lj51xsaf_bxfr"

	payload, _ := json.Marshal(a)
	httpReq, _ := http.NewRequest(http.MethodPost, awardObtainUrl, bytes.NewReader(payload))
	httpReq.Header = header()

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {

		return
	}
	fmt.Println("awardObtain res == ", string(res))
}

func header() http.Header {
	headers := http.Header{}
	headers.Set("Wepaytest-Proxyip", "905333")
	headers.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.38(0x18002629) NetType/WIFI Language/zh_CN miniProgram/wxe73c2db202c7eebf")
	headers.Set("Referer", "https://td.cdn-go.cn/")
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Requested-With", "com.tencent.mm")
	return headers
}

func randLetterStr(l int) string {
	str := "ABCDEFGHIGKLMNOPQRSTUVWXYZabcdefghigklmnopqrstuvwxyz0123456789"
	b := []byte(str)
	var result []byte
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, b[r.Intn(len(b))])
	}
	return string(result)
}
