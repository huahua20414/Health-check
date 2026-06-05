# 东软熙心健康体检管理系统

作业版健康体检管理系统，覆盖用户预约体检、医生生成报告、用户查看报告的核心闭环。

## 技术栈

- 后端：Go + Gin + GORM
- 前端：Vue3 + Vite + ElementPlus
- 数据库：MySQL + Redis
- 通知：SMTP 邮件
- 本地环境：Docker Compose + Makefile

## 快速启动

```bash
make up
make seed
```

访问前端：http://localhost:5173

后端健康检查：http://localhost:8080/api/health

## 常用命令

```bash
make up       # 构建并启动 MySQL、后端、前端
make seed     # 插入模拟数据
make logs     # 查看服务日志
make down     # 停止服务
make clean    # 停止并删除数据库卷
```

## 演示账号

- 用户：`13800000001`
- 医生：`13900000001`
- 管理员：`13700000001`

默认密码：

- 用户/医生：`123456`
- 管理员：`admin123`

## 认证与权限

- 登录后后端签发 JWT，前端通过 `Authorization: Bearer <token>` 调用接口。
- Redis 保存服务端 session，退出登录会删除 session。
- 用户只能查看和维护自己的预约、报告。
- 医生可以处理预约并生成报告。
- 管理员可以审核医生账号和管理用户状态。

## 邮件配置

复制 `.env.example` 为 `.env`，填入 SMTP 配置后重启服务：

```bash
cp .env.example .env
make up
```

`.env` 已被 git 忽略，不能提交真实邮箱授权码。

## 预约与候补

- 管理员/种子数据维护医生半小时号源。
- 用户预约时选择套餐、日期、时段，系统自动分配医生和具体半小时时间。
- 同一医生同一半小时只能预约 1 人。
- 号源满时进入候补；有人取消后系统自动递补最早候补用户。
- 预约成功、候补递补、报告生成会发送邮件通知并记录邮件日志。
