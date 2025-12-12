import csv
import statistics

# 读取CSV文件
price_changes = []
data_rows = []

with open('ETHUSDT_minute_klines.csv', 'r', encoding='utf-8') as f:
    reader = csv.DictReader(f)
    for row in reader:
        try:
            open_price = float(row['Open'])
            close_price = float(row['Close'])
            
            # 计算涨跌百分比：(Close - Open) / Open * 100
            price_change_pct = ((close_price - open_price) / open_price) * 100
            
            price_changes.append(price_change_pct)
            data_rows.append({
                'Open Time (UTC)': row['Open Time (UTC)'],
                'Open': open_price,
                'Close': close_price,
                'Price_Change_Pct': price_change_pct
            })
        except (ValueError, KeyError) as e:
            continue

# 计算统计信息
if price_changes:
    mean = statistics.mean(price_changes)
    std_dev = statistics.stdev(price_changes) if len(price_changes) > 1 else 0
    median = statistics.median(price_changes)
    min_val = min(price_changes)
    max_val = max(price_changes)
    
    # 输出结果
    print(f"数据总条数: {len(price_changes)}")
    print(f"涨跌百分比均值: {mean:.6f}%")
    print(f"涨跌百分比标准差: {std_dev:.6f}%")
    print(f"涨跌百分比中位数: {median:.6f}%")
    print(f"涨跌百分比最小值: {min_val:.6f}%")
    print(f"涨跌百分比最大值: {max_val:.6f}%")
    
    # 保存结果到CSV
    with open('minute_price_changes.csv', 'w', newline='', encoding='utf-8') as f:
        if data_rows:
            writer = csv.DictWriter(f, fieldnames=['Open Time (UTC)', 'Open', 'Close', 'Price_Change_Pct'])
            writer.writeheader()
            writer.writerows(data_rows)
    print(f"\n结果已保存到 minute_price_changes.csv")
else:
    print("没有有效数据")


