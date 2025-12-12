package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"
)

func main() {
	fmt.Println("正在读取数据...")
	
	// 读取CSV文件
	file, err := os.Open("ETHUSDT_minute_klines.csv")
	if err != nil {
		log.Fatal("无法打开文件:", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal("读取CSV失败:", err)
	}

	// 解析价格数据（跳过标题行）
	prices := make([]float64, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 6 {
			continue
		}
		closePrice, err := strconv.ParseFloat(records[i][5], 64) // Close在索引5
		if err != nil {
			continue
		}
		prices = append(prices, closePrice)
	}

	fmt.Printf("共读取 %d 条数据\n", len(prices))

	// 计算不同时间窗口的标准差
	maxWindow := 1440 * 7 // 7天 = 10080分钟
	results := make([]Result, 0, maxWindow)

	fmt.Printf("\n开始计算从1分钟到%d分钟的标准差...\n", maxWindow)
	fmt.Println("这可能需要一些时间，请耐心等待...\n")

	startTime := time.Now()
	
	for window := 1; window <= maxWindow && window < len(prices); window++ {
		// 计算该窗口的收益率
		returns := make([]float64, 0, len(prices)-window)
		for i := window; i < len(prices); i++ {
			returnPct := ((prices[i] - prices[i-window]) / prices[i-window]) * 100
			returns = append(returns, returnPct)
		}

		if len(returns) > 1 {
			mean := calculateMean(returns)
			stdDev := calculateStdDev(returns, mean)
			
			results = append(results, Result{
				WindowMinutes: window,
				WindowDays:    float64(window) / 1440.0,
				MeanPct:       mean,
				StdDevPct:     stdDev,
				SampleCount:   len(returns),
			})

			// 进度输出
			if window <= 100 && window%10 == 0 {
				progress := float64(window) / float64(maxWindow) * 100
				elapsed := time.Since(startTime).Seconds()
				fmt.Printf("[%.1f%%] 窗口 %d 分钟 (%.4f 天): 标准差 = %.6f%%, 样本数 = %d, 已用时: %.1f秒\n",
					progress, window, float64(window)/1440.0, stdDev, len(returns), elapsed)
			} else if window > 100 && window%100 == 0 {
				progress := float64(window) / float64(maxWindow) * 100
				elapsed := time.Since(startTime).Seconds()
				fmt.Printf("[%.1f%%] 窗口 %d 分钟 (%.4f 天): 标准差 = %.6f%%, 样本数 = %d, 已用时: %.1f秒\n",
					progress, window, float64(window)/1440.0, stdDev, len(returns), elapsed)
			} else if window <= 10 {
				fmt.Printf("窗口 %d 分钟 (%.4f 天): 标准差 = %.6f%%, 样本数 = %d\n",
					window, float64(window)/1440.0, stdDev, len(returns))
			}
		} else if len(returns) == 1 {
			results = append(results, Result{
				WindowMinutes: window,
				WindowDays:    float64(window) / 1440.0,
				MeanPct:       returns[0],
				StdDevPct:     0.0,
				SampleCount:   1,
			})
		}
	}

	// 保存结果到CSV
	fmt.Println("\n正在保存结果到CSV...")
	outputFile, err := os.Create("multi_timeframe_volatility.csv")
	if err != nil {
		log.Fatal("创建输出文件失败:", err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// 写入标题
	writer.Write([]string{"Window_Minutes", "Window_Days", "Mean_Pct", "StdDev_Pct", "Sample_Count"})

	// 写入数据
	for _, result := range results {
		writer.Write([]string{
			strconv.Itoa(result.WindowMinutes),
			strconv.FormatFloat(result.WindowDays, 'f', 4, 64),
			strconv.FormatFloat(result.MeanPct, 'f', 6, 64),
			strconv.FormatFloat(result.StdDevPct, 'f', 6, 64),
			strconv.Itoa(result.SampleCount),
		})
	}

	totalTime := time.Since(startTime).Seconds()
	fmt.Printf("\n计算完成！\n")
	fmt.Printf("共计算了 %d 个时间窗口\n", len(results))
	fmt.Printf("总用时: %.1f秒\n", totalTime)
	fmt.Printf("结果已保存到 multi_timeframe_volatility.csv\n")

	// 显示关键时间点的结果
	fmt.Println("\n关键时间窗口的标准差:")
	keyWindows := []int{1, 5, 15, 30, 60, 240, 1440, 1440 * 2, 1440 * 3, 1440 * 7}
	for _, kw := range keyWindows {
		if kw <= len(results) {
			result := results[kw-1]
			fmt.Printf("%d 分钟 (%.4f 天): 标准差 = %.6f%%\n",
				result.WindowMinutes, result.WindowDays, result.StdDevPct)
		}
	}
}

type Result struct {
	WindowMinutes int
	WindowDays    float64
	MeanPct       float64
	StdDevPct     float64
	SampleCount   int
}

func calculateMean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0.0
	}
	sumSqDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSqDiff += diff * diff
	}
	variance := sumSqDiff / float64(len(values)-1)
	return math.Sqrt(variance)
}

