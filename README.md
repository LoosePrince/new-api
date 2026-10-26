<div align="center">

![new-api](/web/public/logo.png)

# New API · 公益免费 AI API

**把 new-api 网关改造成面向公众开放的免费公益 AI API 服务**

</div>

---

## 项目目标

本仓库基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 定制，方向不是商业 API 分销，而是**公益服务**：运营者接入自己的上游渠道，通过公开首页向所有人提供可直接使用的 AI 接口。

打开站点首页即可看到完整效果：

- **公益宣传区**：说明本站为免费公共服务，无需付费套餐。
- **公开运行数据**：展示累计请求数、累计 Token 数与最终成功率；按 request ID 聚合，最终回退成功只计一次成功，并支持在后台填写历史修正值。
- **免 Key 在线对话**：访客不登录、不填写 API Key，直接在首页与模型对话；回复以流式渲染，并保留实际回答的模型名。
- **API 接入指引**：展示 API Base、兼容端点（沿用站点的公开 API 前缀）、是否需要 Key、模型名要求、自动回退策略与推理字段说明，复制即可接入第三方客户端。

是否真的“免 Key”由分组访问策略决定：把公开分组的请求密钥要求关闭后，任意 Key 或空 Key 都能调用；需要时也可以改回必须携带有效 Key。

### 后台开启方式

系统设置 → **内容** → **公益首页**，可配置：

| 配置项 | 作用 |
| --- | --- |
| 公开运行数据 | 首页是否展示请求数 / Token / 成功率 |
| 免 Key 在线对话 | 首页是否展示公开对话入口 |
| 公开号分组 | 公开对话与 API 指引使用的分组 |
| 对话路由方式 | 跟随分组策略，或指定具体模型 |
| 指定模型 | 路由方式为“指定模型”时使用 |
| API 参考分组 | API 接入指引展示的分组状态（是否需 Key、是否允许不传模型、是否有回退） |
| 统计修正值 | 对请求数、Token 数、成功数、失败数做历史补正 |

对应的后端配置位于 `setting/public_home.go`，首页组件位于 `web/src/features/home/components/sections/`。

## 与官方 new-api 的差异

以下改动均来自本仓库作者（树梢）的提交，未改动上游既有功能的行为。

### 1. 公益首页与免 Key 分组对话

- 重构默认首页：新增公益宣传、公开运行数据、公开对话、API 接入指引四个板块。
- 公开对话**复用现有的分组 Relay 链路**（GroupEndpoint → TokenAuth → Distribute → Relay），不新增对话 API 路由。
- 修复免 Key 请求中 `UsingGroup` 与 `TokenGroup` 上下文不一致的问题；公开对话请求会跳过浏览器登录态的 `Authorization`，避免被识别成 API Key。
- 指定模型从真实模型目录加载，并按 `enable_groups` 过滤；模型列表支持完整分页。
- 公开对话支持流式解析，前端逐段渲染并记录消息模型名。
- 首页 API 指引在没有配置地址时回退到 `window.location.origin` 与默认兼容路由。
- 保存“跟随分组配置”时会自动启用该分组的无模型请求策略。

### 2. 分组访问策略与模型回退

- 分组可配置端点别名、共享端点、是否必须携带 Key，并支持 Gemini `v1beta` 路径。
- 支持“允许请求不指定模型”，并可选择轮流、随机或指定模型。
- 新增分组内模型回退：`fallback_times`（回退次数）、回退选择策略（`sequential` / `random` / `nearest`）、`fallback_group`（兜底分组）、`fallback_model`（兜底模型）。
- 回退候选遵守 Token 模型限制、渠道过滤、固定渠道与会话亲和性约束；切换兜底分组后会同步更新实际请求模型和分组。
- 新增 `AttemptModelName`：客户端请求模型与计费模型身份保持稳定，同时允许当前上游尝试使用回退模型。
- 上游返回“模型不存在”时自动重试其他渠道；本地 NewAPI 模型不可用仍保持不可重试，避免误重试。
- 策略配置收拢到系统设置的独立页面“请求策略 → Relay access policies”，接口为 `/api/option/request_policy`。

### 3. 渠道级回退控制

- 新增“仅回退渠道”：该渠道所有模型不参与普通直接选路，但仍可被模型回退与兜底分组选中。
- 新增“仅回退模型”（逗号分隔）：指定模型不参与直接选路，但保留在回退候选中。
- 配置保存在渠道 `settings` JSON 中，已接入表单默认值、新建/编辑序列化、模型选择、选择缓存与固定渠道校验。

### 4. 访客账户与权限

- 新增 `IsGuest` 访客角色，管理员可设置访客注册默认值。
- 访客的 API 凭证与业务访问受到限制，仅开放公益相关路由。
- 用户列表、个人资料、安全页、侧边栏与登录/注册流程均已适配访客身份。

### 5. 贡献者渠道管理

- 新增“贡献”板块：贡献者概览（渠道数、启用渠道数、请求数、Token 用量、模型排行）与贡献者渠道管理页。
- 访客 / 贡献者可创建、编辑、禁用、删除**自己**的渠道，并可“从上游获取模型”。
- 严格的渠道所有权隔离：贡献者只能查看和操作自己的渠道。
- 上游模型发现错误会做脱敏，避免泄露渠道密钥。
- 管理员可查看全部贡献者渠道及其贡献者信息。
- 新增贡献相关路由、访问令牌路由声明与侧边栏入口。

### 6. 请求捕获与用量日志增强

- 支持按配置捕获原始请求、上游请求与上游响应，限制单次体积并按保留期清理。
- 管理员可查询、查看详情、导出与清理捕获记录；用量日志详情中展示每次尝试。
- 日志导出支持管理员 NDJSON / CSV，并新增可读的 TXT 导出。
- 日志详情显示请求模型、重试链中的渠道名称，并在导出时附带请求捕获。
- 日志记录客户端 IP，新增管理员选项 `log_setting.force_record_ip` 强制记录并锁定用户隐私开关。
- 支持在日志详情中一键封禁 IP（封禁调用者当前 IP 需二次确认），新增 `log:write` 个人访问令牌权限。

### 7. 入站 IP 黑名单

- 黑名单保存在主库 `options` 表（`ip_blacklist_setting.enabled` / `ip_blacklist_setting.ip_list`）。
- 支持单个 IP 与 CIDR 网段，在认证前直接返回 HTTP 403。
- 提供管理界面、输入校验与回归测试。

### 8. 构建与发布

- 分支镜像发布从 Docker Hub 改为发布到 GitHub Container Registry（GHCR）。

## 上游同步

- 上游仓库：[QuantumNous/new-api](https://github.com/QuantumNous/new-api)
- 本仓库 `main` 会定期合并 `upstream/main`，自定义功能通过普通提交与合并提交保留，冲突处理遵循上游结构。
- 本项目保留 new-api / QuantumNous 的名称、品牌与归属信息。

## 许可与归属

本项目沿用上游的 [GNU Affero General Public License v3.0 (AGPLv3)](./LICENSE)，并基于 [One API](https://github.com/songquanpeng/one-api)（MIT License）发展而来。修改版本需保留原作者归属声明，以及指向原项目 <https://github.com/QuantumNous/new-api> 的可见链接。第三方依赖与归属见 [NOTICE](./NOTICE) 与 [THIRD-PARTY-LICENSES.md](./THIRD-PARTY-LICENSES.md)。

Frontend design and development by New API contributors.
