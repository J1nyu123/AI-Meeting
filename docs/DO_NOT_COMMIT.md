# 禁止提交到版本库的文件清单

本清单用于提交前检查。`.gitignore` 负责阻止常见文件被误加入，但不能替代人工检查；已经被 Git 跟踪的文件也不会因为后来加入忽略规则而自动移除。

## 1. 密钥与私密配置

禁止提交：

- 根目录和前端目录中的 `.env`、`.env.*`，仅 `.env.example` 可以提交。
- `LLM_API_KEY`、三组讯飞星辰 API Key、API Secret、Flow ID 的真实值。
- `JWT_SECRET`、数据库生产密码、Redis 密码和部署平台 Token。
- 私钥、证书及密钥容器，例如 `*.pem`、`*.key`、`*.p12`、`*.pfx`。
- 包含真实密钥的终端输出、截图、日志、SQL、工作流导出文件或临时备份。

示例配置只能保留明显的占位符，不能复制本机 `.env` 的真实值。

## 2. 用户和运行数据

禁止提交：

- `data/`、`uploads/` 及任何本地文件存储目录。
- 用户简历 PDF、语音录音、转写原文、面试答案和报告导出文件。
- MySQL 导出、Redis 快照、Docker volume 备份及数据库二进制文件。
- 为人工测试创建的真实账号资料或可识别个人身份的信息。

自动测试夹具必须使用虚构数据，不应由真实简历直接脱敏后提交。

## 3. 依赖、构建产物与缓存

禁止提交：

- `frontend/node_modules/`。
- `frontend/dist/`、`frontend/dist-ssr/`、覆盖率目录和构建产物。
- 根目录 `.cache/`、`bin/`、Go 测试二进制、可执行文件和覆盖率报告。
- 日志、临时文件、npm/yarn/pnpm 调试日志。
- Docker 本地数据卷内容。

`go.sum` 和 `frontend/package-lock.json` 是依赖锁定文件，应当提交，不属于缓存。

## 4. 编辑器与操作系统文件

禁止提交：

- `.idea/`、`.vscode/` 中仅适用于个人环境的配置。
- Visual Studio 用户文件，例如 `*.suo`、`*.user`、`*.userosscache`。
- `.DS_Store`、`Thumbs.db`、`Desktop.ini`。
- 本机绝对路径、个人用户名或仅在个人电脑可用的启动配置。

如果未来需要共享编辑器配置，应逐项确认其中没有个人路径、插件账号或密钥后，再为该文件设置明确的忽略例外。

## 5. 可以提交的典型文件

- Go、TypeScript、CSS 和测试源码。
- `migrations/` 中的版本化迁移。
- `.env.example` 中不含密钥的配置模板。
- `go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json`。
- Dockerfile、`docker-compose.yml`、验证脚本和事实准确的项目文档。
- 已确认授权可以分发的图片、字体、视频和许可证文件。

## 6. 提交前检查（PowerShell）

查看将要提交的文件：

```powershell
git status --short
git diff --cached --name-only
```

确认常见敏感目录会被忽略：

```powershell
git check-ignore -v .env frontend/node_modules frontend/dist .cache data
```

然后人工检查暂存区：

1. 文件名中是否出现 `.env`、`resume`、`upload`、`dump`、`backup`、`log` 或证书后缀。
2. 配置和文档中是否含真实 API Key、Secret、JWT、密码、内网地址或个人绝对路径。
3. 图片、视频、PDF 和测试数据是否确实需要公开，且拥有分发授权。
4. 新增大文件是否是源码必需素材，而不是构建产物或本地测试文件。

若敏感信息曾经提交过，仅从当前文件删除并不等于从 Git 历史中清除；应立即轮换对应密钥，并在发布前单独处理历史记录。
