#!/bin/bash

# 生成 protobuf 文件的脚本

set -e

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed. Please install Protocol Buffers compiler."
    echo "On macOS: brew install protobuf"
    echo "On Ubuntu: sudo apt-get install protobuf-compiler"
    exit 1
fi

# 检查 protoc-gen-go 是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# 检查 protoc-gen-go-grpc 是否安装
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 创建输出目录
mkdir -p api/protobuf

# 删除旧的生成文件
rm -f api/protobuf/*.pb.go

# 生成 Go 代码
echo "Generating protobuf files..."

# 生成 host.proto
if [ -f "api/protobuf_base/host.proto" ]; then
    protoc \
        --proto_path=api/protobuf_base \
        --go_out=api/protobuf \
        --go_opt=paths=source_relative \
        --go-grpc_out=api/protobuf \
        --go-grpc_opt=paths=source_relative \
        host.proto
    echo "  - api/protobuf/host.pb.go"
    echo "  - api/protobuf/host_grpc.pb.go"
fi

# 生成 command.proto
if [ -f "api/protobuf_base/command.proto" ]; then
    protoc \
        --proto_path=api/protobuf_base \
        --go_out=api/protobuf \
        --go_opt=paths=source_relative \
        --go-grpc_out=api/protobuf \
        --go-grpc_opt=paths=source_relative \
        command.proto
    echo "  - api/protobuf/command.pb.go"
    echo "  - api/protobuf/command_grpc.pb.go"
fi

echo "Protobuf files generated successfully!"