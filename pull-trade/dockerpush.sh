name="pull-trade"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./$name

docker pull --platform=linux/amd64 alpine:latest
docker build --platform=linux/amd64 --push -t docker.io/injoyai/pull-trade-amd64:latest -f ./Dockerfile .

sleep 8