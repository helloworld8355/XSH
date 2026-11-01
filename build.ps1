# 玄枢 XSH 构建脚本
# 用法: .\build.ps1 [-Clean]
#   -Clean  构建前清理（删除 xsh.exe 与测试缓存）
param([switch]$Clean)

$ErrorActionPreference = 'Stop'

# 定位 Go：优先 PATH，其次默认安装位置
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    $candidates = @(
        "C:\Program Files\Go\bin",
        "$env:LOCALAPPDATA\Programs\Go\bin",
        "C:\Go\bin"
    )
    foreach ($c in $candidates) {
        if (Test-Path "$c\go.exe") { $env:Path = "$c;$env:Path"; break }
    }
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "[x] 未找到 Go，请安装 Go 1.22+ 或将其加入 PATH" -ForegroundColor Red
        exit 1
    }
}

if ($Clean) {
    Write-Host "==> 清理..." -ForegroundColor Cyan
    if (Test-Path .\xsh.exe) { Remove-Item .\xsh.exe -Force }
    go clean -testcache
}

Write-Host "==> gofmt 检查..." -ForegroundColor Cyan
$fmt = gofmt -l .
if ($fmt) {
    Write-Host "[!] 以下文件未格式化，已自动格式化:" -ForegroundColor Yellow
    $fmt | ForEach-Object { Write-Host "    $_" }
    gofmt -w $fmt
}

Write-Host "==> go vet..." -ForegroundColor Cyan
go vet ./...
if ($LASTEXITCODE -ne 0) { Write-Host "[x] vet 失败" -ForegroundColor Red; exit 1 }

Write-Host "==> go test..." -ForegroundColor Cyan
go test ./...
if ($LASTEXITCODE -ne 0) { Write-Host "[x] 测试失败" -ForegroundColor Red; exit 1 }

Write-Host "==> go build..." -ForegroundColor Cyan
$ldflags = "-s -w"
go build -ldflags $ldflags -o xsh.exe .
if ($LASTEXITCODE -ne 0) { Write-Host "[x] 构建失败" -ForegroundColor Red; exit 1 }

$size = "{0:N1} MB" -f ((Get-Item .\xsh.exe).Length / 1MB)
Write-Host "[+] 构建成功: xsh.exe ($size) — 玄枢 XSH v1.0.0" -ForegroundColor Green
Write-Host "    运行 .\xsh.exe 进入交互式 Shell"
