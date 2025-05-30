name="pull-tdx"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/$name

docker pull --platform=linux/amd64 alpine:latest
docker build --platform=linux/amd64 --push -t docker.io/injoyai/pull-tdx-amd64:latest -f ./Dockerfile .

sleep 8