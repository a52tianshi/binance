package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
)

func main() {
	fmt.Println("正在分析价格暴涨情况...")

	// 读取价格数据
	priceFile, err := os.Open("ETHUSDT_latest_14days.csv")
	if err != nil {
		log.Fatal("无法打开价格文件:", err)
	}
	defer priceFile.Close()

	priceReader := csv.NewReader(priceFile)
	priceRecords, err := priceReader.ReadAll()
	if err != nil {
		log.Fatal("读取价格CSV失败:", err)
	}

	// 解析价格数据
	prices := make([]float64, 0, len(priceRecords)-1)
	timestamps := make([]string, 0, len(priceRecords)-1)
	for i := 1; i < len(priceRecords); i++ {
		if len(priceRecords[i]) < 6 {
			continue
		}
		closePrice, err := strconv.ParseFloat(priceRecords[i][5], 64)
		if err != nil {
			continue
		}
		prices = append(prices, closePrice)
		timestamps = append(timestamps, priceRecords[i][1])
	}

	if len(prices) < 1440*7 {
		log.Fatalf("数据不足")
	}

	// 只取最近7天的数据
	recent7Days := prices[len(prices)-1440*7:]
	recent7DaysTimestamps := timestamps[len(timestamps)-1440*7:]

	// 三天前的时间点
	threeDaysAgoIdx := 1440 * 3
	fmt.Printf("三天前时间点: %s (索引 %d)\n", recent7DaysTimestamps[threeDaysAgoIdx], threeDaysAgoIdx)
	fmt.Printf("价格: %.2f\n\n", recent7Days[threeDaysAgoIdx])

	// 分析三天前前后6小时的价格变化
	fmt.Println("=" + string(make([]byte, 80)) + "=")
	fmt.Println("三天前前后6小时的价格变化分析:")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	startIdx := threeDaysAgoIdx - 360 // 6小时前
	endIdx := threeDaysAgoIdx + 360    // 6小时后
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(recent7Days) {
		endIdx = len(recent7Days) - 1
	}

	// 找出最大涨幅
	maxGain := 0.0
	maxGainIdx := 0
	maxGainWindow := 0

	// 分析不同时间窗口的收益率
	for idx := startIdx; idx <= endIdx; idx++ {
		// 检查1小时、4小时、1天的收益率
		windows := []int{60, 240, 1440}
		for _, window := range windows {
			if idx >= window {
				prevPrice := recent7Days[idx-window]
				currentPrice := recent7Days[idx]
				gain := ((currentPrice - prevPrice) / prevPrice) * 100

				if gain > maxGain {
					maxGain = gain
					maxGainIdx = idx
					maxGainWindow = window
				}
			}
		}
	}

	fmt.Printf("\n最大涨幅: %.4f%%\n", maxGain)
	fmt.Printf("出现在时间: %s (索引 %d)\n", recent7DaysTimestamps[maxGainIdx], maxGainIdx)
	fmt.Printf("价格: %.2f\n", recent7Days[maxGainIdx])
	fmt.Printf("时间窗口: %d 分钟 (%.1f 小时)\n\n", maxGainWindow, float64(maxGainWindow)/60)

	// 分析三天前前后24小时的价格走势
	fmt.Println("三天前前后24小时的价格走势（每小时）:")
	fmt.Println("时间\t\t\t价格\t\t1小时涨跌%\t4小时涨跌%\t1天涨跌%")
	fmt.Println("-" + string(make([]byte, 100)) + "-")

	hourlyIndices := []int{}
	for i := startIdx; i <= endIdx; i += 60 {
		hourlyIndices = append(hourlyIndices, i)
	}

	for _, idx := range hourlyIndices {
		if idx >= len(recent7Days) {
			break
		}
		price := recent7Days[idx]
		timeStr := recent7DaysTimestamps[idx]

		var gain1h, gain4h, gain1d string
		if idx >= 60 {
			gain1h = fmt.Sprintf("%.2f%%", ((price-recent7Days[idx-60])/recent7Days[idx-60])*100)
		} else {
			gain1h = "N/A"
		}
		if idx >= 240 {
			gain4h = fmt.Sprintf("%.2f%%", ((price-recent7Days[idx-240])/recent7Days[idx-240])*100)
		} else {
			gain4h = "N/A"
		}
		if idx >= 1440 {
			gain1d = fmt.Sprintf("%.2f%%", ((price-recent7Days[idx-1440])/recent7Days[idx-1440])*100)
		} else {
			gain1d = "N/A"
		}

		// 只显示关键时间点
		if idx%60 == 0 || idx == threeDaysAgoIdx {
			fmt.Printf("%s\t%.2f\t\t%s\t\t%s\t\t%s\n", timeStr, price, gain1h, gain4h, gain1d)
		}
	}

	// 读取z-score矩阵，分析三天前的z-score
	fmt.Println("\n" + "=" + string(make([]byte, 80)) + "=")
	fmt.Println("三天前时间点的z-score分析:")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	zscoreFile, err := os.Open("zscore_matrix.csv")
	if err != nil {
		log.Fatal("无法打开z-score文件:", err)
	}
	defer zscoreFile.Close()

	zscoreReader := csv.NewReader(zscoreFile)
	zscoreRecords, err := zscoreReader.ReadAll()
	if err != nil {
		log.Fatal("读取z-score CSV失败:", err)
	}

	if threeDaysAgoIdx+1 < len(zscoreRecords) {
		row := zscoreRecords[threeDaysAgoIdx+1]
		fmt.Println("\n不同时间窗口的z-score（正值表示高于历史均值）:")
		fmt.Println("窗口\t\tz-score\t\t收益率%\t\t说明")
		fmt.Println("-" + string(make([]byte, 70)) + "-")

		windows := []int{1, 5, 15, 30, 60, 240, 1440, 2880, 4320}
		for _, window := range windows {
			if window < len(row) && threeDaysAgoIdx >= window {
				zscore, _ := strconv.ParseFloat(row[window], 64)
				prevPrice := recent7Days[threeDaysAgoIdx-window]
				returnPct := ((recent7Days[threeDaysAgoIdx] - prevPrice) / prevPrice) * 100

				interpretation := ""
				if zscore > 2 {
					interpretation = "显著高于均值"
				} else if zscore > 1 {
					interpretation = "高于均值"
				} else if zscore < -2 {
					interpretation = "显著低于均值"
				} else if zscore < -1 {
					interpretation = "低于均值"
				} else {
					interpretation = "接近均值"
				}

				fmt.Printf("%d分钟\t\t%.4f\t\t%.4f%%\t\t%s\n", window, zscore, returnPct, interpretation)
			}
		}
	}

	// 检查是否有连续的正z-score（暴涨迹象）
	fmt.Println("\n" + "=" + string(make([]byte, 80)) + "=")
	fmt.Println("检查三天前附近是否有连续暴涨（z-score > 2）:")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	surgeCount := 0
	for idx := startIdx; idx <= endIdx; idx++ {
		if idx+1 >= len(zscoreRecords) {
			continue
		}
		row := zscoreRecords[idx+1]
		// 检查1小时窗口的z-score
		if 60 < len(row) && idx >= 60 {
			zscore, _ := strconv.ParseFloat(row[60], 64)
			if zscore > 2 {
				surgeCount++
				if surgeCount == 1 || idx%60 == 0 {
					fmt.Printf("时间: %s, 1小时窗口z-score: %.4f, 价格: %.2f\n",
						recent7DaysTimestamps[idx], zscore, recent7Days[idx])
				}
			}
		}
	}

	if surgeCount > 0 {
		fmt.Printf("\n发现 %d 个时间点的1小时窗口z-score > 2，可能存在暴涨\n", surgeCount)
	} else {
		fmt.Println("\n未发现明显的暴涨迹象（1小时窗口z-score > 2）")
	}
}

