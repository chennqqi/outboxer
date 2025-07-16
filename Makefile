NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
SERVICE_NAME=dna-srv

.PHONY: all test lint
all: test

test:
	@echo "$(OK_COLOR)==> Running tests$(NO_COLOR)"
	@go test -v -cover -covermode=atomic -coverprofile=tests.out ./...

test-integration:
	@echo "$(OK_COLOR)==> Running integration tests$(NO_COLOR)"
	@go test --tags=integration -v -cover -covermode=atomic -coverprofile=tests.out ./...

test-mongodb:
	@echo "$(OK_COLOR)==> Running MongoDB integration tests$(NO_COLOR)"
	@go test --tags=integration -v -run "TestIntegration_MongoDB" ./...

test-mongodb-rabbitmq:
	@echo "$(OK_COLOR)==> Running MongoDB + RabbitMQ integration tests$(NO_COLOR)"
	@go test --tags=integration -v -run "TestIntegration_MongoDB_RabbitMQ" ./...

test-mongodb-amqp:
	@echo "$(OK_COLOR)==> Running MongoDB + AMQP integration tests$(NO_COLOR)"
	@go test --tags=integration -v -run "TestIntegration_MongoDB_AMQP" ./...

test-mongodb-comparison:
	@echo "$(OK_COLOR)==> Running MongoDB performance comparison tests$(NO_COLOR)"
	@go test --tags=integration -v -run "TestIntegration_MongoDB_Comparison" ./...

# Docker 相关命令
docker-up:
	@echo "$(OK_COLOR)==> Starting integration test services$(NO_COLOR)"
	@docker-compose -f docker-compose.integration.yml up -d

docker-down:
	@echo "$(OK_COLOR)==> Stopping integration test services$(NO_COLOR)"
	@docker-compose -f docker-compose.integration.yml down

docker-logs:
	@echo "$(OK_COLOR)==> Showing integration test services logs$(NO_COLOR)"
	@docker-compose -f docker-compose.integration.yml logs -f

docker-clean:
	@echo "$(OK_COLOR)==> Cleaning up integration test services$(NO_COLOR)"
	@docker-compose -f docker-compose.integration.yml down -v
	@docker system prune -f

# 完整的集成测试流程 (Linux/Mac)
integration-test-full: docker-up
	@echo "$(OK_COLOR)==> Waiting for services to be ready$(NO_COLOR)"
	@sleep 15
	@echo "$(OK_COLOR)==> Running full integration tests$(NO_COLOR)"
	@MONGO_URI="mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin" \
	 AMQP_URI="amqp://guest:guest@localhost:5672/" \
	 POSTGRES_URI="postgres://postgres:password@localhost:5432/outboxer_test?sslmode=disable" \
	 go test --tags=integration -v ./...
	@$(MAKE) docker-down

# MongoDB 专项测试 (Linux/Mac)
integration-test-mongodb: docker-up
	@echo "$(OK_COLOR)==> Waiting for services to be ready$(NO_COLOR)"
	@sleep 15
	@echo "$(OK_COLOR)==> Running MongoDB integration tests$(NO_COLOR)"
	@MONGO_URI="mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin" \
	 AMQP_URI="amqp://guest:guest@localhost:5672/" \
	 go test --tags=integration -v -run "TestIntegration_MongoDB" ./...
	@$(MAKE) docker-down

# Windows PowerShell 集成测试命令
integration-test-windows:
	@echo "$(OK_COLOR)==> Running integration tests on Windows$(NO_COLOR)"
	@powershell -ExecutionPolicy Bypass -File test_integration.ps1

integration-test-mongodb-windows:
	@echo "$(OK_COLOR)==> Running MongoDB integration tests on Windows$(NO_COLOR)"
	@powershell -ExecutionPolicy Bypass -File test_integration.ps1 -TestType mongodb-all

integration-test-rabbitmq-windows:
	@echo "$(OK_COLOR)==> Running MongoDB + RabbitMQ tests on Windows$(NO_COLOR)"
	@powershell -ExecutionPolicy Bypass -File test_integration.ps1 -TestType mongodb-rabbitmq

integration-test-amqp-windows:
	@echo "$(OK_COLOR)==> Running MongoDB + AMQP tests on Windows$(NO_COLOR)"
	@powershell -ExecutionPolicy Bypass -File test_integration.ps1 -TestType mongodb-amqp

# Windows Docker 管理命令
docker-up-windows:
	@echo "$(OK_COLOR)==> Starting integration test services on Windows$(NO_COLOR)"
	@docker-compose -f docker-compose.integration.yml up -d

docker-down-windows:
	@echo "$(OK_COLOR)==> Stopping integration test services on Windows$(NO_COLOR)"
	@docker-compose -f docker-compose.integration.yml down

# 帮助命令
help-integration:
	@echo "$(OK_COLOR)==> Integration Test Commands$(NO_COLOR)"
	@echo "Linux/Mac:"
	@echo "  make integration-test-full      - 运行完整集成测试"
	@echo "  make integration-test-mongodb   - 运行 MongoDB 集成测试"
	@echo "  make test-mongodb-rabbitmq      - 运行 MongoDB + RabbitMQ 测试"
	@echo "  make test-mongodb-amqp          - 运行 MongoDB + AMQP 测试"
	@echo ""
	@echo "Windows:"
	@echo "  make integration-test-windows         - 运行完整集成测试"
	@echo "  make integration-test-mongodb-windows - 运行 MongoDB 集成测试"
	@echo "  make integration-test-rabbitmq-windows- 运行 MongoDB + RabbitMQ 测试"
	@echo "  make integration-test-amqp-windows    - 运行 MongoDB + AMQP 测试"
	@echo ""
	@echo "PowerShell 直接使用:"
	@echo "  .\\test_integration.ps1                        - 运行所有测试"
	@echo "  .\\test_integration.ps1 -TestType mongodb-rabbitmq - 运行特定测试"
	@echo "  .\\test_integration.ps1 -StartOnly              - 仅启动服务"

setup:
	@echo "$(OK_COLOR)==> Setting up deps$(NO_COLOR)"
	@awslocal kinesis create-stream --stream-name test --shard-count 1
