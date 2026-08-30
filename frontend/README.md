# AI Meeting 前端

AI Meeting 的 React 19 + TypeScript 6 前端，提供登录、简历上传、面试对话、实时语音转写、题目语音播报、PDF 预览和面试报告等功能。

## 本地开发

后端默认运行在 `http://localhost:8080`，Vite 会将 `/api` 请求和 WebSocket 连接代理到后端。

```bash
npm install
npm run dev
```

## 质量检查

```bash
npm run test:run
npm run lint
npm run build
```

生产构建输出到 `dist/`。Docker 镜像使用 Nginx 托管静态资源，并将 `/api/` 代理到 `api:8080`。
