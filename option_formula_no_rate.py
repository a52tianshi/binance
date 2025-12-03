"""
期权定价公式 - 无利率情况 (r = 0)
假设标准差 σ = 3

Option Pricing Formulas - Zero Interest Rate Case (r = 0)
Assuming Standard Deviation σ = 3
"""

import math
from typing import List, Tuple, Callable
import numpy as np

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

def simpson_rule(f: Callable, a: float, b: float, n: int = 1000) -> float:
    """
    Simpson规则数值积分
    
    Simpson's Rule for Numerical Integration
    
    Args:
        f: 被积函数 Function to integrate
        a: 积分下限 Lower bound
        b: 积分上限 Upper bound
        n: 分段数（必须是偶数）Number of subintervals (must be even)
    
    Returns:
        积分值 Integral value
    """
    if n % 2 != 0:
        n += 1  # 确保n是偶数
    
    h = (b - a) / n
    x = np.linspace(a, b, n + 1)
    y = np.array([f(xi) for xi in x])
    
    # Simpson规则公式
    integral = h / 3.0 * (
        y[0] +  # f(a)
        y[-1] +  # f(b)
        4 * np.sum(y[1:-1:2]) +  # 奇数索引项
        2 * np.sum(y[2:-1:2])  # 偶数索引项（除了首尾）
    )
    
    return integral

def expected_call_price_at_half_T_numerical(S0: float, K: float, T: float, sigma_abs: float, n_points: int = 2000):
    """
    使用数值积分计算在T/2时刻call_price的期望值（不使用蒙特卡洛）
    
    计算方法：
    E[C(S_t, K, T/2)] = ∫_{-∞}^{+∞} C(S_t, K, T/2) × f(S_t) dS_t
    
    其中：
    - f(S_t) 是 S_t 的概率密度函数（正态分布）
    - S_t ~ N(S0, σ² × T/2)
    - C(S_t, K, T/2) 是剩余T/2时间的call price
    
    Calculate Expected Call Price at Half-Time using Numerical Integration (not Monte Carlo)
    
    Args:
        S0: 初始资产价格 Initial asset price
        K: 执行价格 Strike price
        T: 总到期时间 Total time to expiration (in years)
        sigma_abs: 绝对标准差 Absolute standard deviation
        n_points: 数值积分的分段数 Number of integration points
    
    Returns:
        期望call价格 Expected call price
    """
    t = T / 2.0  # 已过时间：半个T
    remaining_T = T / 2.0  # 剩余到期时间：半个T
    
    # 在t=T/2时刻，价格分布参数
    mean_at_t = S0
    variance_at_t = sigma_abs ** 2 * t  # σ² × t
    std_at_t = math.sqrt(variance_at_t)
    
    # 转换为相对波动率（用于Black-Scholes公式）
    sigma_rel = sigma_abs / S0
    
    # 定义被积函数
    def integrand(S_t):
        """
        被积函数：C(S_t, K, T/2) × f(S_t)
        f(S_t) 是正态分布的PDF
        """
        if S_t <= 0:
            return 0.0
        
        # 计算剩余时间的call price
        call_price = black_scholes_call_no_rate(S_t, K, remaining_T, sigma_rel)
        
        # 正态分布的PDF
        pdf = normal_pdf((S_t - mean_at_t) / std_at_t) / std_at_t
        
        return call_price * pdf
    
    # 数值积分区间：从mean-6*std到mean+6*std（覆盖99.99%的概率）
    a = max(0.01, mean_at_t - 6 * std_at_t)  # 下界，确保价格>0
    b = mean_at_t + 6 * std_at_t  # 上界
    
    # 使用Simpson规则进行数值积分
    result = simpson_rule(integrand, a, b, n=n_points)
    
    return result

def calculate_call_price_for_different_T():
    """
    对于不同的T值（0.1到0.9），计算执行价格105在T/2时刻的期望call price
    使用数值积分方法
    
    Calculate Expected Call Price at T/2 for different T values (0.1 to 0.9)
    Using numerical integration method
    """
    S0 = 100.0      # 初始价格
    K = 105.0       # 执行价格
    sigma_abs = 3.0  # 绝对标准差 3
    
    print("\n" + "="*90)
    print("不同T值下，在T/2时刻执行价格105的期望call price（数值积分方法）")
    print("Expected Call Price at T/2 for Strike 105 with Different T Values")
    print("Using Numerical Integration Method")
    print("="*90)
    
    print(f"\n【参数 Parameters】")
    print(f"  初始价格 S0 = {S0}")
    print(f"  执行价格 K = {K}")
    print(f"  绝对标准差 σ = {sigma_abs}")
    
    print(f"\n【结果 Results】")
    print(f"{'T':>8} | {'t=T/2':>10} | {'σ_t':>12} | {'Current Call':>15} | {'Exp Call at T/2':>18} | {'Difference':>15}")
    print("-" * 90)
    
    results = []
    
    # 转换为相对波动率
    sigma_rel = sigma_abs / S0
    
    for T in [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9]:
        # 当前时刻的call price
        current_call = black_scholes_call_no_rate(S0, K, T, sigma_rel)
        
        # 在T/2时刻的期望call price（使用数值积分）
        t = T / 2.0
        std_at_t = sigma_abs * math.sqrt(t)
        expected_call = expected_call_price_at_half_T_numerical(S0, K, T, sigma_abs)
        
        diff = expected_call - current_call
        
        print(f"{T:>8.1f} | {t:>10.3f} | {std_at_t:>12.6f} | {current_call:>15.6f} | {expected_call:>18.6f} | {diff:>15.6f}")
        
        results.append((T, t, std_at_t, current_call, expected_call, diff))
    
    print("="*90)
    
    # 总结
    print(f"\n【总结 Summary】")
    print(f"  对于执行价格 K = {K}：")
    print(f"  - 使用数值积分方法计算期望值")
    print(f"  - 在t=T/2时刻，价格分布：S_t ~ N({S0}, σ² × T/2)")
    print(f"  - 对每个T值，计算剩余T/2时间的call price期望值")
    
    return results

def expected_call_price_at_half_T(S0: float, K: float, T: float, sigma_abs: float, num_samples: int = 10000, verbose: bool = True):
    """
    计算在T/2时刻call_price的期望值
    
    假设：
    - 价格按照标准差σ随机波动（符合正态分布）
    - 在t=T/2时刻，价格S_{T/2} ~ N(S0, σ² × T/2)
    - 对于每个可能的价格，计算剩余T/2时间的call option价格
    - 计算所有这些价格的期望值
    
    Expected Call Price at Half-Time (t = T/2)
    
    Assumptions:
    - Price follows normal distribution with standard deviation σ
    - At time t=T/2, price S_{T/2} ~ N(S0, σ² × T/2)
    - For each possible price, calculate call option price with remaining time T/2
    - Calculate expected value of all these prices
    
    Args:
        S0: 初始资产价格 Initial asset price
        K: 执行价格 Strike price
        T: 总到期时间 Total time to expiration (in years)
        sigma_abs: 绝对标准差 Absolute standard deviation
        num_samples: 蒙特卡洛模拟样本数 Number of Monte Carlo samples
    
    Returns:
        期望call价格 Expected call price
    """
    t = T / 2.0  # 已过时间：半个T
    remaining_T = T / 2.0  # 剩余到期时间：半个T
    
    # 在t=T/2时刻，价格的标准差
    sigma_at_t = sigma_abs * math.sqrt(t)
    # sigma_at_t = sigma_abs * math.sqrt(t) if we assume σ is annual volatility
    # But if σ is absolute std dev per unit time, then at time t, std dev = σ × √t
    
    # 价格分布：S_{T/2} ~ N(S0, σ² × T/2)
    mean_at_t = S0
    variance_at_t = sigma_abs ** 2 * t  # σ² × t
    std_at_t = math.sqrt(variance_at_t)
    
    if verbose:
        print(f"\n【在t=T/2时刻的价格分布 Price Distribution at t = T/2】")
        print(f"  已过时间 t = T/2 = {t:.4f} 年")
        print(f"  剩余时间 T_remaining = T/2 = {remaining_T:.4f} 年")
        print(f"  价格分布: S_{{{t:.4f}}} ~ N(μ = {mean_at_t:.2f}, σ² = {variance_at_t:.6f})")
        print(f"  标准差: σ_t = σ × √t = {sigma_abs} × √{t:.4f} = {std_at_t:.6f}")
    
    # 使用数值积分计算期望值
    # 方法1：蒙特卡洛模拟
    np.random.seed(42)  # 为了可重复性
    price_samples = np.random.normal(mean_at_t, std_at_t, num_samples)
    
    call_prices = []
    for S_t in price_samples:
        # 对于每个可能的价格S_t，计算剩余时间的call price
        # 需要将绝对标准差转换为相对波动率
        # 假设sigma_abs是绝对标准差，我们需要相对波动率来计算BS公式
        
        # 如果sigma_abs是绝对标准差，我们需要找到一个相对波动率
        # 使得 S0 × σ_rel × √(T/2) = sigma_abs × √(T/2)
        # 即 σ_rel = sigma_abs / S0
        
        if S_t <= 0:
            call_price = 0
        else:
            # 使用相对波动率
            sigma_rel = sigma_abs / S0  # 将绝对标准差转换为相对波动率
            
            # 计算剩余T/2时间的call price
            call_price = black_scholes_call_no_rate(S_t, K, remaining_T, sigma_rel)
        
        call_prices.append(call_price)
    
    expected_call = np.mean(call_prices)
    
    call_prices_array = np.array(call_prices)
    
    # 计算当前时刻的call price（总是需要）
    sigma_rel = sigma_abs / S0
    current_call = black_scholes_call_no_rate(S0, K, T, sigma_rel)
    
    if verbose:
        # 计算一些统计信息
        std_call = np.std(call_prices_array)
        min_call = np.min(call_prices_array)
        max_call = np.max(call_prices_array)
        
        print(f"\n【计算结果 Results】")
        print(f"  蒙特卡洛模拟样本数: {num_samples}")
        print(f"  在t=T/2时刻的期望call价格: E[C(S_{{{t:.4f}}}, K, T/2)] = {expected_call:.6f}")
        
        print(f"\n【统计信息 Statistics】")
        print(f"  期望值 Expected value: {expected_call:.6f}")
        print(f"  标准差 Standard deviation: {std_call:.6f}")
        print(f"  最小值 Minimum: {min_call:.6f}")
        print(f"  最大值 Maximum: {max_call:.6f}")
        
        print(f"\n【对比 Comparison】")
        print(f"  当前时刻(t=0)的call价格: C(S0, K, T) = {current_call:.6f}")
        print(f"  在t=T/2时刻的期望call价格: E[C(S_{{{t:.4f}}}, K, T/2)] = {expected_call:.6f}")
        print(f"  差异 Difference: {expected_call - current_call:.6f}")
    
    return expected_call, current_call, call_prices_array

def example_expected_call_at_half_T():
    """
    举例：计算在T/2时刻call_price的期望值
    Example: Expected Call Price at Half-Time
    """
    S0 = 100.0      # 初始价格
    K = 105.0       # 执行价格
    T = 1.0         # 总到期时间 1年
    sigma_abs = 3.0  # 绝对标准差 3
    
    print("\n" + "="*90)
    print("举例：在T/2时刻call_price的期望值")
    print("Example: Expected Call Price at Half-Time (t = T/2)")
    print("="*90)
    
    print(f"\n【参数 Parameters】")
    print(f"  初始价格 S0 = {S0}")
    print(f"  执行价格 K = {K}")
    print(f"  总到期时间 T = {T} 年")
    print(f"  绝对标准差 σ = {sigma_abs}")
    print(f"  时间点 t = T/2 = {T/2:.4f} 年")
    
    # 计算期望call price
    expected_call, current_call, call_prices = expected_call_price_at_half_T(
        S0, K, T, sigma_abs, num_samples=50000
    )
    
    print("\n" + "="*90)
    
    # 计算不同执行价格的期望值
    print(f"\n【不同执行价格的期望call价格】")
    print(f"{'K':>10} | {'Current Call':>15} | {'Exp Call at T/2':>18} | {'Difference':>15}")
    print("-" * 90)
    
    sigma_rel = sigma_abs / S0
    
    for strike in [95, 100, 105, 110]:
        current_c = black_scholes_call_no_rate(S0, strike, T, sigma_rel)
        exp_c, _, _ = expected_call_price_at_half_T(S0, strike, T, sigma_abs, num_samples=10000, verbose=False)
        diff = exp_c - current_c
        print(f"{strike:>10.0f} | {current_c:>15.6f} | {exp_c:>18.6f} | {diff:>15.6f}")
    
    print("="*90)

def example_sell_option():
    """
    举例说明：卖出执行价格为105的期权
    Example: Selling an option with strike price 105
    """
    S0 = 100.0      # 当前价格
    K = 105.0       # 执行价格
    T = 1.0         # 到期时间 1年
    sigma_rel = 0.03  # 相对波动率 3%
    sigma_abs = 3.0   # 绝对标准差 3
    
    print("\n" + "="*90)
    print("举例：卖出执行价格为105的期权")
    print("Example: Selling an Option with Strike Price 105")
    print("="*90)
    
    print(f"\n【市场情况 Market Situation】")
    print(f"  当前价格 S₀ = {S0}")
    print(f"  执行价格 K = {K}")
    print(f"  到期时间 T = {T} 年")
    print(f"  标准差 σ = {sigma_abs}")
    
    # 计算看涨期权和看跌期权价格
    call_price = black_scholes_call_no_rate(S0, K, T, sigma_rel)
    put_price = black_scholes_put_no_rate(S0, K, T, sigma_rel)
    
    print(f"\n【期权价格 Option Prices】")
    print(f"  执行价格105的看涨期权价格 Call Price (K=105) = {call_price:.6f}")
    print(f"  执行价格105的看跌期权价格 Put Price (K=105)  = {put_price:.6f}")
    
    # 当前的内在价值
    current_call_iv = max(S0 - K, 0)  # max(100 - 105, 0) = 0
    current_put_iv = max(K - S0, 0)   # max(105 - 100, 0) = 5
    
    print(f"\n【当前内在价值 Current Intrinsic Value】")
    print(f"  看涨期权当前内在价值 = max({S0} - {K}, 0) = {current_call_iv}")
    print(f"  看跌期权当前内在价值 = max({K} - {S0}, 0) = {current_put_iv}")
    
    print(f"\n【期权状态 Option Status】")
    if current_call_iv == 0:
        print(f"  看涨期权（K={K}）是虚值期权（OTM - Out of the Money）")
        print(f"  因为当前价格 {S0} < 执行价格 {K}")
    if current_put_iv > 0:
        print(f"  看跌期权（K={K}）是实值期权（ITM - In the Money）")
        print(f"  因为当前价格 {S0} < 执行价格 {K}")
    
    print(f"\n【卖出期权的情景 Selling Options】")
    print(f"  " + "-"*88)
    
    # 场景1：卖出看涨期权
    print(f"\n【场景1：卖出看涨期权 Short Call (K={K})】")
    print(f"  如果你卖出1份执行价格为{K}的看涨期权：")
    print(f"  你会收到期权费（Premium）= {call_price:.6f}")
    print(f"  是的，你说得对！价值不高，只能收到约 {call_price:.6f} 的期权费")
    print(f"  ")
    print(f"  风险与收益：")
    print(f"    - 最大收益：你收到的期权费 = {call_price:.6f}")
    print(f"    - 最大损失：理论上无限（如果价格大涨）")
    print(f"    - 盈亏平衡点：K + 期权费 = {K} + {call_price:.6f} = {K + call_price:.6f}")
    print(f"    - 如果到期时价格 S_T <= {K}，你赚取全部期权费 {call_price:.6f}")
    print(f"    - 如果到期时价格 S_T > {K}，你需要按{K}的价格卖出资产")
    
    # 场景2：卖出看跌期权
    print(f"\n【场景2：卖出看跌期权 Short Put (K={K})】")
    print(f"  如果你卖出1份执行价格为{K}的看跌期权：")
    print(f"  你会收到期权费（Premium）= {put_price:.6f}")
    print(f"  注意：看跌期权价值较高，因为它是实值期权（ITM）")
    print(f"  ")
    print(f"  风险与收益：")
    print(f"    - 最大收益：你收到的期权费 = {put_price:.6f}")
    print(f"    - 最大损失：K - 期权费 = {K} - {put_price:.6f} = {K - put_price:.6f}")
    print(f"    - 盈亏平衡点：K - 期权费 = {K} - {put_price:.6f} = {K - put_price:.6f}")
    print(f"    - 如果到期时价格 S_T >= {K}，你赚取全部期权费 {put_price:.6f}")
    print(f"    - 如果到期时价格 S_T < {K}，你需要按{K}的价格买入资产")
    
    # 预期内在价值
    mu = S0
    sigma_price = sigma_abs
    d_call = (mu - K) / sigma_price
    expected_call_iv = (mu - K) * normal_cdf(d_call) + sigma_price * normal_pdf(d_call)
    expected_call_iv = max(expected_call_iv, 0)
    
    d_put = (K - mu) / sigma_price
    expected_put_iv = (K - mu) * normal_cdf(d_put) + sigma_price * normal_pdf(d_put)
    expected_put_iv = max(expected_put_iv, 0)
    
    print(f"\n【到期时的预期内在价值 Expected Intrinsic Value at Expiration】")
    print(f"  假设价格分布：S_T ~ N({mu}, {sigma_price}²)")
    print(f"  看涨期权预期内在价值 = {expected_call_iv:.6f}")
    print(f"  看跌期权预期内在价值 = {expected_put_iv:.6f}")
    
    print(f"\n【总结 Summary】")
    print(f"  如果你卖出执行价格为{K}的看涨期权：")
    print(f"  ✓ 当前价值确实不高，约 {call_price:.6f}")
    print(f"  ✓ 这是因为看涨期权是虚值（当前价格{S0} < 执行价格{K}）")
    print(f"  ✓ 你卖出期权会收到约 {call_price:.6f} 的期权费")
    print(f"  ✓ 但要注意：如果价格涨到{K}以上，你可能面临亏损")
    
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
    
    # 举例说明卖出执行价格105的期权
    example_sell_option()
    
    # 举例：计算在T/2时刻call_price的期望值
    example_expected_call_at_half_T()
    
    # 对于不同的T值，使用数值积分计算期望call price
    calculate_call_price_for_different_T()

if __name__ == "__main__":
    main()

