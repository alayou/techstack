# Techstack 开发规划

## 规划目的

这份文档面向实现层，目标是把 README 里的方向拆到当前代码结构上，方便逐步落地。

当前代码里已经有这些基础对象可以复用：

- `model.PublicRepo`
- `model.Package`
- `model.PackageVersion`
- `model.RepoDependency`
- `model.RepoTechAnalysis`
- `model.UserRepoStar`
- `httpd/taskmgr/*` 异步任务框架

因此后续规划不应该推翻现有模型，而应该以“增量扩展”为主。

## 一期重点

建议先做下面四件事，这四件事是后续 MCP、推荐和 skill 生成的基础：

- 补齐包收藏能力
- 把仓库收藏和包收藏统一沉淀为偏好模型
- 建立 AI 技术栈仓库分类体系
- 预留安全数据关联模型

原因很直接：

- 没有偏好模型，Code Agent 无法“优先选用开发者认可的包或仓库”
- 没有分类体系，推荐与检索只能停留在字符串匹配
- 没有安全模型，后续推荐会缺少基本治理约束
- 没有统一数据模型，MCP 接口只会变成一层薄包装

## 数据模型规划

建议在现有模型基础上新增下面几类表。

### 偏好相关

- `user_package_star`
  用途：记录用户收藏的包。
  建议字段：`id`、`user_id`、`package_id`、`created_at`。

- `user_stack_preference`
  用途：记录开发者显式偏好或系统聚合后的偏好结果。
  建议字段：`id`、`user_id`、`target_type`、`target_id`、`signal_type`、`score`、`reason`、`created_at`、`updated_at`。

- `user_stack_preference_snapshot`
  用途：保存可直接给 Agent 使用的偏好快照，避免每次在线聚合。
  建议字段：`id`、`user_id`、`snapshot_json`、`generated_at`。

说明：

- 现有 `user_repo_star` 可以继续保留，作为偏好信号来源之一。
- `user_stack_preference` 不只是“收藏”，还应该能容纳导入频次、手动认可、实际使用等后续信号。

### 仓库分类相关

- `repo_category`
  用途：定义仓库分类，如 agent framework、mcp server、rag、workflow engine、vector store、observability。
  建议字段：`id`、`name`、`slug`、`description`、`created_at`、`updated_at`。

- `public_repo_category_rel`
  用途：建立公共仓库与分类的多对多关系。
  建议字段：`id`、`public_repo_id`、`category_id`、`source`、`confidence`、`created_at`。

- `public_repo_tag`
  用途：补充轻量标签体系，支持快速检索与推荐排序。
  建议字段：`id`、`public_repo_id`、`tag`、`source`、`created_at`。

说明：

- 分类适合表达稳定结构，标签适合表达快速补充的特征。
- `source` 应区分人工标注、规则识别、AI 识别。

### 安全相关

- `package_security_advisory`
  用途：保存外部安全公告的标准化数据。
  建议字段：`id`、`package_id`、`package_version_id`、`advisory_id`、`severity`、`summary`、`source`、`published_at`、`updated_at`。

- `package_security_snapshot`
  用途：保存当前包或版本的聚合安全状态，便于查询。
  建议字段：`id`、`package_id`、`package_version_id`、`risk_level`、`advisory_count`、`snapshot_json`、`updated_at`。

- `repo_security_summary`
  用途：从仓库依赖层面汇总风险。
  建议字段：`id`、`repo_type`、`repo_id`、`risk_level`、`summary`、`updated_at`。

说明：

- 安全数据不应只挂在仓库层，也要能回溯到包和版本层。
- 后续 Agent 做选型时，应该优先读取 `package_security_snapshot` 和 `repo_security_summary`。

### Skill 生成相关

- `generated_skill`
  用途：保存按项目生成的 skill 结果。
  建议字段：`id`、`repo_id`、`repo_type`、`user_id`、`title`、`skill_type`、`content`、`status`、`created_at`、`updated_at`。

- `generated_skill_source`
  用途：记录 skill 来源，便于追踪其依据。
  建议字段：`id`、`skill_id`、`source_type`、`source_id`、`weight`。

说明：

- skill 生成不能只依赖单一仓库分析，应能联合偏好、分类、安全信息和已有仓库样本。

## 接口规划

建议优先补齐 HTTP API，再在此基础上输出 MCP tools。

### HTTP API 建议

- `POST /api/v1/c/libraries/{package_id}/star`
  收藏包。

- `DELETE /api/v1/c/libraries/{package_id}/star`
  取消收藏包。

- `GET /api/v1/c/user/star-packages`
  查询用户收藏包列表。

- `GET /api/v1/c/preferences`
  查询用户偏好快照。

- `POST /api/v1/c/public-repos/{id}/classify`
  触发仓库分类任务。

- `GET /api/v1/c/public-repos/{id}/categories`
  查询仓库分类和标签。

- `GET /api/v1/c/libraries/{package_id}/security`
  查询包安全信息。

- `GET /api/v1/c/public-repos/{id}/security`
  查询仓库安全汇总。

- `POST /api/v1/c/skills/generate`
  触发 skill 生成任务。

- `GET /api/v1/c/skills`
  查询 skill 列表。

- `GET /api/v1/c/skills/{id}`
  查询 skill 详情。

### MCP Tools 建议

MCP 初期不要做得太大，建议先做下面这些只读工具：

- `techstack.search_packages`
  搜索包，并返回生态、版本、基础描述和收藏状态。

- `techstack.search_repositories`
  搜索仓库，并返回分类、分析摘要、收藏状态和风险摘要。

- `techstack.get_repository_analysis`
  查询单仓库技术分析。

- `techstack.get_preference_snapshot`
  查询用户当前偏好快照。

- `techstack.get_package_security`
  查询包或版本的安全信息。

- `techstack.get_recommended_choices`
  返回当前项目上下文下的优先候选包和仓库。

- `techstack.get_project_skill`
  查询某个项目已生成的 skill。

说明：

- 初期尽量先做只读 MCP tools，避免过早开放写接口。
- 如果后续增加写操作，也应该优先支持“创建任务”，而不是直接同步执行长任务。

## 异步任务规划

当前已经有任务框架，建议新增下列任务类型：

- `repo_classify`
  对导入仓库做分类、标签提取和候选用途识别。

- `package_security_sync`
  同步包级和版本级安全数据。

- `repo_security_refresh`
  根据仓库依赖重算仓库安全汇总。

- `preference_snapshot_refresh`
  重算用户偏好快照。

- `discover_recommendations`
  发现优秀包和仓库，并生成候选推荐结果。

- `generate_project_skill`
  按项目上下文生成 skill。

## 推荐系统原则

“自动发现优秀包和仓库”不应该等于“系统直接决定采用什么技术”。

推荐系统建议同时考虑：

- 开发者显式收藏
- 开发者已有仓库中的真实依赖
- 仓库分类是否匹配当前项目
- 包和仓库的安全风险
- 仓库分析结论是否稳定且适用

推荐输出建议分成三层：

- 已认可
  开发者已收藏或已明确认可。

- 可推荐
  系统认为质量较高，但尚未得到开发者确认。

- 不建议
  存在明显安全、维护或适配问题。

## Code Agent 约束策略

后续在 Agent 侧要坚持下面几条约束：

- 优先使用开发者已收藏或已认可的包与仓库。
- 如果必须超出偏好范围，Agent 应显式说明原因。
- 安全风险高的依赖不应作为默认推荐项。
- 生成的 skill 应体现团队已有代码风格、依赖习惯和技术边界。

这意味着 Techstack 输出的不只是“信息”，还应该是带排序和约束的信息。

## 建议落地顺序

建议按下面顺序推进：

1. 增加包收藏与统一偏好模型。
2. 增加仓库分类和标签体系。
3. 增加偏好快照查询接口。
4. 增加安全数据模型和基础同步任务。
5. 增加只读 MCP tools。
6. 增加候选推荐任务与推荐查询接口。
7. 增加项目 skill 生成任务。
8. 把推荐、偏好和安全信息整合到 Code Agent 选择策略。

## 当前建议先做的具体任务

- [ ] 为包收藏新增模型、DAO、Handler 和路由
- [ ] 把仓库收藏与包收藏统一沉淀到偏好聚合逻辑
- [ ] 设计偏好快照结构，并提供查询接口
- [ ] 设计仓库分类和标签模型
- [ ] 增加仓库分类任务入口
- [ ] 设计安全公告与安全快照模型
- [ ] 确定外部安全数据源接入方式
- [ ] 定义第一批 MCP tools 的输入输出结构

## 待确认问题

- “优秀包和仓库”的判断标准由谁定义，是否允许不同团队有不同权重
- MCP 服务是内嵌在当前进程，还是单独拆服务
- skill 的产物格式是 Markdown、JSON、YAML，还是多种并存
- 安全数据优先接入哪个来源，以及更新频率怎么控制
- 偏好快照是实时计算还是异步生成
