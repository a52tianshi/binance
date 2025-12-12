import csv
import numpy as np
try:
    import matplotlib.pyplot as plt
    from matplotlib.colors import TwoSlopeNorm
    HAS_MATPLOTLIB = True
except ImportError:
    HAS_MATPLOTLIB = False
    print("警告: matplotlib未安装，将只输出统计信息")

print("正在读取z-score矩阵...")

# 读取CSV文件
matrix_data = []
with open('zscore_matrix.csv', 'r') as f:
    reader = csv.reader(f)
    header = next(reader)  # 跳过标题行
    for row in reader:
        # 跳过第一列（时间索引），只读取z-score值
        matrix_data.append([float(x) for x in row[1:]])

matrix = np.array(matrix_data)

print(f"矩阵大小: {matrix.shape}")
print(f"数据范围: z-score从 {matrix.min():.2f} 到 {matrix.max():.2f}")

if HAS_MATPLOTLIB:
    # 创建图形
    fig, ax = plt.subplots(figsize=(16, 10))

    # 使用对称的颜色映射，以0为中心
    vmax = max(abs(matrix.min()), abs(matrix.max()))
    vmin = -vmax

    # 创建颜色映射（红色=负值，白色=0，蓝色=正值）
    cmap = plt.cm.RdBu_r
    norm = TwoSlopeNorm(vmin=vmin, vcenter=0, vmax=vmax)

    # 绘制热力图
    im = ax.imshow(matrix, aspect='auto', cmap=cmap, norm=norm, interpolation='nearest')

    # 设置坐标轴标签
    ax.set_xlabel('Time Window (minutes)', fontsize=12)
    ax.set_ylabel('Time Index (last 7 days, per minute)', fontsize=12)
    ax.set_title('Z-Score Heatmap (Last 7 Days × 1-10080 Minutes Windows)', fontsize=14, fontweight='bold')

    # 添加颜色条
    cbar = plt.colorbar(im, ax=ax, label='Z-Score', shrink=0.8)
    cbar.ax.set_ylabel('Z-Score', rotation=270, labelpad=20)

    # 设置x轴刻度（显示关键时间窗口）
    x_ticks = [0, 60, 240, 1440, 2880, 4320, 5760, 7200, 8640, 10080]
    x_labels = ['1m', '1h', '4h', '1d', '2d', '3d', '4d', '5d', '6d', '7d']
    ax.set_xticks(x_ticks)
    ax.set_xticklabels(x_labels)

    # 设置y轴刻度（显示天数）
    y_ticks = [0, 1440, 2880, 4320, 5760, 7200, 8640, 10080]
    y_labels = ['7d ago', '6d ago', '5d ago', '4d ago', '3d ago', '2d ago', '1d ago', 'Now']
    ax.set_yticks(y_ticks)
    ax.set_yticklabels(y_labels)

    # 添加网格线
    ax.grid(True, alpha=0.3, linestyle='--', linewidth=0.5)

    plt.tight_layout()

    # 保存图片
    output_file = 'zscore_matrix_heatmap.png'
    plt.savefig(output_file, dpi=300, bbox_inches='tight')
    print(f"\n图片已保存到 {output_file}")
    
    plt.show()
else:
    print("\n注意: matplotlib未安装，无法生成图片")
    print("请运行: pip3 install matplotlib numpy")

# 显示统计信息
print("\n统计信息:")
print(f"  最小z-score: {matrix.min():.4f}")
print(f"  最大z-score: {matrix.max():.4f}")
print(f"  均值: {matrix.mean():.4f}")
print(f"  标准差: {matrix.std():.4f}")

# 统计不同范围的z-score数量
total = matrix.size
non_zero = np.count_nonzero(matrix)
abs_gt_1 = np.sum(np.abs(matrix) > 1)
abs_gt_2 = np.sum(np.abs(matrix) > 2)
abs_gt_3 = np.sum(np.abs(matrix) > 3)

print(f"\nZ-score分布:")
print(f"  |z| > 1: {abs_gt_1} ({abs_gt_1/total*100:.2f}%)")
print(f"  |z| > 2: {abs_gt_2} ({abs_gt_2/total*100:.2f}%)")
print(f"  |z| > 3: {abs_gt_3} ({abs_gt_3/total*100:.2f}%)")
print(f"  非零值: {non_zero} ({non_zero/total*100:.2f}%)")

plt.show()

