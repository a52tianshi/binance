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

	// 读取最新的14天数据
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

	if len(prices) < 1440*7 {
		log.Fatalf("数据不足，需要至少 %d 条，实际只有 %d 条", 1440*7, len(prices))
	}

	// 只取最近7天的数据
	recent7Days := prices[len(prices)-1440*7:]
	fmt.Printf("最近7天数据: %d 条\n", len(recent7Days))

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

	maxWindow := 1440 * 7
	fmt.Printf("开始计算 %d x %d 的z-score矩阵...\n", len(recent7Days), maxWindow)
	fmt.Println("这可能需要一些时间，请耐心等待...\n")

	// 创建矩阵：行=时间点，列=时间窗口
	matrix := make([][]float64, len(recent7Days))
	for i := range matrix {
		matrix[i] = make([]float64, maxWindow)
	}

	// 计算每个时间点的z-score
	for timeIdx := 0; timeIdx < len(recent7Days); timeIdx++ {
		currentPrice := recent7Days[timeIdx]

		// 对于每个时间窗口
		for window := 1; window <= maxWindow && timeIdx >= window; window++ {
			prevPrice := recent7Days[timeIdx-window]
			returnPct := ((currentPrice - prevPrice) / prevPrice) * 100

			// 获取该窗口的均值和标准差
			volData, exists := volatilityData[window]
			if !exists {
				matrix[timeIdx][window-1] = 0
				continue
			}

			// 计算z-score
			var zScore float64
			if volData.StdDev > 0 {
				zScore = (returnPct - volData.Mean) / volData.StdDev
			} else {
				zScore = 0
			}

			matrix[timeIdx][window-1] = zScore
		}

		// 对于 window > timeIdx 的情况，无法计算，设为0
		for window := timeIdx + 1; window <= maxWindow; window++ {
			matrix[timeIdx][window-1] = 0
		}

		// 进度输出
		if (timeIdx+1)%1000 == 0 || timeIdx < 10 {
			progress := float64(timeIdx+1) / float64(len(recent7Days)) * 100
			fmt.Printf("进度: %.1f%% (%d/%d)\n", progress, timeIdx+1, len(recent7Days))
		}
	}

	// 保存矩阵到CSV
	fmt.Println("\n正在保存矩阵到CSV...")
	outputFile, err := os.Create("zscore_matrix.csv")
	if err != nil {
		log.Fatal("创建输出文件失败:", err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// 写入标题行（窗口1到maxWindow）
	header := make([]string, maxWindow+1)
	header[0] = "TimeIndex"
	for i := 1; i <= maxWindow; i++ {
		header[i] = strconv.Itoa(i)
	}
	writer.Write(header)

	// 写入数据
	for i, row := range matrix {
		rowStr := make([]string, maxWindow+1)
		rowStr[0] = strconv.Itoa(i)
		for j, val := range row {
			rowStr[j+1] = strconv.FormatFloat(val, 'f', 4, 64)
		}
		writer.Write(rowStr)

		if (i+1)%1000 == 0 {
			fmt.Printf("已写入 %d/%d 行\n", i+1, len(matrix))
		}
	}

	fmt.Printf("\n计算完成！\n")
	fmt.Printf("矩阵大小: %d x %d\n", len(recent7Days), maxWindow)
	fmt.Printf("结果已保存到 zscore_matrix.csv\n")
}

type VolatilityData struct {
	Mean   float64
	StdDev float64
}

