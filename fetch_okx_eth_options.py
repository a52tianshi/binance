"""
从OKX交易所拉取ETH期权数据，并按照买一价格对应的波动率由高到低排序
Fetch ETH options from OKX exchange and sort by implied volatility based on bid price
"""

import requests
import json
import math
from typing import List, Dict, Optional
from datetime import datetime
import time


def normal_cdf(x: float) -> float:
    """
    标准正态分布的累积分布函数 (CDF)
    Cumulative Distribution Function of Standard Normal Distribution
    """
    a1 = 0.254829592
    a2 = -0.284496736
    a3 = 1.421413741
    a4 = -1.453152027
    a5 = 1.061405429
    p = 0.3275911

    sign = 1
    if x < 0:
        sign = -1
    x = abs(x) / math.sqrt(2.0)

    t = 1.0 / (1.0 + p * x)
    y = 1.0 - (((((a5 * t + a4) * t) + a3) * t + a2) * t + a1) * t * math.exp(-x * x)

    return 0.5 * (1.0 + sign * y)


def black_scholes_call(S: float, K: float, T: float, r: float, sigma: float) -> float:
    """
    Black-Scholes看涨期权定价公式
    Black-Scholes Call Option Price Formula
    
    Args:
        S: 当前资产价格 Current asset price
        K: 执行价格 Strike price
        T: 到期时间（年）Time to expiration (in years)
        r: 无风险利率 Risk-free interest rate
        sigma: 波动率 Volatility
    Returns:
        看涨期权价格 Call option price
    """
    if T <= 0:
        return max(S - K, 0)
    
    if sigma == 0:
        return max(S - K, 0)
    
    sqrt_T = math.sqrt(T)
    d1 = (math.log(S / K) + (r + 0.5 * sigma ** 2) * T) / (sigma * sqrt_T)
    d2 = d1 - sigma * sqrt_T
    
    call_price = S * normal_cdf(d1) * math.exp(-r * T) - K * normal_cdf(d2) * math.exp(-r * T)
    
    return call_price


def black_scholes_put(S: float, K: float, T: float, r: float, sigma: float) -> float:
    """
    Black-Scholes看跌期权定价公式
    Black-Scholes Put Option Price Formula
    
    Args:
        S: 当前资产价格 Current asset price
        K: 执行价格 Strike price
        T: 到期时间（年）Time to expiration (in years)
        r: 无风险利率 Risk-free interest rate
        sigma: 波动率 Volatility
    Returns:
        看跌期权价格 Put option price
    """
    call_price = black_scholes_call(S, K, T, r, sigma)
    put_price = call_price - S + K * math.exp(-r * T)  # Put-Call Parity
    
    return put_price


def calculate_implied_volatility(
    market_price: float,
    S: float,
    K: float,
    T: float,
    r: float,
    option_type: str = "CALL",
    max_iterations: int = 100,
    tolerance: float = 1e-6
) -> Optional[float]:
    """
    使用二分法计算隐含波动率
    Calculate implied volatility using bisection method
    
    Args:
        market_price: 市场价格 Market price (bid price)
        S: 当前资产价格 Current asset price
        K: 执行价格 Strike price
        T: 到期时间（年）Time to expiration (in years)
        r: 无风险利率 Risk-free interest rate
        option_type: 期权类型 "CALL" 或 "PUT"
        max_iterations: 最大迭代次数
        tolerance: 容差
    
    Returns:
        隐含波动率 Implied volatility, 如果无法计算则返回None
    """
    # 如果到期时间太短（小于1小时），使用最小时间
    min_T = 1.0 / (365.0 * 24.0)  # 1小时
    T = max(T, min_T)
    
    if T <= 0 or market_price <= 0 or S <= 0 or K <= 0:
        return None
    
    # 计算内在价值
    if option_type.upper() == "CALL":
        intrinsic_value = max(S - K, 0)
    else:
        intrinsic_value = max(K - S, 0)
    
    # 如果市场价格小于内在价值，返回None
    if market_price < intrinsic_value:
        return None
    
    # 使用二分法搜索隐含波动率
    # 波动率范围：0.001 到 5.0 (0.1% 到 500%)
    vol_low = 0.001
    vol_high = 5.0
    
    # 计算上下界的期权价格
    if option_type.upper() == "CALL":
        price_low = black_scholes_call(S, K, T, r, vol_low)
        price_high = black_scholes_call(S, K, T, r, vol_high)
    else:
        price_low = black_scholes_put(S, K, T, r, vol_low)
        price_high = black_scholes_put(S, K, T, r, vol_high)
    
    # 如果市场价格不在范围内，调整边界
    if market_price < price_low:
        # 市场价格太低，可能接近内在价值
        return 0.001
    if market_price > price_high:
        # 市场价格太高，可能需要更大的波动率
        vol_high = 10.0
        if option_type.upper() == "CALL":
            price_high = black_scholes_call(S, K, T, r, vol_high)
        else:
            price_high = black_scholes_put(S, K, T, r, vol_high)
    
    # 二分法迭代
    for i in range(max_iterations):
        vol_mid = (vol_low + vol_high) / 2.0
        
        if option_type.upper() == "CALL":
            price_mid = black_scholes_call(S, K, T, r, vol_mid)
        else:
            price_mid = black_scholes_put(S, K, T, r, vol_mid)
        
        error = abs(price_mid - market_price)
        
        if error < tolerance:
            return vol_mid
        
        if price_mid < market_price:
            vol_low = vol_mid
        else:
            vol_high = vol_mid
    
    # 如果迭代未收敛，返回中间值
    return (vol_low + vol_high) / 2.0


def get_eth_spot_price() -> Optional[float]:
    """
    获取ETH现货价格
    Get ETH spot price from OKX
    """
    try:
        # OKX使用ETH-USD作为期权标的，但现货可能是ETH-USDT
        # 先尝试ETH-USDT，如果失败则尝试ETH-USD
        for symbol in ["ETH-USDT", "ETH-USD"]:
            url = f"https://www.okx.com/api/v5/market/ticker?instId={symbol}"
            response = requests.get(url, timeout=10)
            data = response.json()
            
            if data.get("code") == "0" and data.get("data"):
                price = float(data["data"][0]["last"])
                if price > 0:
                    return price
        return None
    except Exception as e:
        print(f"获取ETH现货价格失败: {e}")
        return None


def get_eth_options_instruments() -> List[Dict]:
    """
    获取ETH期权产品列表
    Get ETH options instruments list from OKX
    """
    try:
        url = "https://www.okx.com/api/v5/public/instruments"
        # OKX期权使用ETH-USD作为标的资产
        params = {
            "instType": "OPTION",
            "uly": "ETH-USD"
        }
        
        response = requests.get(url, params=params, timeout=10)
        data = response.json()
        
        if data.get("code") == "0" and data.get("data"):
            return data["data"]
        else:
            print(f"获取期权列表失败: {data}")
            return []
    except Exception as e:
        print(f"获取期权列表异常: {e}")
        return []


def get_option_ticker(inst_id: str) -> Optional[Dict]:
    """
    获取期权ticker数据（包含买一价格）
    Get option ticker data including bid price
    """
    try:
        url = "https://www.okx.com/api/v5/market/ticker"
        params = {"instId": inst_id}
        
        response = requests.get(url, params=params, timeout=10)
        data = response.json()
        
        if data.get("code") == "0" and data.get("data") and len(data["data"]) > 0:
            return data["data"][0]
        return None
    except Exception as e:
        print(f"获取ticker数据失败 {inst_id}: {e}")
        return None


def get_option_mark_price(inst_id: str) -> Optional[Dict]:
    """
    获取期权标记价格
    Get option mark price from OKX API
    
    Args:
        inst_id: 期权合约ID Option instrument ID
    
    Returns:
        包含标记价格的字典，如果失败返回None
    """
    try:
        url = "https://www.okx.com/api/v5/public/mark-price"
        params = {
            "instType": "OPTION",
            "instId": inst_id
        }
        
        response = requests.get(url, params=params, timeout=10)
        data = response.json()
        
        if data.get("code") == "0" and data.get("data") and len(data["data"]) > 0:
            return data["data"][0]
        return None
    except Exception as e:
        print(f"获取标记价格失败 {inst_id}: {e}")
        return None


def get_options_summary(uly: str = "ETH-USD") -> Dict[str, Dict]:
    """
    获取期权汇总数据（包含标记波动率）
    Get options summary data including mark volatility
    
    Args:
        uly: 标的资产 Underlying asset
    
    Returns:
        以instId为key的字典，包含每个期权的汇总数据
    """
    try:
        url = "https://www.okx.com/api/v5/public/opt-summary"
        params = {"uly": uly}
        
        response = requests.get(url, params=params, timeout=10)
        data = response.json()
        
        if data.get("code") == "0" and data.get("data"):
            # 转换为以instId为key的字典，方便查找
            summary_dict = {}
            for item in data["data"]:
                inst_id = item.get("instId")
                if inst_id:
                    summary_dict[inst_id] = item
            return summary_dict
        return {}
    except Exception as e:
        print(f"获取期权汇总数据失败: {e}")
        return {}


def parse_option_symbol(inst_id: str, instrument_data: Dict = None) -> Dict:
    """
    解析期权symbol，提取执行价格、到期时间等信息
    Parse option symbol to extract strike price, expiration time, etc.
    
    OKX期权格式示例: ETH-USD-240329-3000-C
    如果提供了instrument_data，优先使用其中的字段
    """
    try:
        # 如果提供了instrument_data，优先使用其中的字段
        if instrument_data:
            strike_str = instrument_data.get("stk", "")
            option_type = instrument_data.get("optType", "")
            exp_time_ms = instrument_data.get("expTime", "")
            
            if strike_str and option_type and exp_time_ms:
                strike = float(strike_str)
                option_type = option_type.upper()
                
                # 解析到期时间（毫秒时间戳）
                expiry_date = datetime.fromtimestamp(int(exp_time_ms) / 1000)
                
                # 计算剩余时间（使用总秒数，更精确）
                now = datetime.now()
                time_delta = expiry_date - now
                total_seconds = time_delta.total_seconds()
                days_to_expiry = time_delta.days
                
                # 转换为年（使用更精确的计算）
                # 如果剩余时间小于1天，至少使用1小时来计算
                if total_seconds <= 0:
                    time_to_expiry = 0.0
                else:
                    # 使用小时数来计算，至少1小时
                    hours_to_expiry = max(total_seconds / 3600.0, 1.0)
                    time_to_expiry = hours_to_expiry / (365.0 * 24.0)
                
                return {
                    "strike": strike,
                    "expiry_date": expiry_date,
                    "days_to_expiry": days_to_expiry,
                    "time_to_expiry": time_to_expiry,
                    "option_type": option_type
                }
        
        # 如果没有提供instrument_data，则从inst_id解析
        parts = inst_id.split("-")
        if len(parts) >= 5:
            # OKX期权格式: ETH-USD-240329-3000-C 或 ETH-USD-240329-3000-P
            expiry_str = parts[2]  # 240329 (YYMMDD)
            strike_str = parts[3]  # 3000
            option_type = parts[4]  # C or P
            
            # 解析到期时间
            year = 2000 + int(expiry_str[0:2])
            month = int(expiry_str[2:4])
            day = int(expiry_str[4:6])
            expiry_date = datetime(year, month, day)
            
            # 计算剩余时间（使用总秒数，更精确）
            now = datetime.now()
            time_delta = expiry_date - now
            total_seconds = time_delta.total_seconds()
            days_to_expiry = time_delta.days
            
            # 转换为年（使用更精确的计算）
            if total_seconds <= 0:
                time_to_expiry = 0.0
            else:
                # 使用小时数来计算，至少1小时
                hours_to_expiry = max(total_seconds / 3600.0, 1.0)
                time_to_expiry = hours_to_expiry / (365.0 * 24.0)
            
            return {
                "strike": float(strike_str),
                "expiry_date": expiry_date,
                "days_to_expiry": days_to_expiry,
                "time_to_expiry": time_to_expiry,
                "option_type": option_type.upper()
            }
    except Exception as e:
        print(f"解析期权symbol失败 {inst_id}: {e}")
    
    return {
        "strike": 0.0,
        "expiry_date": None,
        "days_to_expiry": 0,
        "time_to_expiry": 0.0,
        "option_type": "UNKNOWN"
    }


def filter_option(spot_price: float, strike: float, option_type: str) -> bool:
    """
    过滤期权：只保留虚值期权和±1%范围内的实值期权
    深度实值期权（执行价远离现货价±1%范围）会被过滤掉
    Filter options: only keep OTM options and ITM options within ±1% of spot price
    Deep ITM options (strike price far from ±1% range) will be filtered out
    
    Args:
        spot_price: 现货价格 Spot price
        strike: 执行价格 Strike price
        option_type: 期权类型 "CALL"/"C" 或 "PUT"/"P"
    
    Returns:
        True if option should be kept, False otherwise
    """
    # 标准化期权类型：C/CALL -> CALL, P/PUT -> PUT
    opt_type_upper = option_type.upper()
    if opt_type_upper in ["C", "CALL"]:
        opt_type = "CALL"
    elif opt_type_upper in ["P", "PUT"]:
        opt_type = "PUT"
    else:
        # 未知类型，默认过滤掉
        return False
    
    if opt_type == "CALL":
        # 看涨期权
        # 虚值：执行价 > 现货价（保留所有虚值期权）
        is_otm = strike > spot_price
        # 实值：执行价 <= 现货价，且执行价 >= 现货价 * 0.99 (在-1%范围内，包括ATM)
        # 例如：现货3090，只保留执行价在 [3059.1, 3090] 范围内的实值期权
        # 执行价600等深度实值期权会被过滤掉
        is_itm_near = strike <= spot_price and strike >= spot_price * 0.99
        return is_otm or is_itm_near
    else:  # PUT
        # 看跌期权
        # 虚值：执行价 < 现货价（保留所有虚值期权）
        is_otm = strike < spot_price
        # 实值：执行价 >= 现货价，且执行价 <= 现货价 * 1.01 (在+1%范围内，包括ATM)
        # 例如：现货3090，只保留执行价在 [3090, 3120.9] 范围内的实值期权
        # 执行价远高于3120.9的深度实值期权会被过滤掉
        is_itm_near = strike >= spot_price and strike <= spot_price * 1.01
        return is_otm or is_itm_near


def fetch_all_eth_options() -> List[Dict]:
    """
    获取所有ETH期权数据并使用API返回的买一价波动率（bidVol）
    Fetch all ETH options and use bid volatility from API
    """
    print("正在获取ETH现货价格...")
    spot_price = get_eth_spot_price()
    if spot_price is None:
        print("无法获取ETH现货价格，退出")
        return []
    
    print(f"ETH现货价格: ${spot_price:.2f}")
    
    print("\n正在获取ETH期权列表...")
    instruments = get_eth_options_instruments()
    print(f"找到 {len(instruments)} 个期权产品")
    
    if not instruments:
        return []
    
    # 获取期权汇总数据（包含标记波动率）
    print("\n正在获取期权汇总数据（包含标记波动率）...")
    options_summary = get_options_summary("ETH-USD")
    print(f"获取到 {len(options_summary)} 个期权的汇总数据")
    
    options_data = []
    
    print("\n正在处理期权数据...")
    for i, instrument in enumerate(instruments):
        inst_id = instrument.get("instId", "")
        if not inst_id:
            continue
        
        # 解析期权信息（优先使用instrument中的字段）
        option_info = parse_option_symbol(inst_id, instrument)
        
        # 过滤期权：只保留虚值期权和±1%范围内的实值期权
        # 深度实值期权（执行价远离现货价±1%范围）会被过滤掉
        if not filter_option(spot_price, option_info["strike"], option_info["option_type"]):
            continue
        
        # 从汇总数据中获取买一价波动率
        summary = options_summary.get(inst_id)
        if not summary:
            continue
        
        # 获取买一价对应的波动率（bidVol）
        bid_vol_str = summary.get("bidVol", "")
        if not bid_vol_str or bid_vol_str == "":
            continue
        
        try:
            # bidVol是小数形式（如0.725表示0.725%），需要转换为小数形式存储
            # 显示时需要乘以100才能带%符号
            bid_vol_value = float(bid_vol_str)
            if bid_vol_value <= 0:
                continue
            # 存储为小数形式（0.725 -> 0.00725），显示时乘以100得到百分比
            implied_vol = bid_vol_value / 100.0
        except:
            continue
        
        # 获取标记价格（从mark-price API获取）
        mark_price_data = get_option_mark_price(inst_id)
        if not mark_price_data:
            continue
        
        mark_price_str = mark_price_data.get("markPx", "0")
        if not mark_price_str or mark_price_str == "":
            continue
        
        try:
            mark_price = float(mark_price_str)
        except:
            continue
        
        if mark_price <= 0:
            continue
        
        # 检查买一价：如果没有买一价，则跳过该期权
        ticker = get_option_ticker(inst_id)
        if not ticker:
            continue
        
        bid_price_str = ticker.get("bidPx", "")
        if not bid_price_str or bid_price_str == "":
            continue
        
        try:
            bid_price = float(bid_price_str)
        except:
            continue
        
        if bid_price <= 0:
            continue
        
        # 剔除买一价为0.000100的期权（可能是无效报价）
        if abs(bid_price - 0.000100) < 1e-9:
            continue
        
        # 判断期权是虚值还是实值
        if option_info["option_type"] == "CALL":
            is_itm = option_info["strike"] < spot_price
        else:  # PUT
            is_itm = option_info["strike"] > spot_price
        
        moneyness = "ITM" if is_itm else "OTM"
        
        # 收集数据
        option_data = {
            "inst_id": inst_id,
            "strike": option_info["strike"],
            "expiry_date": option_info["expiry_date"],
            "days_to_expiry": option_info["days_to_expiry"],
            "time_to_expiry": option_info["time_to_expiry"],
            "option_type": option_info["option_type"],
            "mark_price": mark_price,
            "bid_price": bid_price,
            "implied_volatility": implied_vol,
            "spot_price": spot_price,
            "moneyness": moneyness
        }
        
        options_data.append(option_data)
        
        # 显示进度
        if (i + 1) % 10 == 0:
            print(f"已处理 {i + 1}/{len(instruments)} 个期权...")
        
        # 避免请求过快
        time.sleep(0.1)
    
    print(f"\n成功获取 {len(options_data)} 个有效期权数据")
    print("过滤条件：")
    print("  - 仅保留虚值期权和±1%范围内的实值期权")
    print("  - 必须有买一价（bidPx）")
    print("  - 剔除买一价为0.000100的期权（无效报价）")
    
    return options_data


def print_options_table(options_data: List[Dict]):
    """
    打印期权数据表格
    Print options data in table format
    """
    if not options_data:
        print("没有期权数据")
        return
    
    # 按隐含波动率排序（从高到低）
    sorted_options = sorted(options_data, key=lambda x: x["implied_volatility"], reverse=True)
    
    print("\n" + "=" * 180)
    print("ETH期权数据 - 按买一价波动率排序（从高到低）")
    print("波动率来源：OKX API返回的买一价波动率（bidVol）")
    print("过滤条件：仅显示虚值期权和±1%范围内的实值期权，必须有买一价，剔除买一价为0.000100的期权")
    print("=" * 180)
    print(f"{'期权ID':<30} | {'类型':<6} | {'虚实':<4} | {'执行价':>10} | {'到期天数':>8} | {'现货价':>10} | {'标记价格':>12} | {'买一价':>12} | {'隐含波动率':>12}")
    print("-" * 180)
    
    for opt in sorted_options:
        expiry_str = opt["expiry_date"].strftime("%Y-%m-%d") if opt["expiry_date"] else "N/A"
        inst_id_short = opt["inst_id"][-20:] if len(opt["inst_id"]) > 20 else opt["inst_id"]
        
        print(
            f"{inst_id_short:<30} | "
            f"{opt['option_type']:<6} | "
            f"{opt.get('moneyness', 'N/A'):<4} | "
            f"{opt['strike']:>10.2f} | "
            f"{opt['days_to_expiry']:>8} | "
            f"{opt['spot_price']:>10.2f} | "
            f"{opt['mark_price']:>12.6f} | "
            f"{opt.get('bid_price', 0):>12.6f} | "
            f"{opt['implied_volatility']*100:>11.2f}%"
        )
    
    print("=" * 180)
    print(f"\n总计: {len(sorted_options)} 个期权")


def save_to_csv(options_data: List[Dict], filename: str = "okx_eth_options.csv"):
    """
    保存期权数据到CSV文件
    Save options data to CSV file
    """
    if not options_data:
        print("没有数据可保存")
        return
    
    import csv
    
    # 按隐含波动率排序
    sorted_options = sorted(options_data, key=lambda x: x["implied_volatility"], reverse=True)
    
    with open(filename, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        
        # 写入表头
        writer.writerow([
            "期权ID", "类型", "虚实", "执行价", "到期日期", "到期天数", "时间到期(年)",
            "现货价", "标记价格", "买一价", "隐含波动率(%)"
        ])
        
        # 写入数据
        for opt in sorted_options:
            expiry_str = opt["expiry_date"].strftime("%Y-%m-%d") if opt["expiry_date"] else "N/A"
            writer.writerow([
                opt["inst_id"],
                opt["option_type"],
                opt.get("moneyness", "N/A"),
                opt["strike"],
                expiry_str,
                opt["days_to_expiry"],
                f"{opt['time_to_expiry']:.6f}",
                opt["spot_price"],
                opt["mark_price"],
                opt.get("bid_price", 0),
                opt["implied_volatility"] * 100
            ])
    
    print(f"\n数据已保存到: {filename}")


def main():
    """
    主函数
    Main function
    """
    print("=" * 80)
    print("OKX ETH期权数据获取工具")
    print("=" * 80)
    
    # 获取所有ETH期权数据
    options_data = fetch_all_eth_options()
    
    if not options_data:
        print("未能获取到任何期权数据")
        return
    
    # 打印表格
    print_options_table(options_data)
    
    # 保存到CSV
    save_to_csv(options_data)
    
    print("\n完成！")


if __name__ == "__main__":
    main()

