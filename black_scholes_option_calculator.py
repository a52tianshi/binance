import math
import csv
from typing import List, Tuple
import matplotlib.pyplot as plt
import numpy as np

def normal_cdf(x: float) -> float:
    """
    Calculate the cumulative distribution function (CDF) of the standard normal distribution
    Using approximation formula
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
    Calculate the probability density function (PDF) of the standard normal distribution
    """
    return (1.0 / math.sqrt(2.0 * math.pi)) * math.exp(-0.5 * x * x)

def expected_call_intrinsic_normal(
    current_price: float,
    strike: float,
    time_to_expiration: float,
    volatility: float
) -> float:
    """
    Calculate expected intrinsic value of a call option at expiration
    assuming price follows a normal distribution
    
    Assumes: S_T ~ N(current_price, (current_price * volatility * sqrt(T))^2)
    
    Args:
        current_price: Current asset price (S_0)
        strike: Strike price (K)
        time_to_expiration: Time to expiration in years (T)
        volatility: Volatility (annual, σ)
    
    Returns:
        Expected intrinsic value: E[max(S_T - K, 0)]
    """
    if time_to_expiration <= 0:
        return max(current_price - strike, 0)
    
    # Mean and standard deviation of price at expiration
    mu = current_price  # Mean stays at current price (assuming no drift since r=0)
    sigma = current_price * volatility * math.sqrt(time_to_expiration)
    
    if sigma == 0:
        return max(mu - strike, 0)
    
    # Standardized value
    d = (mu - strike) / sigma
    
    # Expected intrinsic value formula for normal distribution
    # E[max(S_T - K, 0)] = (μ - K) * Φ(d) + σ * φ(d)
    expected_value = (mu - strike) * normal_cdf(d) + sigma * normal_pdf(d)
    
    return max(expected_value, 0)

def expected_put_intrinsic_normal(
    current_price: float,
    strike: float,
    time_to_expiration: float,
    volatility: float
) -> float:
    """
    Calculate expected intrinsic value of a put option at expiration
    assuming price follows a normal distribution
    
    Assumes: S_T ~ N(current_price, (current_price * volatility * sqrt(T))^2)
    
    Args:
        current_price: Current asset price (S_0)
        strike: Strike price (K)
        time_to_expiration: Time to expiration in years (T)
        volatility: Volatility (annual, σ)
    
    Returns:
        Expected intrinsic value: E[max(K - S_T, 0)]
    """
    if time_to_expiration <= 0:
        return max(strike - current_price, 0)
    
    # Mean and standard deviation of price at expiration
    mu = current_price  # Mean stays at current price (assuming no drift since r=0)
    sigma = current_price * volatility * math.sqrt(time_to_expiration)
    
    if sigma == 0:
        return max(strike - mu, 0)
    
    # Standardized value
    d = (strike - mu) / sigma
    
    # Expected intrinsic value formula for normal distribution
    # E[max(K - S_T, 0)] = (K - μ) * Φ(d) + σ * φ(d)
    expected_value = (strike - mu) * normal_cdf(d) + sigma * normal_pdf(d)
    
    return max(expected_value, 0)

def black_scholes_call(S: float, K: float, T: float, r: float, sigma: float) -> float:
    """
    Calculate Black-Scholes call option price
    
    Args:
        S: Current stock/asset price
        K: Strike price
        T: Time to expiration (in years)
        r: Risk-free interest rate (annual)
        sigma: Volatility (annual)
    
    Returns:
        Call option price
    """
    if T <= 0:
        return max(S - K, 0)
    
    d1 = (math.log(S / K) + (r + 0.5 * sigma ** 2) * T) / (sigma * math.sqrt(T))
    d2 = d1 - sigma * math.sqrt(T)
    
    call_price = S * normal_cdf(d1) - K * math.exp(-r * T) * normal_cdf(d2)
    
    return call_price

def black_scholes_put(S: float, K: float, T: float, r: float, sigma: float) -> float:
    """
    Calculate Black-Scholes put option price
    
    Args:
        S: Current stock/asset price
        K: Strike price
        T: Time to expiration (in years)
        r: Risk-free interest rate (annual)
        sigma: Volatility (annual)
    
    Returns:
        Put option price
    """
    # Using put-call parity: Put = Call - S + K * e^(-r*T)
    call_price = black_scholes_call(S, K, T, r, sigma)
    put_price = call_price - S + K * math.exp(-r * T)
    
    return put_price

def calculate_option_values_for_strikes(
    current_price: float,
    strike_prices: List[float],
    time_to_expiration: float,
    risk_free_rate: float,
    volatility: float
) -> List[Tuple[float, float, float]]:
    """
    Calculate call and put option values for multiple strike prices
    
    Args:
        current_price: Current asset price
        strike_prices: List of strike prices
        time_to_expiration: Time to expiration in years
        risk_free_rate: Risk-free interest rate (annual)
        volatility: Volatility (annual)
    
    Returns:
        List of tuples: (strike_price, call_price, put_price)
    """
    results = []
    
    for K in strike_prices:
        call_price = black_scholes_call(current_price, K, time_to_expiration, risk_free_rate, volatility)
        put_price = black_scholes_put(current_price, K, time_to_expiration, risk_free_rate, volatility)
        results.append((K, call_price, put_price))
    
    return results

def generate_strike_prices(current_price: float, min_percent: float = 0.5, max_percent: float = 1.5, step_percent: float = 0.05) -> List[float]:
    """
    Generate a range of strike prices around the current price
    
    Args:
        current_price: Current asset price
        min_percent: Minimum strike as percentage of current price (e.g., 0.5 = 50%)
        max_percent: Maximum strike as percentage of current price (e.g., 1.5 = 150%)
        step_percent: Step size as percentage of current price (e.g., 0.05 = 5%)
    
    Returns:
        List of strike prices
    """
    strikes = []
    price = current_price * min_percent
    
    while price <= current_price * max_percent:
        strikes.append(round(price, 2))
        price += current_price * step_percent
    
    return strikes

def save_to_csv(results: List[Tuple[float, float, float]], current_price: float, filename: str = "option_values.csv"):
    """
    Save option values to CSV file
    
    Args:
        results: List of tuples (strike_price, call_price, put_price)
        current_price: Current asset price
        filename: Output filename
    """
    with open(filename, mode="w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        
        writer.writerow([
            "Current Price",
            "Strike Price",
            "Strike % of Current",
            "Call Option Price",
            "Put Option Price",
            "Call Intrinsic Value",
            "Put Intrinsic Value",
            "Call Time Value",
            "Put Time Value"
        ])
        
        for strike, call_price, put_price in results:
            strike_pct = (strike / current_price) * 100
            call_intrinsic = max(current_price - strike, 0)
            put_intrinsic = max(strike - current_price, 0)
            call_time_value = call_price - call_intrinsic
            put_time_value = put_price - put_intrinsic
            
            writer.writerow([
                current_price,
                strike,
                f"{strike_pct:.2f}%",
                f"{call_price:.6f}",
                f"{put_price:.6f}",
                f"{call_intrinsic:.6f}",
                f"{put_intrinsic:.6f}",
                f"{call_time_value:.6f}",
                f"{put_time_value:.6f}"
            ])
    
    print(f"\nOption values saved to {filename}")

def plot_option_values(
    results: List[Tuple[float, float, float]],
    current_price: float,
    risk_free_rate: float,
    volatility: float,
    time_to_expiration: float,
    filename: str = "option_values_plot.png"
):
    """
    Create 2D plot of Time Value vs strike prices
    
    Args:
        results: List of tuples (strike_price, call_price, put_price)
        current_price: Current asset price
        risk_free_rate: Risk-free interest rate
        volatility: Volatility
        time_to_expiration: Time to expiration in years
        filename: Output filename for the plot
    """
    strikes = [r[0] for r in results]
    call_prices = [r[1] for r in results]
    put_prices = [r[2] for r in results]
    
    # Calculate intrinsic values
    call_intrinsic = [max(current_price - K, 0) for K in strikes]
    put_intrinsic = [max(K - current_price, 0) for K in strikes]
    
    # Calculate time values
    call_time_value = [call_prices[i] - call_intrinsic[i] for i in range(len(strikes))]
    put_time_value = [put_prices[i] - put_intrinsic[i] for i in range(len(strikes))]
    
    # Create a single figure for Time Value only
    fig, ax = plt.subplots(figsize=(12, 8))
    
    # Plot with smooth lines (no markers for smoother appearance)
    ax.plot(strikes, call_time_value, 'b-', linewidth=2.5, label='Call Time Value', alpha=0.8)
    ax.plot(strikes, put_time_value, 'r-', linewidth=2.5, label='Put Time Value', alpha=0.8)
    
    # Add vertical line at current price
    ax.axvline(x=current_price, color='g', linestyle='--', linewidth=2, label=f'Current Price ({current_price})', alpha=0.7)
    
    # Add horizontal line at y=0
    ax.axhline(y=0, color='k', linestyle='-', linewidth=0.5, alpha=0.3)
    
    # Labels and title
    ax.set_xlabel('Strike Price', fontsize=13, fontweight='bold')
    ax.set_ylabel('Time Value', fontsize=13, fontweight='bold')
    ax.set_title(f'Time Value of Options\nS={current_price}, r={risk_free_rate*100}%, σ={volatility*100}%, T={time_to_expiration:.3f} years', 
                 fontsize=14, fontweight='bold')
    
    # Legend and grid
    ax.legend(loc='best', fontsize=12, framealpha=0.9)
    ax.grid(True, alpha=0.3, linestyle='--')
    
    # Improve appearance
    ax.spines['top'].set_visible(False)
    ax.spines['right'].set_visible(False)
    
    plt.tight_layout()
    plt.savefig(filename, dpi=300, bbox_inches='tight')
    print(f"Plot saved to {filename}")
    plt.close()

def calculate_expected_intrinsic_values(
    current_price: float,
    strike_prices: List[float],
    time_to_expiration: float,
    volatility: float
) -> List[Tuple[float, float, float]]:
    """
    Calculate expected intrinsic values for call and put options at expiration
    assuming price follows a normal distribution
    
    Args:
        current_price: Current asset price
        strike_prices: List of strike prices
        time_to_expiration: Time to expiration in years
        volatility: Volatility (annual)
    
    Returns:
        List of tuples: (strike_price, expected_call_intrinsic, expected_put_intrinsic)
    """
    results = []
    
    for K in strike_prices:
        call_exp_intrinsic = expected_call_intrinsic_normal(current_price, K, time_to_expiration, volatility)
        put_exp_intrinsic = expected_put_intrinsic_normal(current_price, K, time_to_expiration, volatility)
        results.append((K, call_exp_intrinsic, put_exp_intrinsic))
    
    return results

def plot_expected_intrinsic_values(
    results: List[Tuple[float, float, float]],
    current_price: float,
    volatility: float,
    time_to_expiration: float,
    filename: str = "expected_intrinsic_values.png"
):
    """
    Create 2D plot of Expected Intrinsic Values vs strike prices
    
    Args:
        results: List of tuples (strike_price, expected_call_intrinsic, expected_put_intrinsic)
        current_price: Current asset price
        volatility: Volatility
        time_to_expiration: Time to expiration in years
        filename: Output filename for the plot
    """
    strikes = [r[0] for r in results]
    call_exp_intrinsic = [r[1] for r in results]
    put_exp_intrinsic = [r[2] for r in results]
    
    # Create a single figure
    fig, ax = plt.subplots(figsize=(12, 8))
    
    # Plot with smooth lines
    ax.plot(strikes, call_exp_intrinsic, 'b-', linewidth=2.5, label='Expected Call Intrinsic Value', alpha=0.8)
    ax.plot(strikes, put_exp_intrinsic, 'r-', linewidth=2.5, label='Expected Put Intrinsic Value', alpha=0.8)
    
    # Add vertical line at current price
    ax.axvline(x=current_price, color='g', linestyle='--', linewidth=2, label=f'Current Price ({current_price})', alpha=0.7)
    
    # Add horizontal line at y=0
    ax.axhline(y=0, color='k', linestyle='-', linewidth=0.5, alpha=0.3)
    
    # Labels and title
    ax.set_xlabel('Strike Price', fontsize=13, fontweight='bold')
    ax.set_ylabel('Expected Intrinsic Value at Expiration', fontsize=13, fontweight='bold')
    ax.set_title(f'Expected Intrinsic Value at Expiration (Normal Distribution)\nS={current_price}, σ={volatility*100}%, T={time_to_expiration:.3f} years', 
                 fontsize=14, fontweight='bold')
    
    # Legend and grid
    ax.legend(loc='best', fontsize=12, framealpha=0.9)
    ax.grid(True, alpha=0.3, linestyle='--')
    
    # Improve appearance
    ax.spines['top'].set_visible(False)
    ax.spines['right'].set_visible(False)
    
    plt.tight_layout()
    plt.savefig(filename, dpi=300, bbox_inches='tight')
    print(f"Expected intrinsic values plot saved to {filename}")
    plt.close()

def save_expected_intrinsic_to_csv(
    results: List[Tuple[float, float, float]],
    current_price: float,
    time_to_expiration: float,
    filename: str = "expected_intrinsic_values.csv"
):
    """
    Save expected intrinsic values to CSV file
    """
    with open(filename, mode="w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        
        writer.writerow([
            "Current Price",
            "Strike Price",
            "Strike % of Current",
            "Expected Call Intrinsic Value",
            "Expected Put Intrinsic Value"
        ])
        
        for strike, call_exp, put_exp in results:
            strike_pct = (strike / current_price) * 100
            
            writer.writerow([
                current_price,
                strike,
                f"{strike_pct:.2f}%",
                f"{call_exp:.6f}",
                f"{put_exp:.6f}"
            ])
    
    print(f"Expected intrinsic values saved to {filename}")

def calculate_intrinsic_at_T1(current_price: float = 100.0, volatility: float = 0.03, strike_price: float = None):
    """
    Calculate expected intrinsic value at T=1 with given parameters
    
    Formula:
    For Call: E[max(S_T - K, 0)] = (μ - K) * Φ((μ-K)/σ) + σ * φ((μ-K)/σ)
    For Put:  E[max(K - S_T, 0)] = (K - μ) * Φ((K-μ)/σ) + σ * φ((K-μ)/σ)
    
    Where:
    - μ = current_price (mean of normal distribution)
    - σ = current_price * volatility * sqrt(T) = 100 * 0.03 * sqrt(1) = 3
    - K = strike price
    - Φ = CDF of standard normal distribution
    - φ = PDF of standard normal distribution
    
    Args:
        current_price: Current asset price (default: 100)
        volatility: Volatility (default: 0.03 = 3%)
        strike_price: Strike price (if None, calculates for multiple strikes)
    """
    T = 1.0
    mu = current_price
    sigma = current_price * volatility * math.sqrt(T)
    
    print("="*90)
    print("EXPECTED INTRINSIC VALUE FUNCTION AT T = 1")
    print("="*90)
    print(f"\nParameters:")
    print(f"  Current Price (S₀) = {current_price}")
    print(f"  Volatility (σ/α) = {volatility} ({volatility*100}%)")
    print(f"  Time to Expiration (T) = {T} year")
    print(f"\nNormal Distribution Parameters:")
    print(f"  Mean (μ) = S₀ = {mu}")
    print(f"  Standard Deviation (σ) = S₀ × σ × √T = {current_price} × {volatility} × √{T} = {sigma:.6f}")
    print(f"\nAssumption: S_T ~ N(μ, σ²) = N({mu}, {sigma**2:.6f})")
    
    print(f"\n" + "="*90)
    print("MATHEMATICAL FORMULA:")
    print("="*90)
    print("\nFor Call Option Expected Intrinsic Value:")
    print("  E[max(S_T - K, 0)] = (μ - K) × Φ((μ-K)/σ) + σ × φ((μ-K)/σ)")
    print(f"  E[max(S_T - K, 0)] = ({mu} - K) × Φ(({mu}-K)/{sigma:.6f}) + {sigma:.6f} × φ(({mu}-K)/{sigma:.6f})")
    
    print("\nFor Put Option Expected Intrinsic Value:")
    print("  E[max(K - S_T, 0)] = (K - μ) × Φ((K-μ)/σ) + σ × φ((K-μ)/σ)")
    print(f"  E[max(K - S_T, 0)] = (K - {mu}) × Φ((K-{mu})/{sigma:.6f}) + {sigma:.6f} × φ((K-{mu})/{sigma:.6f})")
    
    print("\nWhere:")
    print("  Φ = Cumulative Distribution Function (CDF) of standard normal")
    print("  φ = Probability Density Function (PDF) of standard normal")
    
    if strike_price is not None:
        # Calculate for specific strike
        call_exp = expected_call_intrinsic_normal(current_price, strike_price, T, volatility)
        put_exp = expected_put_intrinsic_normal(current_price, strike_price, T, volatility)
        
        print(f"\n" + "="*90)
        print(f"RESULTS FOR STRIKE PRICE K = {strike_price}:")
        print("="*90)
        print(f"  Expected Call Intrinsic Value = {call_exp:.6f}")
        print(f"  Expected Put Intrinsic Value  = {put_exp:.6f}")
    else:
        # Calculate for multiple strikes
        strikes = [95 + i*1.0 for i in range(11)]  # 95 to 105
        
        print(f"\n" + "="*90)
        print("RESULTS FOR STRIKE PRICES 95 to 105:")
        print("="*90)
        print(f"{'Strike (K)':>12} | {'Call E[IV]':>18} | {'Put E[IV]':>18}")
        print("-" * 90)
        
        for K in strikes:
            call_exp = expected_call_intrinsic_normal(current_price, K, T, volatility)
            put_exp = expected_put_intrinsic_normal(current_price, K, T, volatility)
            print(f"{K:>12.2f} | {call_exp:>18.6f} | {put_exp:>18.6f}")
    
    print("="*90)
    return sigma

def main():
    # Configuration - as specified by user
    current_price = 100.0  # Current asset price
    risk_free_rate = 0.0   # Interest rate = 0% (annual)
    volatility = 0.03      # Volatility = 3% (annual)
    time_to_expiration = 1.0  # 1 year
    
    # Calculate and print intrinsic value function at T=1
    print("\n")
    calculate_intrinsic_at_T1(current_price, volatility)
    print("\n")
    
    # You can modify these parameters above, or use command line arguments
    # For example, to change time to expiration to 7 days:
    # time_to_expiration = 7 / 365  # 7 days in years
    
    # You can customize these parameters
    print("=== Black-Scholes Option Value Calculator ===\n")
    print(f"Current Price (S): {current_price}")
    print(f"Risk-Free Rate (r): {risk_free_rate * 100}%")
    print(f"Volatility (σ): {volatility * 100}%")
    print(f"Time to Expiration (T): {time_to_expiration} years ({time_to_expiration * 365:.0f} days)")
    print("\n" + "="*60 + "\n")
    
    # Generate strike prices from 95 to 105 with small steps for smooth curves
    strike_prices = []
    strike = 95.0
    while strike <= 105.0:
        strike_prices.append(round(strike, 2))
        strike += 0.5  # Small step for smooth curves
    
    print(f"Calculating option values for {len(strike_prices)} strike prices...")
    
    # Calculate option values
    results = calculate_option_values_for_strikes(
        current_price=current_price,
        strike_prices=strike_prices,
        time_to_expiration=time_to_expiration,
        risk_free_rate=risk_free_rate,
        volatility=volatility
    )
    
    # Save to CSV
    save_to_csv(results, current_price, "option_values.csv")
    
    # Create plots
    print("\nGenerating plots...")
    plot_option_values(results, current_price, risk_free_rate, volatility, time_to_expiration, "option_values_plot.png")
    
    # Calculate Expected Intrinsic Values at T = 0.5 (assuming normal distribution)
    print("\n" + "="*80)
    print("CALCULATING EXPECTED INTRINSIC VALUES AT T = 0.5 (Normal Distribution)")
    print("="*80)
    time_to_expiration_intrinsic = 0.5  # 6 months
    
    expected_intrinsic_results = calculate_expected_intrinsic_values(
        current_price=current_price,
        strike_prices=strike_prices,
        time_to_expiration=time_to_expiration_intrinsic,
        volatility=volatility
    )
    
    # Save expected intrinsic values to CSV
    save_expected_intrinsic_to_csv(
        expected_intrinsic_results,
        current_price,
        time_to_expiration_intrinsic,
        "expected_intrinsic_values_T0.5.csv"
    )
    
    # Create plot for expected intrinsic values
    plot_expected_intrinsic_values(
        expected_intrinsic_results,
        current_price,
        volatility,
        time_to_expiration_intrinsic,
        "expected_intrinsic_values_T0.5.png"
    )
    
    # Print summary table for expected intrinsic values (ALL strikes)
    print("\n" + "="*90)
    print("EXPECTED INTRINSIC VALUES AT T = 0.5 (Normal Distribution)")
    print("Assumes: S_T ~ N(current_price, (current_price × σ × √T)²)")
    print("="*90)
    print(f"{'Strike':>10} | {'Strike %':>10} | {'Exp Call IV':>18} | {'Exp Put IV':>18} | {'Total Exp IV':>18}")
    print("-" * 90)
    
    # Print ALL strikes (not every other)
    for strike, call_exp, put_exp in expected_intrinsic_results:
        strike_pct = (strike / current_price) * 100
        total_exp = call_exp + put_exp
        print(f"{strike:>10.2f} | {strike_pct:>9.2f}% | {call_exp:>18.6f} | {put_exp:>18.6f} | {total_exp:>18.6f}")
    
    print("="*90)
    std_dev = current_price * volatility * math.sqrt(time_to_expiration_intrinsic)
    print(f"\nNote: Expected intrinsic values represent the average payoff at expiration")
    print(f"      assuming price follows a normal distribution with:")
    print(f"      - Mean (μ) = {current_price:.2f}")
    print(f"      - Standard Deviation (σ) = {std_dev:.6f}")
    
    # Print summary table
    print("\n" + "="*80)
    print("OPTION VALUES SUMMARY")
    print("="*80)
    print(f"{'Strike':>10} | {'Strike %':>10} | {'Call Price':>12} | {'Put Price':>12} | {'Call IV':>12} | {'Put IV':>12}")
    print("-" * 80)
    
    for strike, call_price, put_price in results[::2]:  # Print every other strike to save space
        strike_pct = (strike / current_price) * 100
        call_intrinsic = max(current_price - strike, 0)
        put_intrinsic = max(strike - current_price, 0)
        print(f"{strike:>10.2f} | {strike_pct:>9.2f}% | {call_price:>12.6f} | {put_price:>12.6f} | {call_intrinsic:>12.6f} | {put_intrinsic:>12.6f}")
    
    print("="*80)
    print(f"\nFull results saved to option_values.csv")
    
    # Also calculate for specific commonly used strikes
    print("\n" + "="*60)
    print("COMMON STRIKE PRICES:")
    print("="*60)
    common_strikes = [80, 85, 90, 95, 100, 105, 110, 115, 120]
    common_strikes = [s for s in common_strikes if s >= current_price * 0.5 and s <= current_price * 1.5]
    
    for strike in common_strikes:
        call_price = black_scholes_call(current_price, strike, time_to_expiration, risk_free_rate, volatility)
        put_price = black_scholes_put(current_price, strike, time_to_expiration, risk_free_rate, volatility)
        call_intrinsic = max(current_price - strike, 0)
        put_intrinsic = max(strike - current_price, 0)
        
        print(f"Strike {strike:>6.2f}: Call = {call_price:>8.6f} (IV: {call_intrinsic:>8.6f}), "
              f"Put = {put_price:>8.6f} (IV: {put_intrinsic:>8.6f})")

if __name__ == "__main__":
    main()

