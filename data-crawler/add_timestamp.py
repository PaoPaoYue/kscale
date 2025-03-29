import csv

input_file = '../data/train_mini.csv'
output_file = '../experiment/benchmark/test1.csv'

with open(input_file, mode='r', newline='') as infile:
    reader = csv.reader(infile)
    rows = list(reader)


header = rows[0] + ['timestamp']
# 添加 timestamp 为 0 的数据到每一行
for row in rows[1:]:
    row.append(0)

with open(output_file, mode='w', newline='') as outfile:
    writer = csv.writer(outfile)
    writer.writerow(header)
    writer.writerows(rows[1:])

print(f"Updated CSV file saved as {output_file}")
