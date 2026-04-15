# Techstack

[English README](README.md)

Techstack 是一个面向受约束 AI 开发流程的仓库认知与技术栈治理服务。它可以导入公共仓库、提取依赖、同步包元数据、生成结构化技术分析，但它的产品目标并不是让 AI 随意替你选择技术路线。它的核心立场很明确：把 Agent 约束在开发者已掌握的技术栈里，用 AI 倒逼理解而不是绕过理解，拒绝技术黑箱，让 AI 只在团队认知边界以内编写代码。

## 项目初衷

- 让 AI 只在开发者或团队已经理解的技术栈内工作。
- 在代码生成之前，把依赖、目录结构、核心 API 和技术选择变成可检查的信息。
- 把 AI 当成学习和实现的放大器，而不是替代理解的黑箱。
- 为 Agent 提供“边界感”：先理解团队已有技术栈，再开始写代码。

## 当前状态

这个项目目前还没有完成，当前仓库提供的是最小可用版本（MVP）。

现阶段主要聚焦在四件事：

- 把公共仓库导入到统一系统中
- 建立跨生态的全局包目录
- 从代码仓库中提取真实依赖和技术栈信号
- 生成可供后续 Agent 使用的结构化 AI 仓库分析

也就是说，现在这一版已经把“数据层”和“技术栈边界”先搭起来了，但距离完整的 Agent 原生开发工作流产品还差后续能力建设。

## 这个项目在做什么

- 维护全局依赖库目录，支持 `npm`、`golang`、`pypi`、`cargo`
- 导入公共 GitHub 仓库
- 使用 `go-git` 克隆仓库，使用 `osv-scalibr` 扫描依赖
- 建立“仓库依赖 -> 全局包目录”的索引关系
- 通过异步任务完成仓库导入与 AI 分析
- 通过 OpenAI 兼容接口生成结构化技术报告
- 同一个二进制同时提供 HTTP API 和嵌入式静态 Web 界面
- 支持登录鉴权，也支持 AK/SK 签名，方便 Agent 或自动化系统接入

## 后续规划

Techstack 的长期目标，不只是做仓库分析，而是成为代码 Agent 可直接使用的技术栈治理层。

后续计划包括：

- 增加 MCP 服务层，让 IDE、Agent 和自动化工具通过标准协议读取技术栈知识
- 让 Agent 自动发现优秀的包和仓库，但是否纳入仍由开发者控制
- 支持开发者收藏项目或包，把收藏行为作为明确的技术偏好信号
- 对 AI 技术栈相关仓库做更清晰的归类，便于检索、推荐和治理
- 对接到具体开发流程中，为 code agent 按项目生成可用 skill
- 同步并尽量实时更新技术栈数据与开发者技术偏好
- 让 code agent 在写代码时优先选择开发者偏好的包或仓库

## 开发任务清单

下面这份清单把上面的方向拆成更具体的产品和工程任务：

- [ ] 支持 MCP 服务层，让 code agent、IDE 和自动化工具可以通过标准协议查询包、仓库、偏好和分析结果。
- [ ] 支持 Agent 自动发现优秀包和仓库，但推荐结果必须可审查，不能静默纳入技术栈。
- [ ] 支持开发者收藏项目或包，并把收藏行为沉淀为明确的技术偏好信号。
- [ ] 建立 AI 技术栈相关仓库的归类体系，支持按角色、领域、框架和使用方式进行组织。
- [ ] 对接到具体开发流程中，基于已知仓库与技术栈上下文，为 code agent 按项目生成可用 skill。
- [ ] 随着仓库、依赖和收藏数据变化，同步并尽量实时更新技术栈知识与开发者技术偏好。
- [ ] 让 code agent 在写代码时优先选择开发者认可或个人偏好的包与仓库。
- [ ] 关联包的安全数据源并生成安全分析，让技术栈决策同时具备漏洞与风险视角。

## 为什么它适合约束 Agent

这个服务本质上是给 Agent 加一层技术边界。

1. 先导入能代表你真实工程实践的仓库。
2. 从源码里提取真实依赖和技术信号。
3. 让 Agent 先读取这些信号，而不是直接自由发挥。
4. 让 Agent 只在你已经掌握的技术空间里写代码。

这样带来的结果不是“AI 全权接管”，而是“开发者继续理解技术，AI 在可审计边界内加速实现”。

## 架构概览

- `daemon/`：服务启动、HTTP Server、Worker Pool 生命周期
- `httpd/`：API、鉴权、中间件、系统设置、HTTP 工具
- `httpd/taskmgr/`：仓库导入与仓库分析异步任务
- `pkg/repofs/`：仓库克隆、内存文件系统、SCALIBR 扫描
- `pkg/pkgclient/`：上游包生态客户端
- `model/`：用户、包、仓库、设置、任务等 GORM 模型
- `web/`：嵌入到二进制中的前端静态资源

## 仓库分析流程

1. 把一个公共 GitHub 仓库导入系统。
2. 后端把仓库克隆到内存文件系统。
3. 使用 `osv-scalibr` 的 `go`、`python`、`javascript`、`rust` 插件扫描依赖。
4. 依赖被保存为仓库依赖，并尝试映射到内部全局包目录。
5. AI 分析任务通过工具读取源码与文档，写回结构化分析报告。

生成的报告会覆盖这类问题：

- 这个项目是做什么的
- 它解决什么问题
- 它真实使用了哪些技术栈和依赖
- 目录结构说明了什么
- 核心 API、优势、限制、适用与不适用场景分别是什么

## 支持范围

- 依赖生态：`npm`、`golang`、`pypi`、`cargo`
- 代码内直接解析的依赖文件：`go.mod`、`package.json`、`pyproject.toml`、`requirements.txt`、`Cargo.toml`
- 公共仓库导入：当前仅支持 GitHub
- LLM：通过 `llm.baseUrl`、`llm.apiKey`、`llm.model` 配置 OpenAI 兼容接口

## 快速开始

### 环境要求

- 与 `go.mod` 兼容的 Go 工具链，当前为 `go 1.25.8`
- PostgreSQL，或你已经验证过能正常跑通本项目的数据库配置
- 一个 OpenAI 兼容的大模型接口

### 配置

复制示例配置：

```bash
cp config.yml.tpl config.yml
```

最小示例：

```yaml
addr: :8775
debug: false
database:
  driver: postgres
  dsn: user=postgres password=changeme dbname=techstack host=127.0.0.1 port=5432 sslmode=disable TimeZone=Asia/Shanghai
session_hash_key: "replace-with-16-char-key"
session_cookie_key: "replace-with-16-char-key"
llm:
  apiKey: "YOUR_API_KEY"
  baseUrl: "https://api.minimaxi.com/v1"
  provider: "openai"
  model: "MiniMax-M2.7"
```

重点配置项：

- `database.*`：GORM 使用的数据库连接
- `session_hash_key`、`session_cookie_key`：会话与 Cookie 签名必须填写
- `llm.*`：仓库 AI 分析必须填写
- `debug`：调试模式下会放开宽松的 CORS，生产环境不要开启

### 运行

构建并启动：

```bash
go build -o techstack .
./techstack daemon
```

或者直接运行：

```bash
go run . daemon
```

示例配置默认监听 `:8775`。

### 首次启动会发生什么

服务第一次启动时会自动：

- 对所有模型执行 `AutoMigrate`
- 初始化系统设置
- 创建默认管理员账号：`admin / techstack`
- 为每个用户生成 `account_key` 和 `account_secret`

默认管理员密码必须在首次登录后立即修改。

## API 概览

公共接口：

- `GET /api/v1/ping`
- `GET /api/v1/health`

登录与个人信息：

- `POST /api/v1/c/login`
- `POST /api/v1/c/signup`
- `POST /api/v1/c/logout`
- `GET /api/v1/c/profile`
- `PUT /api/v1/c/profile`
- `PUT /api/v1/c/user/password`

全局依赖库：

- `GET /api/v1/c/libraries`
- `POST /api/v1/c/libraries`
- `POST /api/v1/c/libraries/batch`
- `GET /api/v1/c/libraries/{package_id}`
- `GET /api/v1/c/libraries/{package_id}/versions`
- `POST /api/v1/c/libraries/{package_id}/sync-versions`
- `GET /api/v1/c/libraries/get/purl`

公共仓库与任务：

- `GET /api/v1/c/public-repos`
- `POST /api/v1/c/public-repos/import`
- `GET /api/v1/c/public-repos/{id}`
- `GET /api/v1/c/tasks`
- `GET /api/v1/c/tasks/{id}`
- `POST /api/v1/c/tasks/{id}/retry`

系统设置：

- `GET /api/v1/c/setting`
- `PUT /api/v1/c/setting`
- `GET /api/v1/c/setting/basic`
- `PUT /api/v1/c/setting/basic`

返回格式约定：

- 列表接口返回 `{"list":[...],"total":N}`
- 错误返回 `{"code":400,"msg":"...","error":"..."}`
- `GET /api/v1/health` 会返回 `health`、`revision`、`version`、`build_at`

## 基本使用示例

健康检查：

```bash
curl http://127.0.0.1:8775/api/v1/health
```

使用默认管理员登录：

```bash
curl -X POST http://127.0.0.1:8775/api/v1/c/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"techstack"}'
```

手动录入一个全局包：

```bash
curl -X POST http://127.0.0.1:8775/api/v1/c/libraries \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer <token>" \
  -d '{"name":"gin","purl_type":"golang"}'
```

导入一个 GitHub 仓库：

```bash
curl -X POST http://127.0.0.1:8775/api/v1/c/public-repos/import \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer <token>" \
  -d '{"repo_url":"https://github.com/gin-gonic/gin"}'
```

仓库导入后会触发异步流程。通过任务接口可以查看进度、失败原因和重试状态。

## Agent 对接

Techstack 的定位不是“让 Agent 无边界写代码”，而是“让 Agent 先理解，再在边界内写代码”。

- 人负责定义可接受的技术栈边界，方式是导入仓库、维护包目录
- 服务把这些边界转换成可查询的结构化数据
- Agent 在生成代码前，可以先查询已知依赖、仓库分析、技术模式

对于非浏览器客户端，受保护接口还支持 AK/SK 签名：

- `x-aksk-version: 1`
- `x-access-key`
- `x-timestamp`
- `x-signature`

签名校验的原始串格式为：

```text
{METHOD}&{PATH}&{sorted_query_params_without_signature}&{timestamp}
```

再对这个字符串做 HMAC-SHA256 校验。这很适合对接外部 Agent、脚本或内部自动化系统。

## 服务模式

二进制还支持作为系统服务安装和管理：

```bash
./techstack service install --config /absolute/path/to/config.yml
./techstack service start --config /absolute/path/to/config.yml
./techstack service stop --config /absolute/path/to/config.yml
./techstack service uninstall --config /absolute/path/to/config.yml
```

建议始终显式传入 `--config`。

## Docker

先构建 Dockerfile 所需的二进制，再构建镜像：

```bash
go build -o build/techstack .
docker build -t techstack:local .
docker run --rm -p 8775:8775 \
  -v "$(pwd)/config.yml:/opt/techstack/config.yml" \
  techstack:local
```

## 跨平台构建脚本

仓库内置了 `build-binary.sh`，会产出：

- `build/techstack.exe`
- `build/techstack-linux`
- `build/techstack`
- `build/techstack-linux-arm64`

示例：

```bash
./build-binary.sh v0.1.0
```

## 生产环境注意事项

- 首次启动后立刻修改默认管理员密码
- 把示例里的 session key 全部替换成随机值
- 生产环境设置 `debug: false`
- 为真实部署配置好 TLS 和 CORS
- 不要把 AI 分析结果当成绝对事实，它更适合作为结构化起点
- 这个项目的目标从来不是“AI 会写所有东西”，而是“AI 只在开发者理解范围内写东西，开发者继续对技术负责”

## License

详见 [LICENSE](LICENSE)。
