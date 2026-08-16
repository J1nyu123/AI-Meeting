# 架构决策

## ADR-001：MySQL 权威、Redis 可丢弃

所有可恢复业务结果先写 MySQL。Redis 仅保存 24 小时热状态、分布式锁和任务事件；Redis 失效时从会话、题目和成功轮次重建。

## ADR：历史面试删除仅允许终态会话

删除接口只允许当前用户删除 `COMPLETED` 或 `FAILED` 会话，并在事务内重新校验归属与状态。数据库提交后再清理本地简历和 Redis 热状态；外部清理失败只记录告警，不撤销已经提交的数据库事务。分析中或进行中的会话不提供强制取消，避免任务与状态竞争。

## ADR-002：双 Token 与轮换

Access Token 短期有效；Refresh Token 使用随机 JTI 和 family，服务端只保存哈希。刷新在数据库事务和行锁内撤销旧 Token、签发同 family 新 Token；复用已撤销 Token 时提交整个 family 的撤销后再返回鉴权失败。

## ADR-003：AI 调用在事务外

先建立带租约的幂等尝试，再调用 AI，最后使用短事务和会话版本号提交。Redis 不可用时可能产生重复模型调用，但数据库唯一约束保证不重复推进和计分。

## ADR-004：讯飞优先、浏览器语音降级

页面依赖 `TranscriptionProvider`。默认使用 Go 鉴权 WebSocket 代理讯飞 AST，失败时降级 Web Speech API，文字输入永久可用。浏览器先用 Access Token 申请 60 秒一次性 Ticket，WebSocket URL 不携带 JWT；Ticket 和单用户连接租约依赖 Redis并在 Redis 故障时失败关闭。

## ADR-005：异步简历分析

API 保存文件与任务记录后投递 Asynq。Worker 幂等处理，状态落 MySQL；SSE 每次连接先返回数据库快照，再订阅 Redis 通知，并每秒补查 MySQL，Redis 事件丢失或断线时仍可得到最终状态。

## ADR-006：版本化迁移先于服务启动

使用 `golang-migrate` 和独立 `cmd/migrate`。Compose 中的一次性迁移服务成功后 API 与 Worker 才启动；不依赖 MySQL 仅首次执行的初始化目录，因此后续可以追加迁移版本。

## ADR-007：题目播报使用鉴权长文本 TTS

题目和追问优先创建讯飞长文本 TTS 任务，同一用户和 `Idempotency-Key` 只创建一次。前端只取得本站 `/audio` 路径；Go 校验用户归属后重新查询上游 URL，仅对精确可信的讯飞下载主机执行 HTTP 到 HTTPS 升级，过滤 DNS 结果中的私有地址、固定连接剩余安全地址并拒绝重定向，再流式代理音频。远程失败时降级浏览器 `speechSynthesis`，用户主动取消不触发降级。

## ADR-010：媒体内容最小化保存

不保存录音、TTS 原文、上游音频地址或音频二进制。MySQL 仅保存 TTS 任务归属、幂等键、状态和格式；日志不记录转写正文、合成正文、签名和下载地址。

## ADR-008：讯飞面试 Agent 采用三个顶层工作流调用

面试模型能力包括出题、简历评分、答案评分和提问四部分；服务直接调用出题、答案评分和提问三个工作流，简历评分由讯飞出题工作流内部子流程完成。三个工作流使用独立凭据，通过 `LLM_MODE=xingchen` 启用；表情分析不进入当前无摄像头范围。出题或评分配置缺失时对应能力回退 Mock，提问配置缺失或调用失败时优先使用评分官建议，避免中断活动面试。

## ADR-009：PDF 文本提取使用 Poppler 并执行质量门禁

`PDFExtractor` 使用容器内 Poppler `pdftotext`。此前使用的 PDF 库无法正确解析部分中文字体的 ToUnicode/CMap，会将正常简历提取为控制字符和逐字空格。提取结果不足 50 字、不是有效 UTF-8、包含替换字符或非法控制字符时统一返回 `PDF_OCR_REQUIRED`，禁止损坏文本进入评分 Agent。扫描版 OCR 仍不在首版范围内。
