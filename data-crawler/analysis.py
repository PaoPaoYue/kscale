import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker

# 读取 CSV 文件，确保第一行作为列名
df = pd.read_csv('../data/all.csv')

# 将 'cfg' 列转换为浮点数类型，处理可能的转换错误
df['cfg'] = df['cfg'].astype(float)

# 设置绘图风格
plt.style.use('ggplot')

# 创建子图
fig, axes = plt.subplots(2, 3, figsize=(18, 10))

# 变量列表
variables = ['step', 'cfg', 'sampler', 'width', 'height', 'token_count']

# 绘制每个变量的直方图
for i, var in enumerate(variables):
    ax = axes[i // 3, i % 3]
    n, bins, patches = ax.hist(df[var], bins=10, color='skyblue', edgecolor='black')
    ax.set_title(f'{var.capitalize()} Distribution')
    ax.set_xlabel(var.capitalize())
    ax.set_ylabel('Frequency')
    
    # 在每个 bin 上方添加数量标签
    for patch, count in zip(patches, n):
        height = patch.get_height()
        ax.annotate(f'{int(count)}', xy=(patch.get_x() + patch.get_width() / 2, height),
                    xytext=(0, 5), textcoords='offset points',
                    ha='center', va='bottom', fontsize=9)
    

# 调整布局
plt.tight_layout()
plt.show()