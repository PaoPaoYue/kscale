import pandas as pd
from urllib.request import urlretrieve

# 下载 metadata.parquet 文件
table_url = 'https://huggingface.co/datasets/poloclub/diffusiondb/resolve/main/metadata.parquet'
urlretrieve(table_url, 'metadata.parquet')

# 读取 parquet 文件
metadata_df = pd.read_parquet('metadata.parquet')

# 选择需要的列
columns_to_keep = ['prompt', 'step', 'cfg', 'sampler', 'width', 'height']
filtered_df = metadata_df[columns_to_keep]

# 使用 iloc 选择第 20,000 至第 79,999 行数据
subset_df = filtered_df.iloc[20000:80000]

# 将数据保存为 CSV 文件
subset_df.to_csv('../data/diffusiondb_metadata_subset.csv', index=False)
