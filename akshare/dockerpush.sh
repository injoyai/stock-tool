version="latest"

docker pull --platform=linux/arm64 python:3.11-slim
docker build --platform=linux/arm64 --push -t docker.io/injoyai/akshare-arm64:$version -f ./Dockerfile .

sleep 8