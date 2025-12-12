import csv
import statistics
import sys
from datetime import datetime

# 读取CSV文件
print("正在读取数据...", flush=True)
prices = []
timestamps = []

with open('ETHUSDT_minute_klines.csv', 'r', encoding='utf-8') as f:
    reader = csv.DictReader(f)
    for row in reader:
        try:
            close_price = float(row['Close'])
            prices.append(close_price)
            timestamps.append(row['Open Time (UTC)'])
        except (ValueError, KeyError):
            continue

print(f"共读取 {len(prices)} 条数据", flush=True)

# 计算不同时间窗口的标准差
max_window = 1440 * 7  # 7天 = 10080分钟
results = []

print(f"\n开始计算从1分钟到{max_window}分钟的标准差...", flush=True)
print("这可能需要一些时间，请耐心等待...\n", flush=True)

start_time = datetime.now()
for window in range(1, max_window + 1):
    if window > len(prices):
        break
    
    # 计算该窗口的收益率（使用列表推导式提高效率）
    returns = [((prices[i] - prices[i - window]) / prices[i - window]) * 100 
               for i in range(window, len(prices))]
    
    if len(returns) > 1:
        std_dev = statistics.stdev(returns)
        mean = statistics.mean(returns)
        results.append({
            'Window_Minutes': window,
            'Window_Days': round(window / 1440, 4),
            'Mean_Pct': mean,
            'StdDev_Pct': std_dev,
            'Sample_Count': len(returns)
        })
        
        # 更频繁的进度输出：前100个每10个输出，之后每100个输出
        if window <= 100 and window % 10 == 0:
            progress = (window / max_window) * 100
            print(f"[{progress:.1f}%] 窗口 {window} 分钟 ({window/1440:.4f} 天): 标准差 = {std_dev:.6f}%, 样本数 = {len(returns)}", flush=True)
        elif window > 100 and window % 100 == 0:
            progress = (window / max_window) * 100
            elapsed = (datetime.now() - start_time).total_seconds()
            print(f"[{progress:.1f}%] 窗口 {window} 分钟 ({window/1440:.4f} 天): 标准差 = {std_dev:.6f}%, 样本数 = {len(returns)}, 已用时: {elapsed:.1f}秒", flush=True)
        elif window <= 10:
            print(f"窗口 {window} 分钟 ({window/1440:.4f} 天): 标准差 = {std_dev:.6f}%, 样本数 = {len(returns)}", flush=True)
    elif len(returns) == 1:
        results.append({
            'Window_Minutes': window,
            'Window_Days': round(window / 1440, 4),
            'Mean_Pct': returns[0],
            'StdDev_Pct': 0.0,
            'Sample_Count': 1
        })

# 保存结果到CSV
print("\n正在保存结果到CSV...", flush=True)
output_file = 'multi_timeframe_volatility.csv'
with open(output_file, 'w', newline='', encoding='utf-8') as f:
    if results:
        writer = csv.DictWriter(f, fieldnames=['Window_Minutes', 'Window_Days', 'Mean_Pct', 'StdDev_Pct', 'Sample_Count'])
        writer.writeheader()
        writer.writerows(results)

total_time = (datetime.now() - start_time).total_seconds()
print(f"\n计算完成！", flush=True)
print(f"共计算了 {len(results)} 个时间窗口", flush=True)
print(f"总用时: {total_time:.1f}秒", flush=True)
print(f"结果已保存到 {output_file}", flush=True)

# 显示一些关键时间点的结果
print("\n关键时间窗口的标准差:", flush=True)
key_windows = [1, 5, 15, 30, 60, 240, 1440, 1440*2, 1440*3, 1440*7]
for kw in key_windows:
    if kw <= len(results):
        result = results[kw - 1]
        print(f"{result['Window_Minutes']} 分钟 ({result['Window_Days']} 天): 标准差 = {result['StdDev_Pct']:.6f}%", flush=True)

