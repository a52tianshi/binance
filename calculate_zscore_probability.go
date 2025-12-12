package main

import (
	"fmt"
	"math"
)

func main() {
	zScore := -2.5432
	
	// 计算标准正态分布的累积分布函数(CDF)
	// P(Z <= z) 表示z-score小于等于该值的概率
	probability := normalCDF(zScore)
	
	fmt.Printf("Z-score: %.4f\n", zScore)
	fmt.Printf("累积概率 P(Z <= %.4f) = %.6f = %.4f%%\n", zScore, probability, probability*100)
	fmt.Printf("这意味着有 %.4f%% 的概率收益率会低于或等于这个值\n\n", probability*100)
	
	// 计算双侧概率（绝对值）
	absZ := math.Abs(zScore)
	twoTailProb := 2 * (1 - normalCDF(absZ))
	fmt.Printf("双侧概率 P(|Z| >= %.4f) = %.6f = %.4f%%\n", absZ, twoTailProb, twoTailProb*100)
	fmt.Printf("这意味着有 %.4f%% 的概率收益率会偏离均值超过 %.4f 个标准差\n", twoTailProb*100, absZ)
}

// 标准正态分布的累积分布函数(CDF)
// 使用误差函数的近似公式
func normalCDF(z float64) float64 {
	// 使用Abramowitz and Stegun近似公式
	// 对于负值，使用对称性: P(Z <= -z) = 1 - P(Z <= z)
	if z < 0 {
		return 1 - normalCDF(-z)
	}
	
	// 对于z >= 0的情况
	t := 1.0 / (1.0 + 0.2316419*z)
	d := 0.3989423 * math.Exp(-z*z/2)
	p := d * t * (0.3193815 + t*(-0.3565638+t*(1.781478+t*(-1.821256+t*1.330274))))
	
	return 1 - p
}

