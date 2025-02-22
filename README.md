# polybuckets - Simple browser app for S3 compatible services.

| Top page | Bucket root | Bucket child |
|:--------:|:-----------:|:------------:|
|![](./images/top-page.png) | ![](./images/bucket-root.png) | ![](./images/bucket-child.png) |

## Features
- List buckets
- List objects in a bucket
- Download an object

## Getting Started

```console
export AWS_REGION=
export AWS_ACCESS_KEY_ID=
export AWS_SECRET_ACCESS_KEY=

docker run -p 1323:1323 --env AWS_REGION=$AWS_REGION --env AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID --env AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY ghcr.io/korosuke613/polybuckets:latest
```

Also, you can use `AWS_PROFILE` instead of `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

```console
export AWS_REGION=
export AWS_PROFILE=

docker run -p 1323:1323 --env AWS_REGION=$AWS_REGION --env AWS_PROFILE=$AWS_PROFILE -v ~/.aws:/home/nonroot/.aws ghcr.io/korosuke613/polybuckets:latest
```

## Configuration

### Environment Variables

- `AWS_REGION`: (Required) Specify the AWS region.
- `AWS_PROFILE`: Specify the AWS profile. This can be used in place of `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
- `AWS_ENDPOINT`: Specify the endpoint for the S3 compatible service.
- `PB_PORT`: Specify the port that the server listens on (the default is `1323`).
- `PB_IP_ADDRESS`: Specify the IP address that the server listens on (the default is `0.0.0.0`).
- `PB_CACHE_DURATION`: Specify the aws s3 list objects cache expiration time (default is `60m`).
- `PB_SITE_NAME`: Specify the site name (default is `polybuckets`).

## Development

### 1. Launch development S3 bucket (Terminal A)

```console
cd tools
./init-minio.sh
./start-minio.sh
```

### 2. Launch polybuckets (Terminal B)

```console
set -a; eval "$(cat ./.env <(echo) <(declare -x))"; set +a;  # Load .env

go run ./main.go
```

### 3. Output

Open localhost:1323 in your browser.

Access log is shown in Terminal A.

```console
❯ go run main.go
{"time":"2025-01-25T21:54:25.594383Z","level":"INFO","msg":"starting server","ip":"0.0.0.0","port":"1323"}
{"time":"2025-01-25T21:54:39.748322Z","level":"INFO","msg":"access log","value":{"remote_ip":"::1","host":"localhost:1323","method":"GET","uri":"/","user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0","status":200,"error":"","latency":40673459,"latency_human":"40.673459ms","bytes_in":0,"bytes_out":818}}
{"time":"2025-01-25T21:54:41.105367Z","level":"INFO","msg":"access log","value":{"remote_ip":"::1","host":"localhost:1323","method":"GET","uri":"/bucket2/","user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0","status":200,"error":"","latency":9485459,"latency_human":"9.485459ms","bytes_in":0,"bytes_out":1829,"hit_cache":false}}
{"time":"2025-01-25T21:54:42.37454Z","level":"INFO","msg":"access log","value":{"remote_ip":"::1","host":"localhost:1323","method":"GET","uri":"/bucket2/hoge/","user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0","status":200,"error":"","latency":13514042,"latency_human":"13.514042ms","bytes_in":0,"bytes_out":1694,"hit_cache":false}}
{"time":"2025-01-25T21:54:44.092354Z","level":"INFO","msg":"access log","value":{"remote_ip":"::1","host":"localhost:1323","method":"GET","uri":"/download/bucket2/hoge/test_2.txt","user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0","status":200,"error":"","latency":7519166,"latency_human":"7.519166ms","bytes_in":0,"bytes_out":34}}
```

minio api call log is shown in Terminal B.

```console
❯ ./start-minio.sh
[+] Running 2/2
 ✔ Container tools-minio-1  Running                                                                                                                                                                     0.0s
 ✔ Container tools-mc-1     Running                                                                                                                                                                     0.0s
Attaching to mc-1, minio-1
mc-1     | 2025-01-25T21:54:39.740 [200 OK] s3.ListBuckets localhost:9000/?x-id=ListBuckets  192.168.107.1    6.236ms      ⇣  5.302117ms  ↑ 131 B ↓ 545 B
mc-1     | 2025-01-25T21:54:41.098 [200 OK] s3.ListObjectsV2 localhost:9000/bucket2?delimiter=%2F&list-type=2&prefix=  192.168.107.1    5.658ms      ⇣  5.579237ms  ↑ 131 B ↓ 721 B
mc-1     | 2025-01-25T21:54:42.369 [200 OK] s3.ListObjectsV2 localhost:9000/bucket2?delimiter=%2F&list-type=2&prefix=hoge%2F  192.168.107.1    3.875ms      ⇣  3.8491ms   ↑ 131 B ↓ 534 B
mc-1     | 2025-01-25T21:54:44.087 [200 OK] s3.GetObject localhost:9000/bucket2/hoge/test_2.txt?x-id=GetObject  192.168.107.1    3.2ms        ⇣  2.90345ms  ↑ 131 B ↓ 34 B
```
