import requests
import datetime
import time
import csv

def fetch_eth_prices_at_16_beijing(start_date, end_date):
    base_url = "https://api.binance.com/api/v3/klines"
    interval = "1h"
    symbol = "ETHUSDT"

    start_dt = datetime.datetime.strptime(start_date, "%Y-%m-%d")
    end_dt = datetime.datetime.strptime(end_date, "%Y-%m-%d")

    results = []

    current = start_dt
    while current <= end_dt:
        utc_target = current + datetime.timedelta(hours=8)  # 转换为 UTC
        utc_ms = int(utc_target.timestamp() * 1000)

        params = {
            "symbol": symbol,
            "interval": interval,
            "startTime": utc_ms,
            "endTime": utc_ms + 60 * 60 * 1000,
            "limit": 1
        }

        try:
            resp = requests.get(base_url, params=params)
            resp.raise_for_status()
            data = resp.json()
            if data:
                candle = data[0]
                close_price = candle[4]
                results.append([current.strftime("%Y-%m-%d"), close_price])
        except Exception as e:
            print(f"Error on {current.date()}: {e}")
        
        current += datetime.timedelta(days=1)
        # 打印当前日期
        print(f"Fetching data for {current.date()}...")
        # 打印价格
        if results:
            print(f"ETH/USDT Close Price: {results[-1][1]}")
        else:
            print("No data found for this date.")
        time.sleep(0.5)  # 避免频率限制

    return results

# 修改起止日期范围
data = fetch_eth_prices_at_16_beijing("2019-05-17", "2025-05-23")

# 写入CSV
with open("eth_prices_16pm_beijing.csv", mode="w", newline="") as file:
    writer = csv.writer(file)
    writer.writerow(["Date (Beijing Time)", "ETH/USDT Close Price"])
    writer.writerows(data)

print("已保存为 eth_prices_16pm_beijing.csv")