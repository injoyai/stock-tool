name="pull-trade"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./$name
echo "$name 编译完成..."
echo "开始压缩..."
#upx -9 -k "./bin/$fullName"
rm "./bin/v.~"
rm "./bin/$name.000"
echo "等待结束..."
sleep 2
