#!/bin/bash

# Outboxer 集成测试脚本
# 用于测试 RabbitMQ+MongoDB 和 AMQP+MongoDB 模式

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        exit 1
    fi
    
    log_info "依赖检查通过"
}

# 启动服务
start_services() {
    log_info "启动集成测试服务..."
    docker-compose -f docker-compose.integration.yml up -d
    
    log_info "等待服务就绪..."
    sleep 20
    
    # 检查服务状态
    if ! docker-compose -f docker-compose.integration.yml ps | grep -q "Up"; then
        log_error "服务启动失败"
        docker-compose -f docker-compose.integration.yml logs
        exit 1
    fi
    
    log_info "服务启动成功"
}

# 停止服务
stop_services() {
    log_info "停止集成测试服务..."
    docker-compose -f docker-compose.integration.yml down
}

# 清理环境
cleanup() {
    log_info "清理测试环境..."
    docker-compose -f docker-compose.integration.yml down -v
    docker system prune -f
}

# 运行测试
run_tests() {
    local test_type=$1
    
    # 设置环境变量
    export MONGO_URI="mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
    export AMQP_URI="amqp://guest:guest@localhost:5672/"
    export POSTGRES_URI="postgres://postgres:password@localhost:5432/outboxer_test?sslmode=disable"
    
    case $test_type in
        "mongodb-rabbitmq")
            log_info "运行 MongoDB + RabbitMQ 集成测试..."
            go test --tags=integration -v -run "TestIntegration_MongoDB_RabbitMQ" ./...
            ;;
        "mongodb-amqp")
            log_info "运行 MongoDB + AMQP 集成测试..."
            go test --tags=integration -v -run "TestIntegration_MongoDB_AMQP" ./...
            ;;
        "mongodb-comparison")
            log_info "运行 MongoDB 性能对比测试..."
            go test --tags=integration -v -run "TestIntegration_MongoDB_Comparison" ./...
            ;;
        "mongodb-transaction")
            log_info "运行 MongoDB 事务测试..."
            go test --tags=integration -v -run "TestIntegration_MongoDB_TransactionSupport" ./...
            ;;
        "mongodb-all")
            log_info "运行所有 MongoDB 集成测试..."
            go test --tags=integration -v -run "TestIntegration_MongoDB" ./...
            ;;
        "all")
            log_info "运行所有集成测试..."
            go test --tags=integration -v ./...
            ;;
        *)
            log_error "未知的测试类型: $test_type"
            show_usage
            exit 1
            ;;
    esac
}

# 显示使用说明
show_usage() {
    echo "用法: $0 [选项] [测试类型]"
    echo ""
    echo "选项:"
    echo "  -h, --help              显示帮助信息"
    echo "  -c, --cleanup           清理环境并退出"
    echo "  -s, --start-only        仅启动服务"
    echo "  -t, --stop-only         仅停止服务"
    echo ""
    echo "测试类型:"
    echo "  mongodb-rabbitmq        MongoDB + RabbitMQ 测试"
    echo "  mongodb-amqp            MongoDB + AMQP 测试"
    echo "  mongodb-comparison      MongoDB 性能对比测试"
    echo "  mongodb-transaction     MongoDB 事务测试"
    echo "  mongodb-all             所有 MongoDB 测试"
    echo "  all                     所有集成测试 (默认)"
    echo ""
    echo "示例:"
    echo "  $0                      # 运行所有集成测试"
    echo "  $0 mongodb-rabbitmq     # 仅运行 MongoDB + RabbitMQ 测试"
    echo "  $0 -s                   # 仅启动服务"
    echo "  $0 -c                   # 清理环境"
}

# 主函数
main() {
    local test_type="all"
    local start_only=false
    local stop_only=false
    local cleanup_only=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -c|--cleanup)
                cleanup_only=true
                shift
                ;;
            -s|--start-only)
                start_only=true
                shift
                ;;
            -t|--stop-only)
                stop_only=true
                shift
                ;;
            mongodb-rabbitmq|mongodb-amqp|mongodb-comparison|mongodb-transaction|mongodb-all|all)
                test_type=$1
                shift
                ;;
            *)
                log_error "未知参数: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # 执行操作
    if [[ $cleanup_only == true ]]; then
        cleanup
        exit 0
    fi
    
    if [[ $stop_only == true ]]; then
        stop_services
        exit 0
    fi
    
    if [[ $start_only == true ]]; then
        check_dependencies
        start_services
        log_info "服务已启动，可以手动运行测试"
        log_info "MongoDB: mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
        log_info "RabbitMQ: amqp://guest:guest@localhost:5672/"
        log_info "RabbitMQ Management: http://localhost:15672 (guest/guest)"
        exit 0
    fi
    
    # 完整测试流程
    check_dependencies
    
    # 设置清理陷阱
    trap 'log_warn "测试被中断，正在清理..."; stop_services; exit 1' INT TERM
    
    start_services
    
    # 运行测试
    if run_tests $test_type; then
        log_info "测试通过！"
        test_result=0
    else
        log_error "测试失败！"
        test_result=1
    fi
    
    stop_services
    
    exit $test_result
}

# 运行主函数
main "$@"