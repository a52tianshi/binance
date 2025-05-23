package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// 配置 lumberjack 日志滚动
func setupLogger() {
	log.SetOutput(&lumberjack.Logger{
		Filename:   "binance.log",
		MaxSize:    100,   // 每个日志文件最大 10MB
		MaxBackups: 10000, //
		MaxAge:     30,    // 最多保留30天
		Compress:   true,
	})
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// 定义响应数据结构
type Product struct {
	ID                   string   `json:"id"`
	InvestCoin           string   `json:"investCoin"`
	ExercisedCoin        string   `json:"exercisedCoin"`
	StrikePrice          string   `json:"strikePrice"`
	Duration             int      `json:"duration"`
	SettleDate           int64    `json:"settleDate"`
	PurchaseDecimal      int      `json:"purchaseDecimal"`
	PurchaseEndTime      int64    `json:"purchaseEndTime"`
	CanPurchase          bool     `json:"canPurchase"`
	APR                  string   `json:"apr"`
	OrderID              int64    `json:"orderId"`
	MinAmount            string   `json:"minAmount"`
	MaxAmount            string   `json:"maxAmount"`
	CreateTimestamp      int64    `json:"createTimestamp"`
	OptionType           string   `json:"optionType"`
	IsAutoCompoundEnable bool     `json:"isAutoCompoundEnable"`
	AutoCompoundPlanList []string `json:"autoCompoundPlanList"`
}

type Response struct {
	Total int       `json:"total"`
	List  []Product `json:"list"`
}

// 签名生成
func getSignedQueryString(params map[string]string, secretKey string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	queryString := values.Encode()
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(queryString))
	signature := hex.EncodeToString(mac.Sum(nil))

	return queryString + "&signature=" + signature
}

// 请求一页数据，返回原始字符串
func fetchPageRaw(apiKey, secretKey, optionType, coin string, pageIndex int) (string, error) {
	endpoint := "https://api.binance.com/sapi/v1/dci/product/list"

	// 按题意，optionType 是 PUT 或 CALL
	// exercisedCoin 和 investCoin 规则（根据你之前说的）
	// CALL: exercisedCoin=USDT, investCoin=coin
	// PUT:  exercisedCoin=coin, investCoin=USDT
	var exercisedCoin, investCoin string
	if optionType == "CALL" {
		exercisedCoin = "USDT"
		investCoin = coin
	} else {
		exercisedCoin = coin
		investCoin = "USDT"
	}

	params := map[string]string{
		"optionType":    optionType,
		"exercisedCoin": exercisedCoin,
		"investCoin":    investCoin,
		"pageSize":      "100",
		"pageIndex":     strconv.Itoa(pageIndex),
		"recvWindow":    "5000",
		"timestamp":     strconv.FormatInt(time.Now().UnixMilli(), 10),
	}

	query := getSignedQueryString(params, secretKey)
	req, err := http.NewRequest("GET", endpoint+"?"+query, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-MBX-APIKEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

var apiKey, secretKey string

func runFullScrape() {

	coins := []string{"BTC", "ETH", "WBETH"}
	optionTypes := []string{"PUT", "CALL"}

	for _, coin := range coins {
		for _, optionType := range optionTypes {
			for page := 1; ; page++ {
				rawData, err := fetchPageRaw(apiKey, secretKey, optionType, coin, page)
				if err != nil {
					fmt.Println("请求失败:", err)
					break
				}

				if strings.Contains(rawData, "code") {
					fmt.Println("请求失败，可能是参数错误或其他问题:", rawData)
					break
				}

				log.Println(rawData)

				// 假设返回的 JSON 数据中有一个字段表示是否还有下一页
				if !strings.Contains(rawData, `id`) {
					break
				}

			}
		}
	}
}

func main() {
	setupLogger()
	apiKey = os.Getenv("BINANCE_API_KEY")
	secretKey = os.Getenv("BINANCE_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		log.Println("请设置环境变量 BINANCE_API_KEY 和 BINANCE_SECRET_KEY")
		return
	}

	var ticker *time.Ticker
	//每5s抓取一次
	ticker = time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			runFullScrape()
		}
		fmt.Println("抓取完成，等待下一次抓取...", time.Now().Format("2006-01-02 15:04:05"))
	}
	// 阻塞主线程
	select {}

	return
}
