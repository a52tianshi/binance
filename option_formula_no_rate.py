"""
期权定价公式 - 无利率情况 (r = 0)
假设标准差 σ = 3

Option Pricing Formulas - Zero Interest Rate Case (r = 0)
Assuming Standard Deviation σ = 3
"""

import math
from typing import List, Tuple

def normal_cdf(x: float) -> float:
    """
    标准正态分布的累积分布函数 (CDF)
    Cumulative Distribution Function of Standard Normal Distribution
    """
    a1 =  0.254829592
    a2 = -0.284496736
    a3 =  1.421413741
    a4 = -1.453152027
    a5 =  1.061405429
    p  =  0.3275911

    sign = 1
    if x < 0:
        sign = -1
    x = abs(x) / math.sqrt(2.0)

    t = 1.0 / (1.0 + p * x)
    y = 1.0 - (((((a5 * t + a4) * t) + a3) * t + a2) * t + a1) * t * math.exp(-x * x)

    return 0.5 * (1.0 + sign * y)

def normal_pdf(x: float) -> float:
    """
    标准正态分布的概率密度函数 (PDF)
    Probability Density Function of Standard Normal Distribution
    """
    return (1.0 / math.sqrt(2.0 * math.pi)) * math.exp(-0.5 * x * x)

def black_scholes_call_no_rate(S: float, K: float, T: float, sigma: float) -> float:
    """
    无利率情况下的Black-Scholes看涨期权定价公式
    
    Black-Scholes Call Option Price Formula (r = 0)
    
    公式 Formula:
    C = S × Φ(d1) - K × Φ(d2)
    
    其中 where:
    d1 = [ln(S/K) + (σ²/2) × T] / (σ × √T)
    d2 = d1 - σ × √T
    
    当 r = 0 时，公式简化为:
    When r = 0, formula simplifies to:
    C = S × Φ(d1) - K × Φ(d2)
    
    Args:
        S: 当前资产价格 Current asset price
        K: 执行价格 Strike price
        T: 到期时间（年）Time to expiration (in years)
        sigma: 波动率/标准差 Volatility/Standard deviation
    
    Returns:
        看涨期权价格 Call option price
    """
    if T <= 0:
        return max(S - K, 0)
    
    if sigma == 0:
        return max(S - K, 0)
    
    sqrt_T = math.sqrt(T)
    d1 = (math.log(S / K) + 0.5 * sigma ** 2 * T) / (sigma * sqrt_T)
    d2 = d1 - sigma * sqrt_T
    
    call_price = S * normal_cdf(d1) - K * normal_cdf(d2)
    
    return call_price

def black_scholes_put_no_rate(S: float, K: float, T: float, sigma: float) -> float:
    """
    无利率情况下的Black-Scholes看跌期权定价公式
    
    Black-Scholes Put Option Price Formula (r = 0)
    
    公式 Formula (Put-Call Parity):
    P = C - S + K
    
    当 r = 0 时:
    When r = 0:
    P = C - S + K
    
    Args:
        S: 当前资产价格 Current asset price
        K: 执行价格 Strike price
        T: 到期时间（年）Time to expiration (in years)
        sigma: 波动率/标准差 Volatility/Standard deviation
    
    Returns:
        看跌期权价格 Put option price
    """
    call_price = black_scholes_call_no_rate(S, K, T, sigma)
    put_price = call_price - S + K  # 无利率时的看跌看涨平价
    
    return put_price

def expected_call_intrinsic_normal(S0: float, K: float, T: float, sigma: float) -> float:
    """
    预期看涨期权内在价值（假设价格服从正态分布）
    
    Expected Call Option Intrinsic Value (assuming price follows normal distribution)
    
    公式 Formula:
    E[max(S_T - K, 0)] = (μ - K) × Φ((μ-K)/σ) + σ × φ((μ-K)/σ)
    
    其中 where:
    μ = S0 (当前价格，无漂移)
    σ = S0 × σ × √T (标准差)
    
    Args:
        S0: 当前资产价格 Current asset price
        K: 执行价格 Strike price
        T: 到期时间（年）Time to expiration (in years)
        sigma: 波动率 Volatility (annual)
    
    Returns:
        预期看涨期权内在价值 Expected call intrinsic value
    """
    if T <= 0:
        return max(S0 - K, 0)
    
    mu = S0  # 均值保持在当前价格（无漂移，因为r=0）
    sigma_price = S0 * sigma * math.sqrt(T)  # 价格的标准差
    
    if sigma_price == 0:
        return max(mu - K, 0)
    
    d = (mu - K) / sigma_price
    
    # 预期内在价值公式
    expected_value = (mu - K) * normal_cdf(d) + sigma_price * normal_pdf(d)
    
    return max(expected_value, 0)

def expected_put_intrinsic_normal(S0: float, K: float, T: float, sigma: float) -> float:
    """
    预期看跌期权内在价值（假设价格服从正态分布）
    
    Expected Put Option Intrinsic Value (assuming price follows normal distribution)
    
    公式 Formula:
    E[max(K - S_T, 0)] = (K - μ) × Φ((K-μ)/σ) + σ × φ((K-μ)/σ)
    
    其中 where:
    μ = S0 (当前价格，无漂移)
    σ = S0 × σ × √T (标准差)
    
    Args:
        S0: 当前资产价格 Current asset price
        K: 执行价格 Strike price
        T: 到期时间（年）Time to expiration (in years)
        sigma: 波动率 Volatility (annual)
    
    Returns:
        预期看跌期权内在价值 Expected put intrinsic value
    """
    if T <= 0:
        return max(K - S0, 0)
    
    mu = S0  # 均值保持在当前价格（无漂移，因为r=0）
    sigma_price = S0 * sigma * math.sqrt(T)  # 价格的标准差
    
    if sigma_price == 0:
        return max(K - mu, 0)
    
    d = (K - mu) / sigma_price
    
    # 预期内在价值公式
    expected_value = (K - mu) * normal_cdf(d) + sigma_price * normal_pdf(d)
    
    return max(expected_value, 0)

def print_formulas():
    """
    打印所有期权定价公式
    Print all option pricing formulas
    """
    print("="*90)
    print("期权定价公式 - 无利率情况 (r = 0)")
    print("Option Pricing Formulas - Zero Interest Rate Case (r = 0)")
    print("="*90)
    
    print("\n【配置参数 Configuration】")
    print("  r = 0 (无风险利率 Risk-free interest rate = 0)")
    print("  σ = 3 (标准差/波动率 Standard deviation/Volatility = 3)")
    
    print("\n【1. Black-Scholes 看涨期权公式 Call Option Formula】")
    print("  C = S × Φ(d1) - K × Φ(d2)")
    print("  其中 where:")
    print("    d1 = [ln(S/K) + (σ²/2) × T] / (σ × √T)")
    print("    d2 = d1 - σ × √T")
    print("    Φ = 标准正态分布的累积分布函数 (CDF of standard normal)")
    
    print("\n【2. Black-Scholes 看跌期权公式 Put Option Formula】")
    print("  P = C - S + K  (看跌看涨平价 Put-Call Parity, r = 0时)")
    
    print("\n【3. 预期看涨期权内在价值 Expected Call Intrinsic Value】")
    print("  假设价格服从正态分布: S_T ~ N(μ, σ²)")
    print("  Assuming price follows normal distribution: S_T ~ N(μ, σ²)")
    print("  公式 Formula:")
    print("    E[max(S_T - K, 0)] = (μ - K) × Φ((μ-K)/σ) + σ × φ((μ-K)/σ)")
    print("  其中 where:")
    print("    μ = S0 (当前价格)")
    print("    σ = S0 × σ × √T (价格标准差)")
    print("    φ = 标准正态分布的概率密度函数 (PDF of standard normal)")
    
    print("\n【4. 预期看跌期权内在价值 Expected Put Intrinsic Value】")
    print("  假设价格服从正态分布: S_T ~ N(μ, σ²)")
    print("  公式 Formula:")
    print("    E[max(K - S_T, 0)] = (K - μ) × Φ((K-μ)/σ) + σ × φ((K-μ)/σ)")
    
    print("\n【符号说明 Notation】")
    print("  S  = 当前资产价格 (Current asset price)")
    print("  K  = 执行价格 (Strike price)")
    print("  T  = 到期时间，年 (Time to expiration, in years)")
    print("  σ  = 波动率/标准差 (Volatility/Standard deviation)")
    print("  r  = 无风险利率 (Risk-free interest rate) = 0")
    print("  C  = 看涨期权价格 (Call option price)")
    print("  P  = 看跌期权价格 (Put option price)")
    print("  S_T = 到期时的资产价格 (Asset price at expiration)")
    print("  Φ  = 标准正态分布的累积分布函数 (CDF of standard normal)")
    print("  φ  = 标准正态分布的概率密度函数 (PDF of standard normal)")
    
    print("="*90)

def calculate_example():
    """
    计算示例
    Calculate example
    """
    # 参数设置 Parameters
    S0 = 100.0      # 当前价格
    sigma_abs = 3.0  # 绝对标准差 σ = 3 (价格的标准差)
    sigma_rel = 0.03  # 相对波动率 3% (用于Black-Scholes)
    T = 1.0         # 到期时间 1年
    K = 100.0       # 执行价格
    
    # 注意：标准差是3可以理解为：
    # 1. 绝对标准差：σ_price = 3 (价格的标准差直接是3)
    # 2. 相对波动率：σ = 3% = 0.03 (年化波动率)
    # 这里我们两种都计算，但主要使用绝对标准差 σ = 3
    
    print("\n" + "="*90)
    print("计算示例 Example Calculation")
    print("="*90)
    print(f"\n参数 Parameters:")
    print(f"  S0 (当前价格) = {S0}")
    print(f"  K (执行价格) = {K}")
    print(f"  T (到期时间) = {T} 年")
    print(f"  σ (绝对标准差) = {sigma_abs}")
    print(f"  σ (相对波动率) = {sigma_rel} ({sigma_rel*100}%)")
    print(f"\n  假设价格分布: S_T ~ N(μ = {S0}, σ² = {sigma_abs**2})")
    
    # Black-Scholes期权价格 (使用相对波动率)
    call_price = black_scholes_call_no_rate(S0, K, T, sigma_rel)
    put_price = black_scholes_put_no_rate(S0, K, T, sigma_rel)
    
    print(f"\n【Black-Scholes 期权价格 Option Prices (使用相对波动率)】")
    print(f"  看涨期权价格 Call Price (C) = {call_price:.6f}")
    print(f"  看跌期权价格 Put Price (P)  = {put_price:.6f}")
    
    # 预期内在价值 (使用绝对标准差 σ = 3)
    # 对于预期内在价值，我们直接使用绝对标准差 σ_price = 3
    # 需要调整函数以接受绝对标准差
    
    # 计算预期内在价值，假设 σ_price = 3
    mu = S0
    sigma_price = sigma_abs  # 直接使用绝对标准差 3
    
    # 手动计算预期内在价值
    d_call = (mu - K) / sigma_price
    expected_call_iv = (mu - K) * normal_cdf(d_call) + sigma_price * normal_pdf(d_call)
    expected_call_iv = max(expected_call_iv, 0)
    
    d_put = (K - mu) / sigma_price
    expected_put_iv = (K - mu) * normal_cdf(d_put) + sigma_price * normal_pdf(d_put)
    expected_put_iv = max(expected_put_iv, 0)
    
    print(f"\n【预期内在价值 Expected Intrinsic Values (使用绝对标准差 σ = {sigma_abs})】")
    print(f"  价格分布: S_T ~ N(μ = {mu}, σ = {sigma_price})")
    print(f"  预期看涨内在价值 Expected Call IV = {expected_call_iv:.6f}")
    print(f"  预期看跌内在价值 Expected Put IV  = {expected_put_iv:.6f}")
    
    # 计算多个执行价格的期权价格
    print(f"\n【不同执行价格的期权价格 Option Prices for Different Strikes】")
    print(f"{'K':>10} | {'Call Price':>15} | {'Put Price':>15} | {'Call E[IV]':>15} | {'Put E[IV]':>15}")
    print("-" * 90)
    
    mu = S0
    sigma_price = sigma_abs  # 使用绝对标准差 3
    
    for strike in [95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105]:
        # Black-Scholes价格（使用相对波动率）
        c = black_scholes_call_no_rate(S0, strike, T, sigma_rel)
        p = black_scholes_put_no_rate(S0, strike, T, sigma_rel)
        
        # 预期内在价值（使用绝对标准差 σ = 3）
        d_call = (mu - strike) / sigma_price
        ec_iv = (mu - strike) * normal_cdf(d_call) + sigma_price * normal_pdf(d_call)
        ec_iv = max(ec_iv, 0)
        
        d_put = (strike - mu) / sigma_price
        ep_iv = (strike - mu) * normal_cdf(d_put) + sigma_price * normal_pdf(d_put)
        ep_iv = max(ep_iv, 0)
        
        print(f"{strike:>10.0f} | {c:>15.6f} | {p:>15.6f} | {ec_iv:>15.6f} | {ep_iv:>15.6f}")
    
    print("="*90)

def main():
    """
    主函数
    Main function
    """
    # 打印所有公式
    print_formulas()
    
    # 计算示例
    calculate_example()

if __name__ == "__main__":
    main()

