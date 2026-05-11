# AgentFlow Pro

**基于 Go + Vue3 构建的通用多智能体工作流编排与报告生成平台**

一款开源、可视化、可扩展的多智能体协作系统，支持自定义AI角色、可视化编排分析流程、接入多数据源、执行多轮讨论与验证，并自动生成全场景专业分析报告。

## 核心特性

- **智能体管理**：自定义角色、系统提示词、数据源绑定、参数映射、输出格式
- **可视化工作流编排**：拖拽式设计，支持串行、并行、辩论、交叉验证、风险评审、条件分支等10种节点
- **LLM 配置中心**：多模型统一管理，API Key 加密存储，连通性测试
- **实时任务监控**：SSE 流式输出、辩论气泡、节点状态、执行日志
- **多格式报告导出**：Markdown / PDF / DOCX
- **工作流分享与互通**：一键导出、导入资源匹配、分享码克隆
- **RBAC 权限**：普通用户 / 创作者 / 管理员三角色
- **私有化部署**：Docker Compose 一键启动

## 技术架构

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Vite + Element Plus + Vue Flow + Pinia |
| 后端 | Go 1.23 + Gin + GORM |
| 数据库 | PostgreSQL 16 |
| 实时通信 | SSE (Server-Sent Events) |
| 加密 | AES-256-GCM |
| 部署 | Docker Compose + Nginx |

## 快速开始

详见 [QUICKSTART.md](QUICKSTART.md)

```bash
# 一键启动
cp .env.example .env
docker compose up -d
```

访问 http://localhost:28232 即可使用。

## 接口概览

| 模块 | 前缀 | 说明 |
|------|------|------|
| 认证 | `/api/v1/auth/` | 登录/登出/刷新/用户信息 |
| 用户 | `/api/v1/users` | 用户 CRUD（管理员） |
| 数据源 | `/api/v1/datasources` | 数据源 CRUD / 克隆 / 测试 |
| 智能体 | `/api/v1/agents` | 智能体 CRUD / 克隆 / 预览 |
| 模型 | `/api/v1/models` | LLM 模型 CRUD / 测试 / 设默认 |
| 工作流 | `/api/v1/workflows` | 工作流 CRUD / 版本 / 导入导出 / 分享 |
| 任务 | `/api/v1/tasks` | 任务创建 / SSE 流 / 停止 / 重跑 |
| 报告 | `/api/v1/reports` | 报告列表 / 详情 / 导出 / 归档 |
| 系统 | `/api/v1/system/` | 仪表盘 / 配置 / 审计日志 |

完整接口列表见 [docs/任务拆解.md](docs/任务拆解.md) 第 1.3 节。

## 项目结构

```
AgentFlowPro/
├── cmd/server/          # 后端入口
├── internal/            # 后端业务代码
│   ├── api/             # HTTP 路由与中间件
│   ├── app/             # 应用层 Handler
│   ├── auth/            # JWT 认证
│   ├── config/          # 配置加载
│   ├── crypto/          # AES-256-GCM 加解密
│   ├── database/        # 数据库连接与迁移
│   ├── datasource/      # 数据源执行引擎
│   ├── engine/          # 工作流 DAG 调度引擎
│   ├── export/          # 报告导出 (MD/PDF/DOCX)
│   ├── llm/             # LLM 统一调用层
│   ├── model/           # GORM 模型
│   ├── repository/      # 数据访问层
│   └── templatex/       # 模板变量渲染
├── web/                 # Vue3 前端
│   ├── src/
│   │   ├── api/         # 接口层
│   │   ├── views/       # 14 个页面视图
│   │   ├── components/  # 组件（布局/工作流/报告）
│   │   ├── stores/      # Pinia 状态管理
│   │   └── router/      # 路由与守卫
│   └── Dockerfile
├── docker/              # 部署配置
│   └── nginx/           # Nginx 配置
├── config/              # 配置模板
├── docker-compose.yml
├── Dockerfile
├── .env.example
└── QUICKSTART.md
```

## 开源协议

MIT License
