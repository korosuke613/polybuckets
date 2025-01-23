#!/bin/bash

docker compose down

rm -rf ./minio-data

mkdir -p ./minio-data
