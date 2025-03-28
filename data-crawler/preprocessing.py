import pandas as pd
from transformers import AutoTokenizer

# 加载预训练的分词器，例如使用 BERT 的分词器
tokenizer = AutoTokenizer.from_pretrained('bert-base-uncased')

# 读取 CSV 文件
df = pd.read_csv('../data/diffusiondb_metadata_subset.csv')

# 定义一个函数，计算文本的 token 数量
def count_tokens(text):
    # 使用分词器将文本编码为 token ID，并获取 token 数量
    tokens = tokenizer.encode(text, add_special_tokens=False)
    return len(tokens)

df['prompt'] = df['prompt'].astype(str)
# 应用该函数到 'prompt' 列，创建一个新的 'token_count' 列
df['token_count'] = df['prompt'].apply(count_tokens)

# 过滤 'cfg' 列大于 20 且 'width' 和 'height' 列之积大于 2500 的行
df = df[(df['cfg'] >= 0)  & (df['cfg'] <= 20) & (df['width'] < 2500) & (df['height'] < 2500)]

# 计算 'step' 列的最小值和最大值
min_step = df['step'].min()
max_step = df['step'].max()

# 定义一个函数，将 'step' 值映射到新的范围
def scale_step(value):
    if value < 50:
        return int((value - min_step) / (50 - min_step) * 20)
    else:
        return int((value - 50) / (max_step - 50) * (50 - 20) + 20)

# 使用 apply 方法应用 scale_step 函数，并将结果赋值回 'step' 列
df['step'] = df['step'].apply(scale_step)

# 过滤掉 'step' 列值为 0 的行
df = df[df['step'] != 0]

df.insert(0, 'id', range(1, len(df) + 1))

# 保存结果到新的 CSV 文件
df.to_csv('../data/all.csv', index=False)
