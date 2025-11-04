name="pull-codes"
arch="amd64"

GOOS=linux GOARCH=$arch go build -v -ldflags="-w -s" -o ./bin/$name 

docker pull --platform=linux/$arch alpine:latest
docker build --platform=linux/$arch --push -t 192.168.192.5:5000/$name:latest -f ./Dockerfile .

sleep 8