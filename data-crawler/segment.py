import pandas as pd
from sklearn.model_selection import train_test_split

df = pd.read_csv('../data/all.csv')

train_df, test_df = train_test_split(df, test_size=0.2, random_state=42)

val1_df, train_df = train_test_split(train_df, test_size=0.99, random_state=42)
val10_df, train_df = train_test_split(train_df, test_size=0.9, random_state=42)  

train_df.to_csv('../data/train.csv', index=False)
val1_df.to_csv('../data/train_mini.csv', index=False)
val10_df.to_csv('../data/train_small.csv', index=False)
test_df.to_csv('../data/test.csv', index=False)
