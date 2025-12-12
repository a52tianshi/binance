package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
)

func main() {
	fmt.Println("正在分析三天前的数据...")

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

	// 解析价格数据（跳过标题行）
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
		timestamps = append(timestamps, priceRecords[i][1]) // UTC时间
	}

	if len(prices) < 1440*7 {
		log.Fatalf("数据不足")
	}

	// 只取最近7天的数据
	recent7Days := prices[len(prices)-1440*7:]
	recent7DaysTimestamps := timestamps[len(timestamps)-1440*7:]

	// 三天前大约是索引 4320 (3 * 1440)
	threeDaysAgoIdx := 1440 * 3
	fmt.Printf("三天前的时间点索引: %d\n", threeDaysAgoIdx)
	fmt.Printf("对应时间: %s\n", recent7DaysTimestamps[threeDaysAgoIdx])
	fmt.Printf("价格: %.2f\n\n", recent7Days[threeDaysAgoIdx])

	// 读取z-score矩阵
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

	// 分析三天前附近的数据（前后各1小时，即60个数据点）
	startIdx := threeDaysAgoIdx - 60
	endIdx := threeDaysAgoIdx + 60
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(recent7Days) {
		endIdx = len(recent7Days) - 1
	}

	fmt.Printf("分析时间段: 索引 %d 到 %d (三天前后各1小时)\n", startIdx, endIdx)
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	// 分析每个时间点的z-score
	maxZScore := 0.0
	maxZScoreIdx := 0
	maxZScoreWindow := 0

	for idx := startIdx; idx <= endIdx; idx++ {
		if idx+1 >= len(zscoreRecords) {
			continue
		}

		row := zscoreRecords[idx+1] // +1因为第一行是标题
		if len(row) < 2 {
			continue
		}

		// 检查不同时间窗口的z-score
		// 重点关注短时间窗口（1-60分钟）和中等窗口（60-240分钟）
		for window := 1; window <= 240 && window < len(row); window++ {
			zscore, err := strconv.ParseFloat(row[window], 64)
			if err != nil {
				continue
			}

			if zscore > maxZScore {
				maxZScore = zscore
				maxZScoreIdx = idx
				maxZScoreWindow = window
			}
		}
	}

	fmt.Printf("\n最大z-score: %.4f\n", maxZScore)
	fmt.Printf("出现在索引: %d\n", maxZScoreIdx)
	fmt.Printf("对应时间: %s\n", recent7DaysTimestamps[maxZScoreIdx])
	fmt.Printf("价格: %.2f\n", recent7Days[maxZScoreIdx])
	fmt.Printf("时间窗口: %d 分钟\n\n", maxZScoreWindow)

	// 分析三天前时间点附近的价格变化
	fmt.Println("三天前附近的价格变化:")
	fmt.Println("时间\t\t\t价格\t\t变化%")
	fmt.Println("-" + string(make([]byte, 60)) + "-")

	basePrice := recent7Days[threeDaysAgoIdx]
	for i := -10; i <= 10; i++ {
		idx := threeDaysAgoIdx + i
		if idx >= 0 && idx < len(recent7Days) {
			price := recent7Days[idx]
			change := ((price - basePrice) / basePrice) * 100
			fmt.Printf("%s\t%.2f\t\t%.4f%%\n", recent7DaysTimestamps[idx], price, change)
		}
	}

	// 分析三天前时间点的z-score分布
	fmt.Println("\n三天前时间点的z-score分布（不同窗口）:")
	fmt.Println("窗口(分钟)\tz-score\t\t收益率%")
	fmt.Println("-" + string(make([]byte, 50)) + "-")

	if threeDaysAgoIdx+1 < len(zscoreRecords) {
		row := zscoreRecords[threeDaysAgoIdx+1]
		keyWindows := []int{1, 5, 15, 30, 60, 120, 240, 1440, 2880, 4320}
		for _, window := range keyWindows {
			if window < len(row) {
				zscore, _ := strconv.ParseFloat(row[window], 64)
				if threeDaysAgoIdx >= window {
					prevPrice := recent7Days[threeDaysAgoIdx-window]
					returnPct := ((recent7Days[threeDaysAgoIdx] - prevPrice) / prevPrice) * 100
					fmt.Printf("%d\t\t%.4f\t\t%.4f%%\n", window, zscore, returnPct)
				}
			}
		}
	}
}

