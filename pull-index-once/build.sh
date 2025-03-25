name="拉取指数"

GOOS=windows GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/$name.exe
echo "$name 编译完成..."
echo "开始压缩..."
upx -9 -k "./bin/$name.exe"
if [ -f "./bin/$name.ex~" ]; then
  rm "./bin/$name.ex~"
fi
if [ -f "./bin/$name.000" ]; then
  rm "./bin/$name.000"
fi

sleep 3
