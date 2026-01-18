name="pull-day-kline"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/$name

docker pull --platform=linux/amd64 alpine:latest
docker build --platform=linux/amd64 --push -t crpi-ayrx20sj8nkmrgmh.cn-hangzhou.personal.cr.aliyuncs.com/injoyai/$name:latest -f ./Dockerfile .

sleep 8