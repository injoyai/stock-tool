name="tdx"

fullName="$name"
GOOS=linux GOARCH=arm64 go build -v -ldflags="-w -s" -o ./bin/$fullName
echo "$fullName 编译完成..."
echo "开始压缩..."
upx -9 -k "./bin/$fullName"
if [ -f "./bin/$fullName.~" ]; then
  rm "./bin/$fullName.~"
fi
if [ -f "./bin/$fullName.000" ]; then
  rm "./bin/$fullName.000"
fi

sleep 8
