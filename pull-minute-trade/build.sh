name="tdx-pull"

#fullName="$name"_linux_amd64
#GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/$fullName
#echo "$fullName 编译完成..."
#echo "开始压缩..."
#upx -9 -k "./bin/$fullName"
#if [ -f "./bin/$fullName.~" ]; then
#  rm "./bin/$fullName.~"
#fi
#if [ -f "./bin/$fullName.000" ]; then
#  rm "./bin/$fullName.000"
#fi

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

#fullName="$name"_linux_arm
#GOOS=linux GOARCH=arm GOARM=7 go build -v -ldflags="-w -s" -o ./bin/$fullName
#echo "$fullName 编译完成..."
#echo "开始压缩..."
#upx -9 -k "./bin/$fullName"
#if [ -f "./bin/$fullName.~" ]; then
#  rm "./bin/$fullName.~"
#fi
#if [ -f "./bin/$fullName.000" ]; then
#  rm "./bin/$fullName.000"
#fi
#
#fullName="$name"
#GOOS=windows GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/$fullName.exe
#echo "$fullName 编译完成..."
#echo "开始压缩..."
#upx -9 -k "./bin/$fullName.exe"
#if [ -f "./bin/$fullName.ex~" ]; then
#  rm "./bin/$fullName.ex~"
#fi
#if [ -f "./bin/$fullName.000" ]; then
#  rm "./bin/$fullName.000"
#fi

sleep 8
