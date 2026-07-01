# 项目概览

## 系统定位

本系统是健康体检管理系统，覆盖核心闭环：

- 用户注册、登录、查看体检套餐、提交体检预约。
- 后端按医生半小时号源自动分配预约；号源满时进入候补。
- 医生查看预约、确认体检状态、生成或更新体检报告。
- 用户查看自己的预约、候补和体检报告。
- 管理员审核医生账号，管理用户、机构、套餐、体检项目、套餐项目组合和医生号源。
- 预约成功、候补递补、报告生成需要发送邮件通知；邮件失败不阻断业务，但必须记录邮件日志。

## 技术栈

- 后端：Go、Gin、GORM、MySQL、Redis、JWT。
- 前端：React、Vite、React Router、lucide-react、自定义 CSS。
- 本地运行：Docker Compose、Makefile。

## 目录说明

后端：

- `backend/cmd/server`：服务启动入口。
- `backend/internal/auth`：JWT 签发与解析、Redis session key。
- `backend/internal/cache`：Redis 连接。
- `backend/internal/config`：环境变量配置。
- `backend/internal/database`：MySQL 连接和 GORM AutoMigrate。
- `backend/internal/handlers`：HTTP 路由和业务处理。
- `backend/internal/middleware`：限流等中间件。
- `backend/internal/models`：数据库模型。
- `backend/internal/seed`：可重复执行的模拟数据。

前端：

- `frontend/src/main.jsx`：前端入口。
- `frontend/src/react/App.jsx`：React 路由入口。
- `frontend/src/react/HealthContext.jsx`：共享业务状态、接口动作和表单状态。
- `frontend/src/react/components`：通用布局与 UI 组件。
- `frontend/src/react/views`：各角色业务页面。
- `frontend/src/api`：请求客户端和 token 注入。

## 模块归属

用户端：

- 健康服务：体检套餐、预约体检。
- 我的体检：我的预约、候补状态、我的报告。
- 个人中心：个人资料、家庭成员、通知。

医生端：

- 体检业务：预约处理、报告录入。
- 档案查询：客户档案。

管理员端：

- 用户与权限：用户管理、医生审核。
- 体检服务管理：套餐管理、机构管理、体检项目、套餐项目组合、医生号源。
