package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
)

func main() {
	fmt.Println("正在读取数据...")

	// 读取价格数据
	priceFile, err := os.Open("ETHUSDT_minute_klines.csv")
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
	for i := 1; i < len(priceRecords); i++ {
		if len(priceRecords[i]) < 6 {
			continue
		}
		closePrice, err := strconv.ParseFloat(priceRecords[i][5], 64) // Close在索引5
		if err != nil {
			continue
		}
		prices = append(prices, closePrice)
	}

	if len(prices) == 0 {
		log.Fatal("没有有效的价格数据")
	}

	lastPrice := prices[len(prices)-1]
	fmt.Printf("最后时刻价格: %.2f\n", lastPrice)
	fmt.Printf("数据总条数: %d\n\n", len(prices))

	// 读取波动率数据
	volFile, err := os.Open("multi_timeframe_volatility.csv")
	if err != nil {
		log.Fatal("无法打开波动率文件:", err)
	}
	defer volFile.Close()

	volReader := csv.NewReader(volFile)
	volRecords, err := volReader.ReadAll()
	if err != nil {
		log.Fatal("读取波动率CSV失败:", err)
	}

	// 解析波动率数据（跳过标题行）
	volatilityData := make(map[int]VolatilityData)
	for i := 1; i < len(volRecords); i++ {
		if len(volRecords[i]) < 5 {
			continue
		}
		window, err := strconv.Atoi(volRecords[i][0])
		if err != nil {
			continue
		}
		mean, err := strconv.ParseFloat(volRecords[i][2], 64)
		if err != nil {
			continue
		}
		stdDev, err := strconv.ParseFloat(volRecords[i][3], 64)
		if err != nil {
			continue
		}
		volatilityData[window] = VolatilityData{
			Mean:   mean,
			StdDev: stdDev,
		}
	}

	fmt.Println("开始计算z-score...")
	fmt.Println("时间窗口范围: 1分钟到1440分钟（1天）\n")

	// 计算z-score
	results := make([]ZScoreResult, 0, 1440)

	for window := 1; window <= 1440 && window < len(prices); window++ {
		// 计算最后时刻相对于窗口前价格的收益率
		prevPrice := prices[len(prices)-1-window]
		returnPct := ((lastPrice - prevPrice) / prevPrice) * 100

		// 获取该窗口的均值和标准差
		volData, exists := volatilityData[window]
		if !exists {
			continue
		}

		// 计算z-score: (收益率 - 均值) / 标准差
		var zScore float64
		if volData.StdDev > 0 {
			zScore = (returnPct - volData.Mean) / volData.StdDev
		} else {
			zScore = 0
		}

		results = append(results, ZScoreResult{
			WindowMinutes: window,
			WindowDays:    float64(window) / 1440.0,
			ReturnPct:     returnPct,
			Mean:          volData.Mean,
			StdDev:        volData.StdDev,
			ZScore:        zScore,
		})

		// 每100个窗口输出一次进度
		if window%100 == 0 || window <= 10 {
			fmt.Printf("窗口 %d 分钟 (%.4f 天): 收益率 = %.6f%%, z-score = %.4f\n",
				window, float64(window)/1440.0, returnPct, zScore)
		}
	}

	// 保存结果到CSV
	fmt.Println("\n正在保存结果到CSV...")
	outputFile, err := os.Create("zscore_results.csv")
	if err != nil {
		log.Fatal("创建输出文件失败:", err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// 写入标题
	writer.Write([]string{"Window_Minutes", "Window_Days", "Return_Pct", "Mean_Pct", "StdDev_Pct", "Z_Score"})

	// 写入数据
	for _, result := range results {
		writer.Write([]string{
			strconv.Itoa(result.WindowMinutes),
			strconv.FormatFloat(result.WindowDays, 'f', 4, 64),
			strconv.FormatFloat(result.ReturnPct, 'f', 6, 64),
			strconv.FormatFloat(result.Mean, 'f', 6, 64),
			strconv.FormatFloat(result.StdDev, 'f', 6, 64),
			strconv.FormatFloat(result.ZScore, 'f', 4, 64),
		})
	}

	fmt.Printf("计算完成！\n")
	fmt.Printf("共计算了 %d 个时间窗口的z-score\n", len(results))
	fmt.Printf("结果已保存到 zscore_results.csv\n\n")

	// 显示关键时间点的结果
	fmt.Println("关键时间窗口的z-score:")
	keyWindows := []int{1, 5, 15, 30, 60, 240, 1440}
	for _, kw := range keyWindows {
		if kw <= len(results) {
			result := results[kw-1]
			fmt.Printf("%d 分钟 (%.4f 天): 收益率 = %.6f%%, z-score = %.4f\n",
				result.WindowMinutes, result.WindowDays, result.ReturnPct, result.ZScore)
		}
	}

	// 找出z-score的极值
	if len(results) > 0 {
		maxZScore := results[0].ZScore
		minZScore := results[0].ZScore
		maxIdx := 0
		minIdx := 0
		for i, r := range results {
			if r.ZScore > maxZScore {
				maxZScore = r.ZScore
				maxIdx = i
			}
			if r.ZScore < minZScore {
				minZScore = r.ZScore
				minIdx = i
			}
		}
		fmt.Printf("\n最大z-score: %.4f (窗口 %d 分钟, %.4f 天)\n",
			maxZScore, results[maxIdx].WindowMinutes, results[maxIdx].WindowDays)
		fmt.Printf("最小z-score: %.4f (窗口 %d 分钟, %.4f 天)\n",
			minZScore, results[minIdx].WindowMinutes, results[minIdx].WindowDays)
	}
}

type VolatilityData struct {
	Mean   float64
	StdDev float64
}

type ZScoreResult struct {
	WindowMinutes int
	WindowDays    float64
	ReturnPct     float64
	Mean          float64
	StdDev        float64
	ZScore        float64
}

