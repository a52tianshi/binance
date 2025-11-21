import requests
import datetime
import time
import csv
from typing import List, Tuple

def fetch_daily_klines(symbol: str, start_date: str, end_date: str = None) -> List[List]:
    """
    Fetch daily K-line data from Binance API
    
    Args:
        symbol: Trading pair symbol (e.g., 'BTCUSDT', 'ETHUSDT')
        start_date: Start date in 'YYYY-MM-DD' format
        end_date: End date in 'YYYY-MM-DD' format (default: today)
    
    Returns:
        List of K-line data [timestamp, open, high, low, close, volume, ...]
    """
    base_url = "https://api.binance.com/api/v3/klines"
    interval = "1d"  # Daily interval
    
    # Parse dates
    start_dt = datetime.datetime.strptime(start_date, "%Y-%m-%d")
    if end_date:
        end_dt = datetime.datetime.strptime(end_date, "%Y-%m-%d")
    else:
        end_dt = datetime.datetime.now()
    
    # Convert to UTC timestamps (milliseconds)
    start_ms = int(start_dt.timestamp() * 1000)
    end_ms = int(end_dt.timestamp() * 1000)
    
    all_data = []
    current_start = start_ms
    limit = 1000  # Binance API limit per request
    
    print(f"Downloading daily K-line data for {symbol} from {start_date} to {end_dt.strftime('%Y-%m-%d')}...")
    
    while current_start < end_ms:
        # Calculate end time for this batch (1000 days max per request)
        current_end = min(current_start + (limit - 1) * 24 * 60 * 60 * 1000, end_ms)
        
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
            
            # Get the timestamp of the last candle and move forward
            last_timestamp = data[-1][0]
            current_start = last_timestamp + 1  # Start from next millisecond
            
            # Print progress
            last_date = datetime.datetime.fromtimestamp(last_timestamp / 1000)
            print(f"Downloaded data up to {last_date.strftime('%Y-%m-%d')} ({len(all_data)} candles total)...")
            
            # Rate limiting - be nice to the API
            time.sleep(0.2)
            
        except requests.exceptions.RequestException as e:
            print(f"Error fetching data: {e}")
            break
        except Exception as e:
            print(f"Unexpected error: {e}")
            break
    
    return all_data

def save_to_csv(data: List[List], symbol: str, filename: str = None):
    """
    Save K-line data to CSV file
    
    Args:
        data: List of K-line data from Binance API
        symbol: Trading pair symbol
        filename: Output filename (default: {symbol}_daily_klines.csv)
    """
    if not filename:
        filename = f"{symbol}_daily_klines.csv"
    
    # Binance K-line format:
    # [0] Open time
    # [1] Open price
    # [2] High price
    # [3] Low price
    # [4] Close price
    # [5] Volume
    # [6] Close time
    # [7] Quote asset volume
    # [8] Number of trades
    # [9] Taker buy base asset volume
    # [10] Taker buy quote asset volume
    # [11] Ignore
    
    with open(filename, mode="w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        
        # Write header
        writer.writerow([
            "Open Time",
            "Open Time (UTC)",
            "Open",
            "High",
            "Low",
            "Close",
            "Volume",
            "Close Time",
            "Close Time (UTC)",
            "Quote Asset Volume",
            "Number of Trades",
            "Taker Buy Base Asset Volume",
            "Taker Buy Quote Asset Volume"
        ])
        
        # Write data
        for candle in data:
            open_time_ms = int(candle[0])
            close_time_ms = int(candle[6])
            
            open_time_utc = datetime.datetime.fromtimestamp(open_time_ms / 1000)
            close_time_utc = datetime.datetime.fromtimestamp(close_time_ms / 1000)
            
            writer.writerow([
                open_time_ms,
                open_time_utc.strftime("%Y-%m-%d %H:%M:%S"),
                candle[1],  # Open
                candle[2],  # High
                candle[3],  # Low
                candle[4],  # Close
                candle[5],  # Volume
                close_time_ms,
                close_time_utc.strftime("%Y-%m-%d %H:%M:%S"),
                candle[7],  # Quote asset volume
                candle[8],  # Number of trades
                candle[9],  # Taker buy base asset volume
                candle[10]  # Taker buy quote asset volume
            ])
    
    print(f"\nData saved to {filename}")
    print(f"Total records: {len(data)}")

def main():
    # Configuration
    symbol = "ETHUSDT"  # Change this to your desired trading pair (e.g., ETHUSDT, BTCUSDT)
    start_date = "2020-01-01"
    end_date = None  # None means up to today
    
    # Download data
    data = fetch_daily_klines(symbol, start_date, end_date)
    
    if data:
        # Save to CSV
        save_to_csv(data, symbol)
        print(f"\nSuccessfully downloaded {len(data)} daily candles for {symbol}")
    else:
        print("No data downloaded. Please check your parameters.")

if __name__ == "__main__":
    main()

