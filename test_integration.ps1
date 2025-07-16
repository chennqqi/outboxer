# Outboxer 集成测试脚本 (PowerShell)
# 用于测试 RabbitMQ+MongoDB 和 AMQP+MongoDB 模式

param(
    [string]$TestType = "all",
    [switch]$StartOnly,
    [switch]$StopOnly,
    [switch]$Cleanup,
    [switch]$Help
)

# 颜色函数
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# 显示使用说明
function Show-Usage {
    Write-Host "用法: .\test_integration.ps1 [参数]"
    Write-Host ""
    Write-Host "参数:"
    Write-Host "  -TestType <type>        指定测试类型 (默认: all)"
    Write-Host "  -StartOnly              仅启动服务"
    Write-Host "  -StopOnly               仅停止服务"
    Write-Host "  -Cleanup                清理环境并退出"
    Write-Host "  -Help                   显示帮助信息"
    Write-Host ""
    Write-Host "测试类型:"
    Write-Host "  mongodb-rabbitmq        MongoDB + RabbitMQ 测试"
    Write-Host "  mongodb-amqp            MongoDB + AMQP 测试"
    Write-Host "  mongodb-comparison      MongoDB 性能对比测试"
    Write-Host "  mongodb-transaction     MongoDB 事务测试"
    Write-Host "  mongodb-all             所有 MongoDB 测试"
    Write-Host "  all                     所有集成测试 (默认)"
    Write-Host ""
    Write-Host "示例:"
    Write-Host "  .\test_integration.ps1                           # 运行所有集成测试"
    Write-Host "  .\test_integration.ps1 -TestType mongodb-rabbitmq # 仅运行 MongoDB + RabbitMQ 测试"
    Write-Host "  .\test_integration.ps1 -StartOnly                # 仅启动服务"
    Write-Host "  .\test_integration.ps1 -Cleanup                  # 清理环境"
}

# 检查依赖
function Test-Dependencies {
    Write-Info "检查依赖..."
    
    $dependencies = @("docker", "docker-compose", "go")
    $missing = @()
    
    foreach ($dep in $dependencies) {
        if (!(Get-Command $dep -ErrorAction SilentlyContinue)) {
            $missing += $dep
        }
    }
    
    if ($missing.Count -gt 0) {
        Write-Error "缺少依赖: $($missing -join ', ')"
        Write-Host "请安装以下工具:"
        Write-Host "- Docker Desktop: https://www.docker.com/products/docker-desktop"
        Write-Host "- Go: https://golang.org/dl/"
        exit 1
    }
    
    Write-Info "依赖检查通过"
}

# 启动服务
function Start-Services {
    Write-Info "启动集成测试服务..."
    
    try {
        docker-compose -f docker-compose.integration.yml up -d
        if ($LASTEXITCODE -ne 0) {
            throw "Docker Compose 启动失败"
        }
        
        Write-Info "等待服务就绪..."
        Start-Sleep -Seconds 20
        
        # 检查服务状态
        $services = docker-compose -f docker-compose.integration.yml ps
        if ($services -notmatch "Up") {
            Write-Error "服务启动失败"
            docker-compose -f docker-compose.integration.yml logs
            exit 1
        }
        
        Write-Info "服务启动成功"
    }
    catch {
        Write-Error "启动服务时出错: $_"
        exit 1
    }
}

# 停止服务
function Stop-Services {
    Write-Info "停止集成测试服务..."
    docker-compose -f docker-compose.integration.yml down
}

# 清理环境
function Clear-Environment {
    Write-Info "清理测试环境..."
    docker-compose -f docker-compose.integration.yml down -v
    docker system prune -f
}

# 运行测试
function Invoke-Tests {
    param([string]$Type)
    
    # 设置环境变量
    $env:MONGO_URI = "mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
    $env:AMQP_URI = "amqp://guest:guest@localhost:5672/"
    $env:POSTGRES_URI = "postgres://postgres:password@localhost:5432/outboxer_test?sslmode=disable"
    
    try {
        switch ($Type) {
            "mongodb-rabbitmq" {
                Write-Info "运行 MongoDB + RabbitMQ 集成测试..."
                go test --tags=integration -v -run "TestIntegration_MongoDB_RabbitMQ" ./...
            }
            "mongodb-amqp" {
                Write-Info "运行 MongoDB + AMQP 集成测试..."
                go test --tags=integration -v -run "TestIntegration_MongoDB_AMQP" ./...
            }
            "mongodb-comparison" {
                Write-Info "运行 MongoDB 性能对比测试..."
                go test --tags=integration -v -run "TestIntegration_MongoDB_Comparison" ./...
            }
            "mongodb-transaction" {
                Write-Info "运行 MongoDB 事务测试..."
                go test --tags=integration -v -run "TestIntegration_MongoDB_TransactionSupport" ./...
            }
            "mongodb-all" {
                Write-Info "运行所有 MongoDB 集成测试..."
                go test --tags=integration -v -run "TestIntegration_MongoDB" ./...
            }
            "all" {
                Write-Info "运行所有集成测试..."
                go test --tags=integration -v ./...
            }
            default {
                Write-Error "未知的测试类型: $Type"
                Show-Usage
                exit 1
            }
        }
        
        if ($LASTEXITCODE -ne 0) {
            throw "测试执行失败"
        }
    }
    catch {
        Write-Error "运行测试时出错: $_"
        throw
    }
}

# 主函数
function Main {
    if ($Help) {
        Show-Usage
        return
    }
    
    if ($Cleanup) {
        Clear-Environment
        return
    }
    
    if ($StopOnly) {
        Stop-Services
        return
    }
    
    if ($StartOnly) {
        Test-Dependencies
        Start-Services
        Write-Info "服务已启动，可以手动运行测试"
        Write-Info "MongoDB: mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
        Write-Info "RabbitMQ: amqp://guest:guest@localhost:5672/"
        Write-Info "RabbitMQ Management: http://localhost:15672 (guest/guest)"
        return
    }
    
    # 完整测试流程
    Test-Dependencies
    
    try {
        Start-Services
        Invoke-Tests -Type $TestType
        Write-Info "测试通过！"
    }
    catch {
        Write-Error "测试失败！"
        Write-Error $_.Exception.Message
        exit 1
    }
    finally {
        Stop-Services
    }
}

# 运行主函数
Main