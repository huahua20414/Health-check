# Health Checkup - AI Assistant Guide

> 本文档只保留高优先级规则和加载入口。详细规则按主题拆到 `docs/guides/`；遇到对应主题必须先加载对应 guide。

## 快速入口

- 项目概览：`docs/guides/project-overview.md`
- 开发工作流与提交规则：`docs/guides/workflow.md`
- 代码与数据约定：`docs/guides/conventions.md`
- 认证与权限：`docs/guides/auth-and-permissions.md`
- 预约、号源与候补：`docs/guides/appointment-rules.md`
- 测试与交付检查：`docs/guides/testing.md`
- 约束文档维护：`docs/guides/constraint-doc-maintenance.md`

## 项目速记

- 项目：东软熙心健康体检管理系统
- 后端：Go、Gin、GORM、MySQL、Redis、JWT
- 前端：React、Vite、自定义 CSS、React Router
- 本地运行：Docker Compose、Makefile
- 后端入口：`backend/cmd/server`
- 前端入口：`frontend/src/main.jsx`
- React 业务入口：`frontend/src/react/`

## 最高优先级规则

- 系统不是纯静态演示页。套餐、预约、报告、用户、医生、号源等业务数据必须来自后端接口和数据库，禁止写死业务记录或假流程。
- 登录和注册使用邮箱验证码，不再采集或校验密码。
- 前端隐藏菜单只是体验优化，权限必须由后端接口校验。
- 用户只能查看自己的预约和报告；医生只能处理分配给自己的预约和报告；管理员才能审核医生和管理基础配置。
- 预约必须由后端按医生半小时号源自动分配；号源满时进入候补。
- 邮件发送失败不阻断业务，但必须记录邮件日志。
- SMTP、数据库、Redis 等环境配置只能来自 `.env` 或部署环境变量，不能写入源码。
- `.env` 禁止提交；仓库只提交 `.env.example`。

## 提交与完成态规则

- 修改前先读相关 guide，不要凭历史记忆改代码。
- 涉及后端业务规则、接口、权限、预约、数据结构时，必须补充或更新后端测试。
- 涉及前端页面和交互时，至少执行 `cd frontend && npm run build`。
- 确认代码、测试、构建和用户要求都没有问题后，可以提交；用户已明确要求 push 时，可以继续推送。
- 不要回滚用户或其他协作者的未提交改动。

## 本地命令

```bash
make up
make seed
make logs
make down
```

访问地址：

- 前端：http://localhost:5174
- 后端健康检查：http://localhost:8081/api/health

种子账号：

- 管理员：`huahua20414@foxmail.com`，登录时使用邮箱验证码。

## 必跑检查

较大修改后至少执行：

```bash
cd frontend && npm run build
cd backend && go test ./...
```

部署或种子数据相关修改还需要验证：

```bash
docker compose up -d --build
make seed
```

需要人工或自动化确认：

- 未登录访问受保护接口返回 `401`。
- 用户访问医生或管理员接口返回 `403`。
- 医生访问自己的预约和报告接口正常。
- 管理员可以审核医生、管理套餐、机构、号源和项目组合。
- 前端用户、医生、管理员看到的菜单不同。
