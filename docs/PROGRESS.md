# 实施进度

更新时间：2026-08-15

计算：`总进度 = Σ(模块权重 × 已完成验收项/验收项总数)`。结构、主路径、异常测试、验收证据分别对应 25/50/75/100%。

| 编号 | 模块 | 权重 | 完成项/总项 | 模块进度 | 加权进度 | 状态 |
|---|---|---:|---:|---:|---:|---|
| P0 | 契约与追溯文档 | 6 | 6/6 | 100 | 6.0 | 完成 |
| P1 | Go 基础工程 | 8 | 6/6 | 100 | 8.0 | 完成，真实 Docker 启动、迁移和健康检查通过 |
| P2 | 双 Token 认证 | 10 | 6/6 | 100 | 10.0 | 完成，含 HTTP 越权、轮换和 family 撤销测试 |
| P3 | 简历与异步出题 | 16 | 7/7 | 100 | 16.0 | 完成，真实 PDF、Redis/Asynq、Worker 和报告闭环通过 |
| P4 | 答题与追问核心 | 22 | 8/8 | 100 | 22.0 | 完成，含幂等和20并发测试 |
| P5 | 长会话恢复与降级 | 15 | 5/6 | 83 | 12.5 | MySQL 降级测试完成；待真实 Redis 故障注入 |
| P6 | 报告模块 | 6 | 4/4 | 100 | 6.0 | 完成，含计分和回放测试 |
| P7 | 前端与浏览器语音 | 11 | 7/7 | 100 | 11.0 | 完成，浏览器语音、去重、恢复和文字降级人工验收通过 |
| P8 | 测试与交付 | 6 | 6/6 | 100 | 6.0 | 单测、HTTP 集成、并发/故障测试、指标、README 与演示脚本齐全 |

当前实施总进度：**98/100**（精确加权值 97.5）。最终验收仍受 Redis 故障注入和 80% 覆盖率门槛约束。

## 更新记录

### 2026-08-15：建立实施依据

- 完成：REFERENCE_MAP、ADR、进度表、ER 图、状态机；OpenAPI 初版随骨架提交。
- 依据：见 `REFERENCE_MAP.md`。
- 测试：文档路径在实施前已逐一使用 `Test-Path` 核验。
- 遗留：代码和运行验证尚未开始。
- 下一项：基础工程、迁移、统一响应和健康检查。

### 2026-08-15：核心实现与本地验证

- 完成：Go API/Worker、迁移、双 Token、异步简历任务、SSE、模型适配、答题状态机、追问、幂等、MySQL 降级恢复、报告、Prometheus 指标、前端新 API 与 Browser Speech Provider。
- Go 证据：`go build ./...`、`go test ./...`、`go vet ./...` 全部通过；默认缓存无权限，验证时使用项目 `.cache`。
- 前端证据：`npm run lint`、`npm run typecheck`、`npm run test:run`（21 文件、94 测试）、`npm run build` 全部通过。
- 并发证据：`TestConcurrentAnswersOnlyAdvanceOnceInMySQLFallback` 使用20个并发请求，仅一个成功推进。
- 降级证据：`TestStateRehydratesFromMySQLWithoutRedis`、`TestAnswerIsIdempotentAndDegradesWithoutRedis` 通过。
- Docker：`docker compose config --quiet` 通过；本机 Docker daemon 未运行，未虚报容器 E2E。
- 覆盖率现状：auth 38.0%、evaluation 77.8%、interview 54.8%、llm 12.3%、report 47.8%，尚未达到最终80%目标。
- 下一项：启动 Docker 环境执行 MySQL/Redis/SSE E2E、Redis 故障注入、浏览器人工语音验收并补覆盖率。

### 2026-08-15：异常路径、迁移和事件恢复加固

- 本次完成：认证 HTTP 越权与完整 Token 流程；简历类型/签名/大小校验；Worker 成功、OCR 失败、LLM 失败后重试恢复；SSE 重连后的 MySQL 最终状态补查；追问两次上限、追问不重复计分、处理租约接管。
- 修复：Refresh Token 轮换改为事务和行锁；任务重试从 `FAILED` 恢复为 `ANALYZING`；AI 失败码不再被延迟清理覆盖；SSE 增加 Redis Pub/Sub 通知并保留 MySQL 轮询降级；前端开发代理由原 Java `8002` 切换到 Go API `8080`。
- 交付：新增 `cmd/migrate` 和 Compose 一次性迁移服务；容器补 CA 证书和上传目录权限；新增 `scripts/demo.ps1` 完整后端演示。
- 原文件依据：认证、异步简历、答题幂等、运行时恢复与 SSE 行，详见 `REFERENCE_MAP.md`。
- 新增/修改测试：`internal/auth/http_test.go`、`service_test.go`；`internal/job/http_test.go`、`service_test.go`；`internal/interview/service_test.go`；`internal/llm/client_test.go`；`internal/resume/resume_test.go`。
- 测试结果：`go build ./...`、`go test ./...`、`go vet ./...`、`docker compose config --quiet` 通过；前端 lint、typecheck、96 项测试和生产构建通过。`scripts/verify.ps1` 已增加每一步非零退出码检查，使用项目内临时 Docker 配置避免用户目录权限警告；`verify.ps1` 与 `demo.ps1` 均通过 PowerShell 7 和 Windows PowerShell 5.1 解析验证，演示上传不依赖 5.1 缺失的 `-Form` 参数。
- 覆盖率：internal 总计 62.9%；auth 83.4%、evaluation 100%、interview 69.1%、job 64.5%、llm 77.2%、report 50.7%、resume 42.9%。尚未满足最终“核心领域不低于 80%”门槛。
- 模块变化：P2 83%→100%，P3 71%→86%，P8 67%→100%；总进度 86→92.3。
- 遗留风险：本机 Docker daemon 未运行，因此未执行真实 MySQL/Redis/Asynq E2E；浏览器语音自动重启仍需 Chrome/Edge 人工验收；整体覆盖率仍需提高。
- 下一验收项：启动 Docker 后运行 `scripts/demo.ps1`，执行 Redis 运行时断连/恢复测试，再做浏览器语音人工验收。

### 2026-08-15：Docker 首次启动验收

- 现象：MySQL 首次初始化耗时约 162 秒，超过原健康检查约 100 秒容忍范围，Compose 提前报告 unhealthy。
- 修复：MySQL healthcheck 增加 180 秒 `start_period`，重试次数调整为 30；API 容器使用 Gin release 模式。
- 结果：MySQL/Redis healthy，`migrate` 退出码 0 且输出 `migrations_applied`，API/Worker 正常运行，`/health/live` 与 `/health/ready` 均返回 `UP`。
- 模块变化：P1 83%→100%；总进度 92.3→93.6。
- 下一验收项：使用文本型 PDF 执行 `scripts/demo.ps1`，验证真实 Asynq、SSE、答题和报告闭环。

### 2026-08-15：真实后端业务闭环验收

- 本次完成：使用真实文本型 PDF 执行注册、登录、创建会话、上传、Redis/Asynq 入队、Worker 分析出题、答题追问、自动完成和报告生成。
- 运行证据：会话 `6e928c3f-d170-431a-9401-1903d429574f`；任务从 `queued` 到 `completed`；完成 5 道主问题和每题最多 2 次追问；最终综合分 75。
- 对应依据：`InterviewQuestionExtractionService.java`、`InterviewAnswerPipeline.java`、`InterviewFollowUpRuleService.java`、`InterviewRecordServiceImpl.java`，完整路径见 `REFERENCE_MAP.md`。
- 测试结果：`scripts/demo.ps1` 在真实 MySQL、Redis、API、Worker 容器上完整执行成功。
- 模块变化：P3 86%→100%；总进度 93.6→95.9。
- 遗留风险：当前使用 mock LLM；浏览器 Speech API、刷新恢复、Redis 运行中断和覆盖率门槛仍待验收。
- 下一验收项：启动 React 前端，在 Chrome/Edge 验证语音输入、去重、文字降级和页面刷新恢复。

### 2026-08-15：题目播放无声修复

- 原因：播放链路当时仍调用未接入的 `/xunzhi/v1/xunfei/tts/synthesize`；与 LLM API 是否配置无关。
- 修复：新增 `TtsProvider` 和 `BrowserSpeechSynthesisProvider`，追问/主问题优先使用浏览器中文语音合成，保留停止、取消和远程链路扩展边界。
- 新增文件：`src/features/tts/types.ts`、`BrowserSpeechSynthesisProvider.ts` 及其测试；修改 `useChatTtsPlayback.ts`。
- 测试结果：前端 typecheck、lint、22 个测试文件共 96 项测试、生产构建全部通过。
- 人工验收：用户已在实际浏览器和主机扬声器环境确认追问播放有声音。
- 遗留风险：题目播报已通过；浏览器语音输入、拒绝麦克风后的文字降级和连续重启仍待人工验收。
- 下一验收项：使用麦克风验证浏览器语音输入、最终文本去重和人工修改后提交。

### 2026-08-15：浏览器语音人工验收

- 本次完成：浏览器语音输入、partial/final 文本治理、重复文本去重、人工修改后提交、页面刷新恢复，以及麦克风不可用时的文字输入降级。
- 自动化证据：前端 22 个测试文件、96 项测试通过；`BrowserSpeechProvider`、去重函数和音频控制器均有对应测试。
- 人工证据：用户已在实际浏览器完成语音和降级流程测试；题目/追问 Browser TTS 同样确认有声音。
- 模块变化：P7 86%→100%；总进度 95.9→97.5。
- 遗留风险：Redis 运行中断、缓存删除恢复仍待真实环境验收；Go internal 总覆盖率仍未达到最终 80% 门槛。
- 下一验收项：在已生成题目的活动会话中执行 Redis 热状态删除和服务停止测试。

### 2026-08-15：刷新恢复与历史报告跳转修复

- 本次完成：发现并修复 Redis 旧版本热状态覆盖 MySQL、刷新后未重建历史消息，以及进行中会话错误进入报告页的问题。
- 数据核对：问题会话的 7 条成功轮次仍保存在 MySQL；报告 404 来自会话尚未完成，并非已持久化轮次丢失。
- 对应依据：运行时重建、前端恢复和报告行，详见 `REFERENCE_MAP.md`。
- 新增/修改文件：`internal/interview/runtime.go`、`service.go`、`service_test.go`；前端会话消息恢复、侧边栏路由、报告查询及对应测试。
- 定向验证：`go test ./internal/interview -run TestState -count=1` 通过；会话恢复 11 项测试通过；侧边栏与报告 12 项测试通过。

### 2026-08-15：前端精简与历史面试删除

- 本次完成：隐藏未接入 Go 后端的聊天/搜索入口，账号信息提供只读反馈，面试室使用通用标题；新增终态历史面试删除、确认、错误提示和查询刷新；登录视频与首页视频拆分，品牌标志更新。
- 对应依据：会话归属和应用编排参考 `REFERENCE_MAP.md`；删除、双视频和账号弹窗均标记为 Go/前端新增，不描述成原版已有能力。
- 新增/修改文件：`internal/interview`、`internal/resume`、`internal/job` 的删除实现与测试；前端侧栏、账号弹窗、登录页、品牌 SVG 和对应测试。
- 定向验证：Go 删除相关包测试通过；前端 4 个测试文件共 9 项通过；完整回归结果见本条后续验收记录。
- 完整回归：`go test ./...` 全部通过；`scripts/verify.ps1` 的 build、test、vet、Compose 配置与 71 条追溯路径全部通过；前端 `npm run check` 的 27 个测试文件、108 项测试全部通过；`npm run build` 成功。
- 模块进度：已有模块百分比不重复增加，总进度保持原表计算值。
- 遗留风险：本地文件或 Redis 清理失败依赖结构化日志告警；浏览器视频自动播放仍受用户浏览器策略控制，已使用静音并保留静态渐变降级。
- 全量验证：`scripts/verify.ps1` 的 Go build/test/vet、Compose 配置及 59 条引用检查通过；前端 lint、typecheck、24 个测试文件共 101 项测试和生产构建通过。
- 模块变化：P5 保持 83%，P7 保持 100%，总进度保持 97.5；未将本次缺陷修复重复计入进度。
- 遗留风险：尚未在真实 Redis 进程断连期间执行故障注入；生产构建提示 Browserslist 数据约 6 个月未更新，不影响本次构建。
- 下一验收项：在浏览器刷新一个包含追问的活动会话，人工确认视觉顺序和继续答题行为。

### 2026-08-15：接入讯飞星辰面试 Agent

- 本次完成：新增讯飞星辰工作流 Provider，使用三套独立凭据调用出题（内部含简历评分）、答案评分和提问工作流；保留 Mock/OpenAI，并按能力降级。
- 对应依据：`XingChenAIClient.java`、`InterviewQuestionExtractionService.java`、`InterviewEvaluationService.java`、`InterviewFollowUpService.java` 和三份工作流 YAML，完整路径见 `REFERENCE_MAP.md`。
- 新增/修改文件：`internal/llm/xingchen.go` 及测试；强类型 AI 输入；Worker PDF 上传出题；答题简历上下文、独立提问和失败回退；环境变量、Compose、ADR 和参考映射。
- 自动验证：`go test ./internal/llm ./internal/job ./internal/interview -count=1` 通过；`scripts/verify.ps1` 的 build、全量 test、vet、Compose 配置及 65 条引用检查通过。
- 真实验证：会话 `e81f8b07-ffeb-4cfd-b658-5f8f571f95bd` 完成真实出题，方向为 AI Agent / RAG 工程师、简历分 92；生成 10 道主问题。一次短回答完成真实评分并生成 1 道追问；数据库仅写入 1 个成功轮次，日志没有 Provider/提问降级。
- 根据真实结果调整：兼容控制台文档式 `http(s)://` 和完整工作流 URL；新增 `.dockerignore` 排除 `.cache`，构建上下文由约 680MB 降至约 201KB。
- 模块变化：P3、P4 已是 100%，总进度保持 97.5，不重复计入既有验收项。
- 遗留风险：新评分工作流返回“未提供需要评分的面试题目和答案”，尽管 Go 已按导出 YAML 发送三个标准变量；需在讯飞控制台核对该 Flow ID 当前发布版本的开始节点和变量映射。宿主机 `.env` 的 MySQL/Redis 地址会覆盖 Compose 容器地址，本次以临时环境变量完成验证，未改变公共配置契约。
- 下一验收项：在讯飞控制台对新评分 Flow ID 执行相同参数的工作流调试，确认发布版本后复测答案评分质量。

### 2026-08-15：中文 PDF 提取乱码修复

- 本次完成：确认原 PDF 与上传记录 SHA-256 一致，数据库中的题目和回答正常，而 `resume_assets.extracted_text` 已在提取阶段损坏；将 `PDFExtractor` 内部实现从 `rsc.io/pdf` 替换为 Poppler，并增加 UTF-8、替换字符、控制字符和最小长度质量门禁。
- 对应参考依据：`InterviewQuestionExtractionService.java` 的文件上传与出题边界、`useInterviewResumeAnalysis.ts` 的上传状态消费；Poppler 和质量门禁为当前实现新增，差异见 `REFERENCE_MAP.md` 与 ADR-009。
- 新增/修改文件：`internal/resume/resume.go`、`internal/resume/resume_test.go`、`Dockerfile`、`go.mod`、`go.sum` 及三份追溯文档。
- 测试结果：修改前真实简历集成测试失败且复现控制字符乱码；修改后相同 SHA-256 文件在 Linux/Poppler 镜像中通过 `TestTextPDFExtractorExternalFixture`；`go test ./internal/resume ./internal/job ./internal/interview -count=1` 全部通过；最终 `scripts/verify.ps1` 的 build、全量 test、vet、Compose 配置和 66 条引用检查全部通过。重建后的 API/Worker 正常运行，Worker 内 Poppler 24.02.0，`/health/ready` 返回 `UP`。
- 模块变化：P3 已为 100%，总进度保持不变，不重复计入既有验收项。
- 遗留风险：已有会话中已保存的乱码 `extracted_text` 不自动回写；必须重新上传简历创建会话后才能使用新提取结果。扫描型 PDF 仍返回 `PDF_OCR_REQUIRED`。
- 下一验收项：重建并重启 API/Worker 后，新建会话上传同一中文简历，确认评分反馈不再声称简历或回答乱码。
