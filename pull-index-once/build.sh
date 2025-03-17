name="拉取指数"
invalid=""

fullName="$name($invalid)"
GOOS=windows GOARCH=amd64 go build -v -ldflags="-w -s -X main.Invalid=$invalid" -o ./bin/$fullName.exe
echo "$fullName 编译完成..."
echo "开始压缩..."
#upx -9 -k "./bin/$fullName.exe"
#if [ -f "./bin/$fullName.ex~" ]; then
#  rm "./bin/$fullName.ex~"
#fi
#if [ -f "./bin/$fullName.000" ]; then
#  rm "./bin/$fullName.000"
#fi

sleep 3
