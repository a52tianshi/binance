import requests
import datetime
import time
import csv

def fetch_minute_klines(symbol: str, start_date: str, end_date: str = None):
    """下载币安分钟K线数据"""
    base_url = "https://api.binance.com/api/v3/klines"
    interval = "1m"
    
    start_dt = datetime.datetime.strptime(start_date, "%Y-%m-%d")
    if end_date:
        end_dt = datetime.datetime.strptime(end_date, "%Y-%m-%d")
    else:
        end_dt = datetime.datetime.now()
    
    start_ms = int(start_dt.timestamp() * 1000)
    end_ms = int(end_dt.timestamp() * 1000)
    
    all_data = []
    current_start = start_ms
    limit = 1000
    
    print(f"正在下载 {symbol} 从 {start_date} 到 {end_dt.strftime('%Y-%m-%d')} 的分钟K线数据...")
    
    while current_start < end_ms:
        current_end = min(current_start + (limit - 1) * 60 * 1000, end_ms)
        
        params = {
            "symbol": symbol,
            "interval": interval,
            "startTime": current_start,
            "endTime": current_end,
            "limit": limit
        }
        
        try:
            resp = requests.get(base_url, params=params)
            resp.raise_for_status()
            data = resp.json()
            
            if not data:
                break
            
            all_data.extend(data)
            last_timestamp = data[-1][0]
            current_start = last_timestamp + 1
            
            if len(all_data) % 5000 == 0:
                last_date = datetime.datetime.fromtimestamp(last_timestamp / 1000)
                print(f"已下载到 {last_date.strftime('%Y-%m-%d %H:%M:%S')} ({len(all_data)} 条)...")
            
            time.sleep(0.2)
            
        except Exception as e:
            print(f"错误: {e}")
            break
    
    return all_data

def save_to_csv(data, filename):
    """保存到CSV"""
    with open(filename, mode="w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        writer.writerow([
            "Open Time", "Open Time (UTC)", "Open", "High", "Low", "Close",
            "Volume", "Close Time", "Close Time (UTC)", "Quote Asset Volume",
            "Number of Trades", "Taker Buy Base Asset Volume", "Taker Buy Quote Asset Volume"
        ])
        
        for candle in data:
            open_time_ms = int(candle[0])
            close_time_ms = int(candle[6])
            open_time_utc = datetime.datetime.fromtimestamp(open_time_ms / 1000)
            close_time_utc = datetime.datetime.fromtimestamp(close_time_ms / 1000)
            
            writer.writerow([
                open_time_ms,
                open_time_utc.strftime("%Y-%m-%d %H:%M:%S"),
                candle[1], candle[2], candle[3], candle[4], candle[5],
                close_time_ms,
                close_time_utc.strftime("%Y-%m-%d %H:%M:%S"),
                candle[7], candle[8], candle[9], candle[10]
            ])
    
    print(f"\n数据已保存到 {filename}")
    print(f"总记录数: {len(data)}")

if __name__ == "__main__":
    symbol = "ETHUSDT"
    end_date = datetime.datetime.now()
    start_date = end_date - datetime.timedelta(days=14)
    
    start_date_str = start_date.strftime("%Y-%m-%d")
    end_date_str = end_date.strftime("%Y-%m-%d")
    
    print(f"下载最近14天的数据...")
    print(f"时间段: {start_date_str} 到 {end_date_str}\n")
    
    data = fetch_minute_klines(symbol, start_date_str, end_date_str)
    
    if data:
        save_to_csv(data, "ETHUSDT_latest_14days.csv")
        print(f"\n成功下载 {len(data)} 条分钟K线数据")
    else:
        print("没有下载到数据")

