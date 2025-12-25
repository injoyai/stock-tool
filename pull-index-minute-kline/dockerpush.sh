name="pull-index-minute-kline"
arch="amd64"

GOOS=linux GOARCH=$arch go build -v -ldflags="-w -s" -o ./bin/$name 

docker pull --platform=linux/$arch alpine:latest

# docker.io/injoyai/$name-$arch:latest
docker build --platform=linux/$arch --push -t crpi-ayrx20sj8nkmrgmh.cn-hangzhou.personal.cr.aliyuncs.com/injoyai/$name:latest -f ./Dockerfile .

sleep 2