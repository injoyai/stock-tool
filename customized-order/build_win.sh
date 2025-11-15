#!/bin/bash
set -euo pipefail

# 输出文件名（默认 test）
name=${1:-pull-order}

# 检查依赖
for cmd in go cmd.exe; do
    if ! command -v $cmd &>/dev/null; then
        echo "❌ 缺少依赖: $cmd"
        exit 1
    fi
done

# 编译
echo "🔨 编译 $name.exe ..."
GOOS=windows GOARCH=amd64 go build -v -ldflags="-H windowsgui -w -s" -o "./bin/$name.exe"

if [ ! -f "./bin/$name.exe" ]; then
    echo "❌ 编译失败，未生成 $name.exe"
    exit 1
fi
echo "✅ $name.exe 编译完成"

# 压缩（如果有 upx）
if command -v upx &>/dev/null; then
    echo "📦 使用 UPX 压缩..."
    upx -9 -k "./bin/$name.exe" || echo "⚠️ UPX 压缩失败，继续执行"
    rm -f "./bin/$name.ex~" "./$name.000"
else
    echo "⚠️ 未检测到 upx，跳过压缩"
fi


echo "🎉 全部完成"
