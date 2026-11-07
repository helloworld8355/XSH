# 玄枢 XSH

> Windows 多合一 CLI 工具 — 系统体检 · 故障排查 · 开发环境审计 · 授权安全评估 · 交互式 Shell

一个 Go 编写的单二进制工具，同时提供 **MSF 风格交互式 Shell** 与 **单命令模式**。

```
__  __  ____   _   _
\ \/ / / ___| | | | |
 \  /  \___ \ | |_| |
 /  \   ___) ||  _  |
/_/\_\ |____/ |_| |_|
  玄枢 XSH — 多合一 CLI 工具
```

## 快速开始

```powershell
# 构建（需要 Go 1.22+，Windows 10/11）
go build -o xsh.exe .
# 或使用脚本（gofmt/vet/test/build 一条龙）
powershell -NoProfile -ExecutionPolicy Bypass -File .\build.ps1

# 进入交互式 Shell（推荐）
.\xsh.exe

# 选择启动横幅样式：simple 简洁 | normal 普通（默认）| cool 炫酷
.\xsh.exe -b cool

# 单命令模式
.\xsh.exe check          # 系统体检
.\xsh.exe diag           # 故障排查向导
.\xsh.exe dev            # 开发环境体检
.\xsh.exe -e "ls | head -5"   # 执行单条管道命令
```

## 功能总览

| 模块 | 命令 | 说明 |
|------|------|------|
| 系统体检 | `xsh check` | 14 项检查（CPU/内存/磁盘/SMART/电池/服务/启动项/事件…）+ 综合评分 A-D，支持 `--json` / `--html` 报告 |
| 故障排查 | `xsh diag [模块]` | 交互向导 + `net` `disk` `perf` `event` `bsod` `driver` `boot` 分模块诊断（蓝屏代码解释、驱动异常筛查等） |
| 开发环境 | `xsh dev` | 57 个工具链探测（版本/路径/环境变量）+ PATH 审计（重复/失效条目） |
| 安全评估 | `xsh sec <模块>` | **仅限已授权目标**，见下文 |
| 工具集 | `fetch` `hash` `enc` `dec` `clip` `serve` `wifi` `startup` `snapshot` | 多线程断点续传下载、哈希、编解码、剪贴板、静态 HTTP 服务、Wi-Fi 密码、启动项管理、系统快照 |

## 交互式 Shell

裸运行 `xsh` 进入 REPL：Tab 补全、历史记录、管道、别名（PowerShell 风格）。

```text
xsh D:\project> ls | where size gt 1MB | sort size desc
xsh D:\project> cat app.log | grep -i error | head -20
xsh D:\project> ipconfig | first 3
xsh D:\project> ports | where state eq LISTEN
xsh D:\project> help        # 全部命令
```

内置 60+ 命令：文件（ls/cat/cp/mv/rm/mkdir/find/stat/touch/diff）、文本（grep/head/tail/wc/cut/uniq）、
数据（fromjson/tojson/fromcsv/tocsv）、动词（where/sort/select/group/measure/first/last）、
网络（ipconfig/ping/dns/curl/ports）、系统（ps/kill/svc/env/history/run）、工具集（fetch/hash/clip/serve/wifi/startup/snapshot…）。

管道动词与 PowerShell 风格对齐：`where` ≈ `Where-Object`，`select` ≈ `Select-Object`，`ConvertTo-Json`、`Select-String` 等别名可直接使用。

## 安全评估模块（sec）

> ⚠️ **仅限对你拥有书面授权的系统使用。** 所有操作强制三重门禁：
> 1. 离线 Ed25519 license 验签（机器指纹绑定可选）
> 2. 显式声明 `--authorized`
> 3. 全量审计日志（`%APPDATA%\xsh\audit\YYYYMMDD.jsonl`，含被授权方/目标/结果）

```powershell
xsh sec scan 192.168.1.10 --license C:\path\to\license --authorized
xsh sec scan 192.168.1.10 --full                    # 全端口
xsh sec hosts 192.168.1.0/24                        # TCP 探针主机发现
xsh sec dir https://target.example.com --qps 20     # 目录枚举（限速）
xsh sec tls target.example.com:443                  # 证书与 TLS 版本
xsh sec web https://target.example.com              # 安全响应头
xsh sec cors https://target.example.com             # CORS 策略
xsh sec methods https://target.example.com          # 危险方法
xsh sec axfr example.com                            # DNS 区域传送
xsh sec subdomain example.com                       # 子域枚举
xsh sec weakpass 192.168.1.10 ssh 22 --authorized   # 弱口令（ssh/smb/mysql，命中即停）
```

明确**不做**：漏洞利用执行、DoS/压力测试、ARP 欺骗、原始套接字伪造。主机发现仅用 TCP connect 探针；
目录枚举/弱口令均内置限速；弱口令仅支持 ssh/smb/mysql 且命中即停。

### License 体系

- `tools/keygen` 生成 Ed25519 密钥对：私钥存 `secret\xsh_signing_key.txt`（**不入库**），公钥内嵌于 `internal/license/pub.go`
- 签发：`go run ./tools/keygen secret issue payload.json out.license`
  payload 示例：`{"licensee":"甲方名称","expiry":"2027-12-31","features":["scan"],"machine":"<可选机器指纹>"}`
- 校验：`xsh license C:\path\to\license`
- 默认查找位置：`%USERPROFILE%\.xsh\license`、`%APPDATA%\xsh\license`

## 常用工具

```powershell
xsh fetch https://example.com/big.zip -o d:\big.zip --threads 8   # 断点续传
xsh hash sha256 file.iso                                          # 流式哈希
xsh enc base64 文本        / xsh dec base64 xxx                    # 编解码
xsh clip "文本"            / xsh clip                              # 剪贴板读写
xsh serve D:\share -p 8000                                        # 静态 HTTP 服务
xsh wifi                     / xsh wifi "SSID"                     # Wi-Fi 信息/密码（后者需管理员）
xsh startup list / add / remove / backup / restore                # 启动项管理
xsh snapshot save / list / show                                    # 系统快照
```

## 配置文件

`%APPDATA%\xsh\config.json`（首次运行自动生成默认值）：

```json
{
  "scan_rate_per_sec": 500,
  "weakpass_rate_per_sec": 5,
  "dirscan_rate_per_sec": 50,
  "vuln_db_mirror": "",
  "probe_domain": "www.msftconnecttest.com",
  "probe_url": "https://www.msftconnecttest.com/connecttest.txt",
  "dns_resolver": ""
}
```

## 项目结构

```
cmd/                 cobra 命令树（root/shell/check/diag/dev/sec/license/tools）
internal/
  shell/             命令注册表 + 管道执行引擎 + REPL（readline 补全）
  shell/builtins/    60+ 内置命令
  pipeline/          管道解析 + 记录/Schema + 动词（where/sort/group/measure…）
  syscheck/          14 项系统体检 + 评分渲染（文本/JSON/HTML）
  diag/              8 模块故障排查（net/disk/perf/event/bsod/driver/boot + 向导）
  devenv/            57 工具链探测器 + PATH 审计
  secscan/           授权安全评估（端口扫描/漏配匹配/TLS/Web/CORS/AXFR/弱口令/主机发现）
  toolset/           独立工具（下载/哈希/编码/剪贴板/服务/Wi-Fi/启动项/快照）
  license/           Ed25519 离线授权（验签/机器指纹）
  sysinfo/           系统信息（WMI/gopsutil：SMART/电池/服务/网关…）
  audit/             JSONL 审计日志
  config/  ui/  winutil/
tools/keygen/        授权密钥生成与 license 签发
```

## 构建

```powershell
# gofmt + go vet + go test + go build -ldflags "-s -w"
powershell -NoProfile -ExecutionPolicy Bypass -File .\build.ps1
# 或直接
go build -ldflags "-s -w" -o xsh.exe .
```

需要：Go 1.22+、Windows 10/11。依赖均为纯 Go 库（gopsutil、cobra、readline、miekg/dns、go-smb2、x/sys 等）。

## 合规提示

sec 模块设计为**授权评估**用途（企业内网资产巡检、甲方渗透测试交付等）。使用前请确认：
- 你已获得目标系统所有者的**书面授权**
- 遵守当地法律法规（如《网络安全法》《刑法》第 285/286 条相关规定）

审计日志不可关闭，请妥善保管授权文件与日志。

## 免责声明

本项目仅供学习研究与**授权**安全评估使用。使用者需遵守所在地区法律法规，对任何滥用本工具造成的后果自行承担责任。sec 模块内置 license 门禁、`--authorized` 显式声明与不可关闭的审计日志，未授权目标无法运行。

## License

[MIT](LICENSE) © 2026 helloworld8355
