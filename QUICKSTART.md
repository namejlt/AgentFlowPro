# AgentFlow Pro 快速部署指南

本文档帮助你从零开始部署 AgentFlow Pro，支持 **Docker 一键部署** 和 **手动部署** 两种方式。

---

## 一、Docker Compose 一键部署（推荐）

### 1.1 环境要求

| 依赖 | 最低版本 |
|------|----------|
| Docker | 20.10+ |
| Docker Compose | v2.0+（含 `docker compose` 命令） |
| 内存 | ≥ 4GB |
| 磁盘 | ≥ 10GB |

### 1.2 部署步骤

```bash
# 1. 克隆仓库
git clone https://github.com/namejlt/AgentFlowPro.git
cd AgentFlowPro

# 2. 创建环境变量文件
cp .env.example .env

# 3. 修改安全配置（重要！）
# 编辑 .env，至少修改以下项：
#   JWT_SECRET       - 替换为 64 字符随机字符串
#   ENCRYPTION_KEY   - 替换为 32 字节随机字符串
#   POSTGRES_PASSWORD - 替换为强密码
vim .env

# 4. 一键启动
docker compose up -d

# 5. 查看服务状态
docker compose ps

# 6. 查看后端日志
docker compose logs -f backend
```

### 1.3 访问服务

| 服务 | 地址 |
|------|------|
| 前端界面 | http://localhost:28232 |
| 后端 API | http://localhost:28131 |
| 健康检查 | http://localhost:28131/healthz |

### 1.4 默认账号

首次启动时后端会自动执行数据库迁移和种子数据。默认管理员账号：

| 字段 | 值 |
|------|------|
| 邮箱 | admin@agentflow.local |
| 密码 | admin123 |

> ⚠️ **生产环境请务必修改默认密码！**

### 1.5 常用运维命令

```bash
# 停止所有服务
docker compose down

# 停止并清除数据卷（重置数据库）
docker compose down -v

# 重新构建并启动
docker compose up -d --build

# 查看日志
docker compose logs -f backend
docker compose logs -f frontend

# 进入后端容器
docker compose exec backend sh

# 进入数据库
docker compose exec postgres psql -U agentflow -d agentflow
```

### 1.6 端口配置

默认端口可在 `.env` 中修改：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| FRONTEND_PORT | 28232 | 前端对外端口 |
| BACKEND_PORT | 28131 | 后端对外端口 |
| POSTGRES_PORT | 5432 | PostgreSQL 端口 |

---

## 二、手动部署

### 2.1 环境要求

| 依赖 | 版本 |
|------|------|
| Go | 1.23+ |
| Node.js | 20+ |
| PostgreSQL | 14+ |

### 2.2 数据库准备

```bash
# 创建数据库和用户
psql -U postgres
CREATE USER agentflow WITH PASSWORD 'your_password';
CREATE DATABASE agentflow OWNER agentflow;
\q
```

### 2.3 后端启动

```bash
cd AgentFlowPro

# 设置环境变量
export DATABASE_URL="host=localhost user=agentflow password=your_password dbname=agentflow port=5432 sslmode=disable"
export JWT_SECRET="your-64-char-random-secret-string-here-replace-in-production"
export ENCRYPTION_KEY="0123456789abcdef0123456789abcdef"
export GIN_MODE=release

# 编译运行
go mod tidy
go build -o bin/server ./cmd/server
./bin/server
```

### 2.4 前端构建

```bash
cd web

# 安装依赖
npm ci

# 构建生产包
npm run build

# 使用 Nginx 托管
# 将 dist/ 目录复制到 Nginx 的 html 目录
# 参考 docker/nginx/default.conf 配置反向代理
```

---

## 三、生产环境安全清单

### 3.1 必须修改

- [ ] `.env` 中 `JWT_SECRET` 替换为 64 字符随机字符串
- [ ] `.env` 中 `ENCRYPTION_KEY` 替换为 32 字节随机字符串
- [ ] `.env` 中 `POSTGRES_PASSWORD` 替换为强密码
- [ ] 修改默认管理员密码
- [ ] 关闭 PostgreSQL 对外端口（移除 `POSTGRES_PORT` 映射）

### 3.2 建议修改

- [ ] 配置 HTTPS（使用 Nginx + Let's Encrypt 或反向代理）
- [ ] 配置防火墙规则，仅开放前端端口
- [ ] 定期备份数据库
- [ ] 配置日志收集与监控

### 3.3 生成安全密钥

```bash
# 生成 JWT_SECRET (64字符)
openssl rand -hex 32

# 生成 ENCRYPTION_KEY (32字节)
openssl rand -hex 16
```

---

## 四、HTTPS 配置（可选）

### 4.1 使用 Nginx 反向代理

在宿主机安装 Nginx，配置 SSL 证书后反向代理到 Docker 服务：

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate     /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:28232;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 4.2 关闭 Docker 直接端口

修改 `.env`，移除后端和数据库的直接端口映射：

```env
# 注释掉或删除
# BACKEND_PORT=28131
# POSTGRES_PORT=5432
```

然后修改 `docker-compose.yml` 中对应服务的 `ports` 配置，仅保留前端端口。

---

## 五、数据备份与恢复

### 5.1 备份数据库

```bash
docker compose exec postgres pg_dump -U agentflow agentflow > backup_$(date +%Y%m%d).sql
```

### 5.2 恢复数据库

```bash
cat backup_20240101.sql | docker compose exec -T postgres psql -U agentflow agentflow
```

### 5.3 备份上传文件

```bash
docker compose run --rm -v $(pwd)/backup:/backup backend cp -r /app/uploads /backup/uploads
```

---

## 六、升级更新

```bash
# 拉取最新代码
git pull origin main

# 重新构建并启动
docker compose up -d --build

# 查看迁移日志
docker compose logs backend | grep migrate
```

---

## 七、故障排查

### 后端无法连接数据库

```bash
# 检查 PostgreSQL 是否就绪
docker compose exec postgres pg_isready -U agentflow

# 检查网络连通
docker compose exec backend ping postgres
```

### 前端页面空白

```bash
# 检查 Nginx 配置
docker compose exec frontend nginx -t

# 检查后端 API 是否可达
curl http://localhost:28131/healthz
```

### SSE 连接断开

- 检查 Nginx 的 `proxy_read_timeout` 配置（已设为 86400s）
- 检查是否有中间代理/CDN 限制了长连接

---

## 八、接口说明

### 统一返回格式

```json
{
  "request_id": "uuid",
  "code": 0,
  "message": "success",
  "data": {},
  "meta": { "page": 1, "page_size": 20, "total": 57 }
}
```

### 认证

除 `/api/v1/auth/login` 和 `/healthz` 外，所有接口需携带：

```
Authorization: Bearer <JWT_TOKEN>
```

### SSE 事件流

```
GET /api/v1/tasks/:id/stream
Content-Type: text/event-stream
```

事件类型：

| event | 说明 |
|-------|------|
| `node_status` | 节点状态变更 |
| `agent_stream_start` | 智能体开始输出 |
| `agent_stream_chunk` | 流式输出片段 |
| `agent_stream_end` | 智能体输出完成 |
| `debate_round` | 辩论轮次完成 |
| `task_complete` | 任务完成 |
| `task_failed` | 任务失败 |

### 业务错误码

| code | HTTP | 含义 |
|------|------|------|
| 0 | 200 | 成功 |
| 1001 | 400 | 参数无效 |
| 1002 | 401 | 未认证 |
| 1003 | 403 | 无权限 |
| 1004 | 404 | 资源不存在 |
| 1005 | 409 | 资源冲突 |
| 2001 | 408 | 上游超时 |
| 2002 | 429 | 限流 |
| 3001 | 500 | 内部错误 |
