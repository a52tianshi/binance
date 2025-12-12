package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
)

func main() {
	fmt.Println("正在分析最近几小时的数据...")

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

	// 分析最近6小时的数据
	recentHours := 6
	startIdx := len(recent7Days) - recentHours*60
	if startIdx < 0 {
		startIdx = 0
	}

	fmt.Printf("分析最近 %d 小时的数据（从索引 %d 到 %d）\n", recentHours, startIdx, len(recent7Days)-1)
	fmt.Printf("开始时间: %s\n", recent7DaysTimestamps[startIdx])
	fmt.Printf("结束时间: %s\n", recent7DaysTimestamps[len(recent7Days)-1])
	fmt.Printf("当前价格: %.2f\n\n", recent7Days[len(recent7Days)-1])

	fmt.Println("=" + string(make([]byte, 80)) + "=")
	fmt.Println("最近6小时的价格变化（每10分钟）:")
	fmt.Println("=" + string(make([]byte, 80)) + "=")
	fmt.Println("时间\t\t\t价格\t\t10分钟涨跌%\t1小时涨跌%\t6小时涨跌%")
	fmt.Println("-" + string(make([]byte, 100)) + "-")

	basePrice := recent7Days[startIdx]
	for i := startIdx; i < len(recent7Days); i += 10 {
		if i >= len(recent7Days) {
			break
		}
		price := recent7Days[i]
		timeStr := recent7DaysTimestamps[i]

		var change10m, change1h, change6h string
		if i >= 10 {
			change10m = fmt.Sprintf("%.4f%%", ((price-recent7Days[i-10])/recent7Days[i-10])*100)
		} else {
			change10m = "N/A"
		}
		if i >= 60 {
			change1h = fmt.Sprintf("%.4f%%", ((price-recent7Days[i-60])/recent7Days[i-60])*100)
		} else {
			change1h = "N/A"
		}
		change6h = fmt.Sprintf("%.4f%%", ((price-basePrice)/basePrice)*100)

		fmt.Printf("%s\t%.2f\t\t%s\t\t%s\t\t%s\n", timeStr, price, change10m, change1h, change6h)
	}

	// 找出最大跌幅
	fmt.Println("\n" + "=" + string(make([]byte, 80)) + "=")
	fmt.Println("寻找最大跌幅:")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	maxDrop := 0.0
	maxDropIdx := 0
	maxDropWindow := 0

	for idx := startIdx; idx < len(recent7Days); idx++ {
		windows := []int{10, 30, 60, 120, 360} // 10分钟, 30分钟, 1小时, 2小时, 6小时
		for _, window := range windows {
			if idx >= window && idx-window >= startIdx {
				prevPrice := recent7Days[idx-window]
				currentPrice := recent7Days[idx]
				drop := ((prevPrice - currentPrice) / prevPrice) * 100 // 跌幅为正数

				if drop > maxDrop {
					maxDrop = drop
					maxDropIdx = idx
					maxDropWindow = window
				}
			}
		}
	}

	fmt.Printf("最大跌幅: %.4f%%\n", maxDrop)
	fmt.Printf("出现在时间: %s (索引 %d)\n", recent7DaysTimestamps[maxDropIdx], maxDropIdx)
	fmt.Printf("价格: %.2f\n", recent7Days[maxDropIdx])
	fmt.Printf("时间窗口: %d 分钟 (%.1f 小时)\n", maxDropWindow, float64(maxDropWindow)/60)

	if maxDropWindow > 0 {
		prevPrice := recent7Days[maxDropIdx-maxDropWindow]
		fmt.Printf("对比价格: %.2f\n", prevPrice)
		fmt.Printf("价格变化: %.2f -> %.2f\n", prevPrice, recent7Days[maxDropIdx])
	}

	// 读取z-score矩阵，分析最近几小时的z-score
	fmt.Println("\n" + "=" + string(make([]byte, 80)) + "=")
	fmt.Println("最近几小时的z-score分析（负值表示低于历史均值，可能是暴跌）:")
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

	// 分析最近6小时的z-score
	fmt.Println("\n最近6小时的关键时间点z-score:")
	fmt.Println("时间\t\t\t价格\t\t1分钟z\t\t15分钟z\t\t1小时z\t\t4小时z")
	fmt.Println("-" + string(make([]byte, 100)) + "-")

	for i := startIdx; i < len(recent7Days); i += 30 { // 每30分钟显示一次
		if i+1 >= len(zscoreRecords) {
			continue
		}
		row := zscoreRecords[i+1] // +1因为第一行是标题
		if len(row) < 240 {
			continue
		}

		price := recent7Days[i]
		timeStr := recent7DaysTimestamps[i]

		var z1m, z15m, z1h, z4h string
		if 1 < len(row) && i >= 1 {
			z, _ := strconv.ParseFloat(row[1], 64)
			z1m = fmt.Sprintf("%.2f", z)
		} else {
			z1m = "N/A"
		}
		if 15 < len(row) && i >= 15 {
			z, _ := strconv.ParseFloat(row[15], 64)
			z15m = fmt.Sprintf("%.2f", z)
		} else {
			z15m = "N/A"
		}
		if 60 < len(row) && i >= 60 {
			z, _ := strconv.ParseFloat(row[60], 64)
			z1h = fmt.Sprintf("%.2f", z)
		} else {
			z1h = "N/A"
		}
		if 240 < len(row) && i >= 240 {
			z, _ := strconv.ParseFloat(row[240], 64)
			z4h = fmt.Sprintf("%.2f", z)
		} else {
			z4h = "N/A"
		}

		fmt.Printf("%s\t%.2f\t\t%s\t\t%s\t\t%s\t\t%s\n", timeStr, price, z1m, z15m, z1h, z4h)
	}

	// 检查是否有显著的负z-score（暴跌迹象）
	fmt.Println("\n" + "=" + string(make([]byte, 80)) + "=")
	fmt.Println("检查暴跌迹象（z-score < -2，表示显著低于历史均值）:")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	crashCount := 0
	for idx := startIdx; idx < len(recent7Days); idx++ {
		if idx+1 >= len(zscoreRecords) {
			continue
		}
		row := zscoreRecords[idx+1]
		// 检查1小时窗口的z-score
		if 60 < len(row) && idx >= 60 {
			zscore, _ := strconv.ParseFloat(row[60], 64)
			if zscore < -2 {
				crashCount++
				if crashCount <= 10 || idx%30 == 0 { // 只显示前10个或每30分钟
					fmt.Printf("时间: %s, 1小时窗口z-score: %.4f, 价格: %.2f\n",
						recent7DaysTimestamps[idx], zscore, recent7Days[idx])
				}
			}
		}
	}

	if crashCount > 0 {
		fmt.Printf("\n发现 %d 个时间点的1小时窗口z-score < -2，可能存在暴跌\n", crashCount)
	} else {
		fmt.Println("\n未发现明显的暴跌迹象（1小时窗口z-score < -2）")
	}

	// 分析当前时刻的z-score
	fmt.Println("\n" + "=" + string(make([]byte, 80)) + "=")
	fmt.Println("当前时刻（最新数据点）的z-score分析:")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	lastIdx := len(recent7Days) - 1
	if lastIdx+1 < len(zscoreRecords) {
		row := zscoreRecords[lastIdx+1]
		fmt.Println("窗口\t\tz-score\t\t收益率%\t\t说明")
		fmt.Println("-" + string(make([]byte, 70)) + "-")

		windows := []int{1, 5, 15, 30, 60, 240}
		for _, window := range windows {
			if window < len(row) && lastIdx >= window {
				zscore, _ := strconv.ParseFloat(row[window], 64)
				prevPrice := recent7Days[lastIdx-window]
				returnPct := ((recent7Days[lastIdx] - prevPrice) / prevPrice) * 100

				interpretation := ""
				if zscore < -2 {
					interpretation = "显著低于均值（暴跌）"
				} else if zscore < -1 {
					interpretation = "低于均值（下跌）"
				} else if zscore > 2 {
					interpretation = "显著高于均值（暴涨）"
				} else if zscore > 1 {
					interpretation = "高于均值（上涨）"
				} else {
					interpretation = "接近均值"
				}

				fmt.Printf("%d分钟\t\t%.4f\t\t%.4f%%\t\t%s\n", window, zscore, returnPct, interpretation)
			}
		}
	}
}

