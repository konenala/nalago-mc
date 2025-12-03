# Nalago-MC Makefile
# 自动化封包生成和项目管理

.PHONY: help gen-packets gen-client gen-server clean test build fmt lint

# 默认目标
help:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🎮 Nalago-MC Makefile"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "📦 封包生成:"
	@echo "  make gen-packets  - 生成所有封包 (client + server)"
	@echo "  make gen-client   - 只生成 client 封包"
	@echo "  make gen-server   - 只生成 server 封包"
	@echo ""
	@echo "🔨 开发工具:"
	@echo "  make build        - 编译项目"
	@echo "  make test         - 运行测试"
	@echo "  make fmt          - 格式化代码"
	@echo "  make lint         - 代码检查"
	@echo ""
	@echo "🧹 清理:"
	@echo "  make clean        - 清理生成的文件"
	@echo ""

# 配置变量
PROTOCOL_JSON = E:/bot編寫/go-mc/minecraft-data-pc-1_21_10/data/pc/1.21.10/protocol.json
GENERATOR = tools/enhanced-generator/main_v2.go
OUTPUT_BASE = pkg/protocol/packet/game

# 生成所有封包
gen-packets:
	@echo "🚀 开始生成所有封包..."
	@cd tools/enhanced-generator && bash generate.sh

# 只生成 client 封包
gen-client:
	@echo "📦 生成 Client 封包..."
	@go run $(GENERATOR) \
		-protocol "$(PROTOCOL_JSON)" \
		-output "$(OUTPUT_BASE)/client" \
		-direction client \
		-codec=true \
		-v
	@cd $(OUTPUT_BASE)/client && \
		for file in packet_*.go; do \
			perl -i -pe 's/(\s+)(temp, err = \(\*pk\.\w+\)\(&)p\./\1\2s./g; s/(\s+)p\.(\w+) = /\1s.\2 = /g; s/(\s+)(temp, err = )p\./\1\2s./g' "$$file"; \
		done
	@echo "✅ Client 封包生成完成"

# 只生成 server 封包
gen-server:
	@echo "📦 生成 Server 封包..."
	@go run $(GENERATOR) \
		-protocol "$(PROTOCOL_JSON)" \
		-output "$(OUTPUT_BASE)/server" \
		-direction server \
		-codec=true \
		-v
	@cd $(OUTPUT_BASE)/server && \
		for file in packet_*.go; do \
			perl -i -pe 's/(\s+)(temp, err = \(\*pk\.\w+\)\(&)p\./\1\2s./g; s/(\s+)p\.(\w+) = /\1s.\2 = /g; s/(\s+)(temp, err = )p\./\1\2s./g' "$$file"; \
		done
	@echo "✅ Server 封包生成完成"

# 编译项目
build:
	@echo "🔨 编译项目..."
	@go build ./...
	@echo "✅ 编译完成"

# 运行测试
test:
	@echo "🧪 运行测试..."
	@go test -v ./...

# 格式化代码
fmt:
	@echo "📝 格式化代码..."
	@go fmt ./...
	@echo "✅ 格式化完成"

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@golangci-lint run || echo "提示: 请安装 golangci-lint"

# 清理生成的文件
clean:
	@echo "🧹 清理生成的文件..."
	@rm -f $(OUTPUT_BASE)/client/packet_*.go
	@rm -f $(OUTPUT_BASE)/server/packet_*.go
	@rm -rf test_output/
	@echo "✅ 清理完成"

# 统计信息
stats:
	@echo "📊 项目统计:"
	@echo ""
	@echo "Client 封包:"
	@find $(OUTPUT_BASE)/client -name "packet_*.go" | wc -l | xargs echo "  总数:"
	@grep -l "// TODO" $(OUTPUT_BASE)/client/packet_*.go 2>/dev/null | wc -l | xargs echo "  有 TODO:"
	@echo ""
	@echo "Server 封包:"
	@find $(OUTPUT_BASE)/server -name "packet_*.go" | wc -l | xargs echo "  总数:"
	@grep -l "// TODO" $(OUTPUT_BASE)/server/packet_*.go 2>/dev/null | wc -l | xargs echo "  有 TODO:"

# 快速开始（首次使用）
quickstart: gen-packets build test
	@echo ""
	@echo "🎉 项目设置完成！"
	@echo ""
	@echo "💡 下一步:"
	@echo "  1. 查看生成的封包: pkg/protocol/packet/game/"
	@echo "  2. 阅读文档: tools/enhanced-generator/README.md"
	@echo "  3. 开始开发: 导入封包并使用"
	@echo ""
