# 实时 ASR WebSocket 协议

1. 使用 Access Token 调用 `POST /api/v1/media/asr/tickets`。
2. 使用返回的一次性 Ticket 连接 `GET /api/v1/media/asr/ws?ticket=...`。
3. 收到 `connected` 后发送 `{"type":"start_transcription"}`。
4. 发送 PCM S16LE、16 kHz、单声道二进制帧；推荐每帧 1280 字节。
5. 服务端返回 `transcription` 原子快照；`revision` 严格递增，客户端忽略旧版本。
6. 停止时发送 `{"type":"stop_transcription"}`，等待 `final` 和 `transcription_stopped`。

控制消息：`ping`、`get_status`、`start_transcription`、`stop_transcription`。

关键关闭/错误：无效 Ticket 为 HTTP 401；已有连接为 HTTP 409；音频过载使用 WebSocket 1013；上游失败、租约丢失和空闲超时通过 `error.code` 返回。Ticket 不可重用，重连必须重新申请。
