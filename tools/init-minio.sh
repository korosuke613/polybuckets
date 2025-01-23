#!/bin/bash

# .env ファイルを読み込んで環境変数を設定する
# ref: https://qiita.com/mpyw/items/6d43d584c7e24d40af7b
set -a; eval "$(cat ../.env <(echo) <(declare -x))"; set +a;

docker compose up -d

# Create multiple buckets and upload multiple dummy data files to each
for bucket in bucket1 bucket2 bucket3; do
  echo "Creating $bucket..."
  aws s3 mb s3://$bucket --endpoint $AWS_ENDPOINT

  for prefix in test_0.txt test_1.txt hoge/test_2.txt hoge/fuga/test_3.txt; do
    echo "Uploading test file $prefix to $bucket..."
    echo "test file content $prefix" > test_minio.txt
    aws s3 cp test_minio.txt s3://$bucket/$prefix --endpoint $AWS_ENDPOINT
    rm test_minio.txt
  done
done

echo "Local S3 setup completed."
