# 数据库总表与数据字典

> 说明：本文档依据 [backend/internal/models/models.go](/Users/huahua/workspace/unknown/Health-checkup/backend/internal/models/models.go) 与 [backend/internal/database/database.go](/Users/huahua/workspace/unknown/Health-checkup/backend/internal/database/database.go) 当前自动迁移定义整理。字段类型按当前 GORM + MySQL 迁移规则描述，未落库字段（`gorm:"-"`）不计入表结构。

## 数据库总表

| 物理表名 | 对应实体（逻辑表） | 核心功能 |
| --- | --- | --- |
| `users` | 用户/医生/管理员 | 存储账号基础资料、角色、状态、医生职业信息 |
| `checkup_institutions` | 体检机构 | 存储机构基础信息、营业状态 |
| `checkup_packages` | 体检套餐 | 存储套餐定义、分类、价格、描述 |
| `institution_packages` | 机构-套餐绑定 | 控制某机构可服务哪些套餐 |
| `checkup_items` | 体检项目 | 存储单项体检项目、科室、时长、价格 |
| `package_items` | 套餐项目组合 | 定义套餐包含哪些项目及排序 |
| `coupons` | 优惠券/自动优惠规则 | 存储优惠码、自动生效规则、适用人群与时段 |
| `family_members` | 家庭成员 | 存储用户代预约对象信息 |
| `package_favorites` | 套餐收藏 | 记录用户收藏的套餐 |
| `package_browse_histories` | 套餐浏览历史 | 记录用户浏览套餐频次与最近浏览时间 |
| `schedule_templates` | 排班模板 | 存储医生在某机构的周排班模板 |
| `schedule_slots` | 医生号源 | 存储实际可预约时段与容量 |
| `appointments` | 预约单 | 存储预约主体、时间、支付、发票、状态 |
| `appointment_items` | 预约项目明细 | 把预约时的套餐项目快照固化下来 |
| `waitlist_entries` | 候补记录 | 存储满号后的候补申请 |
| `reports` | 体检报告 | 存储报告摘要、结论、建议及归属关系 |
| `service_reviews` | 服务评价 | 存储用户对预约服务的评价与医生/管理员回复 |
| `mail_logs` | 邮件日志 | 记录邮件发送结果与失败原因 |
| `login_logs` | 登录日志 | 记录登录行为、IP、UA、失败原因 |
| `operation_logs` | 操作日志 | 记录后台或业务操作审计轨迹 |
| `role_permissions` | 角色权限 | 存储角色到权限点的开关关系 |
| `notifications` | 站内通知 | 存储用户通知、阅读状态 |
| `system_announcements` | 系统公告 | 存储管理员发布给用户/医生的公告内容 |
| `support_tickets` | 客服工单 | 存储用户咨询、回复、处理状态 |
| `system_settings` | 系统设置 | 存储系统配置项、类型、分组与展示标签 |

## `users`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `User.ID` | 用户主键 |
| `name` | `varchar(64)` | `NOT NULL` | `User.Name` | 用户姓名 |
| `phone` | `varchar(64)` | 唯一索引 | `User.Phone` | 手机号，可用于登录 |
| `password_hash` | `varchar(255)` | `NOT NULL`，默认空串 | `User.PasswordHash` | 密码哈希 |
| `role` | `varchar(16)` | `NOT NULL` | `User.Role` | 角色，如用户、医生、管理员 |
| `status` | `varchar(16)` | `NOT NULL`，默认 `active` | `User.Status` | 账号状态 |
| `gender` | `varchar(16)` | - | `User.Gender` | 性别 |
| `age` | `int` | - | `User.Age` | 年龄 |
| `id_card` | `varchar(32)` | - | `User.IDCard` | 身份证号 |
| `email` | `varchar(128)` | 普通索引 | `User.Email` | 邮箱 |
| `avatar_url` | `varchar(255)` | - | `User.AvatarURL` | 头像地址 |
| `bio` | `text` | - | `User.Bio` | 个人简介 |
| `email_notify` | `bool` | `NOT NULL`，默认 `true` | `User.EmailNotify` | 是否接收邮件通知 |
| `employee_no` | `varchar(32)` | - | `User.EmployeeNo` | 医生/员工工号 |
| `department` | `varchar(64)` | - | `User.Department` | 科室 |
| `title` | `varchar(64)` | - | `User.Title` | 职称 |
| `specialties` | `text` | - | `User.Specialties` | 擅长方向 |
| `created_at` | `datetime` | - | `User.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `User.UpdatedAt` | 更新时间 |

## `checkup_institutions`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `CheckupInstitution.ID` | 机构主键 |
| `name` | `varchar(128)` | `NOT NULL`，唯一索引 | `CheckupInstitution.Name` | 机构名称 |
| `address` | `varchar(255)` | `NOT NULL` | `CheckupInstitution.Address` | 机构地址 |
| `phone` | `varchar(32)` | - | `CheckupInstitution.Phone` | 联系电话 |
| `open_hours` | `varchar(128)` | - | `CheckupInstitution.OpenHours` | 营业时间描述 |
| `status` | `varchar(16)` | `NOT NULL`，默认 `active` | `CheckupInstitution.Status` | 机构状态 |
| `created_at` | `datetime` | - | `CheckupInstitution.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `CheckupInstitution.UpdatedAt` | 更新时间 |

## `checkup_packages`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `CheckupPackage.ID` | 套餐主键 |
| `name` | `varchar(128)` | `NOT NULL` | `CheckupPackage.Name` | 套餐名称 |
| `category` | `varchar(64)` | `NOT NULL`，默认 `综合体检` | `CheckupPackage.Category` | 套餐分类 |
| `description` | `text` | - | `CheckupPackage.Description` | 套餐说明 |
| `price` | `float64` | `NOT NULL` | `CheckupPackage.Price` | 套餐价格 |
| `items` | `text` | - | `CheckupPackage.Items` | 项目摘要文本 |
| `status` | `varchar(16)` | `NOT NULL`，默认 `active` | `CheckupPackage.Status` | 套餐状态 |
| `created_at` | `datetime` | - | `CheckupPackage.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `CheckupPackage.UpdatedAt` | 更新时间 |

## `institution_packages`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `InstitutionPackage.ID` | 绑定关系主键 |
| `institution_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_institution_package` | `InstitutionPackage.InstitutionID` | 机构 ID |
| `package_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_institution_package` | `InstitutionPackage.PackageID` | 套餐 ID |
| `created_at` | `datetime` | - | `InstitutionPackage.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `InstitutionPackage.UpdatedAt` | 更新时间 |

## `checkup_items`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `CheckupItem.ID` | 体检项目主键 |
| `name` | `varchar(128)` | `NOT NULL` | `CheckupItem.Name` | 项目名称 |
| `category` | `varchar(64)` | `NOT NULL` | `CheckupItem.Category` | 项目分类 |
| `department` | `varchar(64)` | - | `CheckupItem.Department` | 执行科室 |
| `price` | `float64` | `NOT NULL`，默认 `0` | `CheckupItem.Price` | 单项价格 |
| `duration_min` | `int` | `NOT NULL`，默认 `10` | `CheckupItem.DurationMin` | 时长（分钟） |
| `description` | `text` | - | `CheckupItem.Description` | 项目说明 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `active`，普通索引 | `CheckupItem.Status` | 状态 |
| `created_at` | `datetime` | - | `CheckupItem.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `CheckupItem.UpdatedAt` | 更新时间 |

## `package_items`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `PackageItem.ID` | 套餐项目关联主键 |
| `package_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_package_item` | `PackageItem.PackageID` | 套餐 ID |
| `item_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_package_item` | `PackageItem.ItemID` | 项目 ID |
| `sort_order` | `int` | `NOT NULL`，默认 `0` | `PackageItem.SortOrder` | 展示排序 |
| `required` | `bool` | `NOT NULL`，默认 `true` | `PackageItem.Required` | 是否必检 |
| `created_at` | `datetime` | - | `PackageItem.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `PackageItem.UpdatedAt` | 更新时间 |

## `coupons`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `Coupon.ID` | 优惠规则主键 |
| `name` | `varchar(128)` | `NOT NULL` | `Coupon.Name` | 优惠名称 |
| `code` | `varchar(64)` | `NOT NULL`，唯一索引 | `Coupon.Code` | 优惠码 |
| `type` | `varchar(24)` | `NOT NULL`，默认 `amount` | `Coupon.Type` | 优惠类型，如满减/折扣 |
| `value` | `float64` | `NOT NULL` | `Coupon.Value` | 优惠值 |
| `min_amount` | `float64` | `NOT NULL`，默认 `0` | `Coupon.MinAmount` | 最低订单金额门槛 |
| `package_id` | `uint` | 普通索引 | `Coupon.PackageID` | 适用套餐 ID，空则通用 |
| `apply_mode` | `varchar(24)` | `NOT NULL`，默认 `auto`，普通索引 | `Coupon.ApplyMode` | 生效方式，自动或手动 |
| `audience` | `varchar(32)` | `NOT NULL`，默认 `all`，普通索引 | `Coupon.Audience` | 适用人群 |
| `first_order_only` | `bool` | `NOT NULL`，默认 `false` | `Coupon.FirstOrderOnly` | 是否仅首单 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `active`，普通索引 | `Coupon.Status` | 状态 |
| `start_date` | `varchar(16)` | - | `Coupon.StartDate` | 生效开始日期 |
| `end_date` | `varchar(16)` | - | `Coupon.EndDate` | 生效结束日期 |
| `description` | `text` | - | `Coupon.Description` | 优惠说明 |
| `created_at` | `datetime` | - | `Coupon.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `Coupon.UpdatedAt` | 更新时间 |

## `family_members`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `FamilyMember.ID` | 家庭成员主键 |
| `user_id` | `uint` | `NOT NULL`，普通索引 | `FamilyMember.UserID` | 所属用户 ID |
| `name` | `varchar(64)` | `NOT NULL` | `FamilyMember.Name` | 成员姓名 |
| `relation` | `varchar(32)` | `NOT NULL` | `FamilyMember.Relation` | 与用户关系 |
| `gender` | `varchar(16)` | - | `FamilyMember.Gender` | 性别 |
| `age` | `int` | - | `FamilyMember.Age` | 年龄 |
| `id_card` | `varchar(32)` | - | `FamilyMember.IDCard` | 身份证号 |
| `phone` | `varchar(32)` | - | `FamilyMember.Phone` | 联系电话 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `active`，普通索引 | `FamilyMember.Status` | 状态 |
| `created_at` | `datetime` | - | `FamilyMember.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `FamilyMember.UpdatedAt` | 更新时间 |

## `package_favorites`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `PackageFavorite.ID` | 收藏记录主键 |
| `user_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_user_package_favorite` | `PackageFavorite.UserID` | 用户 ID |
| `package_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_user_package_favorite` | `PackageFavorite.PackageID` | 套餐 ID |
| `created_at` | `datetime` | - | `PackageFavorite.CreatedAt` | 创建时间 |

## `package_browse_histories`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `PackageBrowseHistory.ID` | 浏览记录主键 |
| `user_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_user_package_browse` | `PackageBrowseHistory.UserID` | 用户 ID |
| `package_id` | `uint` | `NOT NULL`，联合唯一索引 `idx_user_package_browse` | `PackageBrowseHistory.PackageID` | 套餐 ID |
| `view_count` | `int` | `NOT NULL`，默认 `1` | `PackageBrowseHistory.ViewCount` | 浏览次数 |
| `viewed_at` | `datetime` | 普通索引 | `PackageBrowseHistory.ViewedAt` | 最近浏览时间 |
| `created_at` | `datetime` | - | `PackageBrowseHistory.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `PackageBrowseHistory.UpdatedAt` | 更新时间 |

## `schedule_templates`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `ScheduleTemplate.ID` | 排班模板主键 |
| `doctor_id` | `uint` | `NOT NULL`，普通索引 | `ScheduleTemplate.DoctorID` | 医生 ID |
| `institution_id` | `uint` | `NOT NULL`，普通索引 | `ScheduleTemplate.InstitutionID` | 机构 ID |
| `category` | `varchar(64)` | `NOT NULL`，默认 `综合体检`，普通索引 | `ScheduleTemplate.Category` | 排班适用分类 |
| `weekdays` | `text` | - | `ScheduleTemplate.WeekdaysText` | 周几数组的 JSON 文本 |
| `start_times` | `text` | - | `ScheduleTemplate.StartTimesText` | 开始时间数组的 JSON 文本 |
| `capacity` | `int` | `NOT NULL`，默认 `1` | `ScheduleTemplate.Capacity` | 单个时段默认容量 |
| `status` | `varchar(16)` | `NOT NULL`，默认 `available`，普通索引 | `ScheduleTemplate.Status` | 模板状态 |
| `created_at` | `datetime` | - | `ScheduleTemplate.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `ScheduleTemplate.UpdatedAt` | 更新时间 |

## `schedule_slots`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `ScheduleSlot.ID` | 号源主键 |
| `template_id` | `uint` | 普通索引 | `ScheduleSlot.TemplateID` | 来源排班模板 ID |
| `doctor_id` | `uint` | `NOT NULL`，普通索引 | `ScheduleSlot.DoctorID` | 医生 ID |
| `institution_id` | `uint` | `NOT NULL`，普通索引 | `ScheduleSlot.InstitutionID` | 机构 ID |
| `date` | `varchar(16)` | `NOT NULL`，普通索引 | `ScheduleSlot.Date` | 实际出诊日期 |
| `period` | `varchar(32)` | `NOT NULL`，普通索引 | `ScheduleSlot.Period` | 上午/下午或时段标签 |
| `category` | `varchar(64)` | `NOT NULL`，默认 `综合体检`，普通索引 | `ScheduleSlot.Category` | 可服务的体检分类 |
| `start_time` | `varchar(8)` | `NOT NULL` | `ScheduleSlot.StartTime` | 开始时间 |
| `end_time` | `varchar(8)` | `NOT NULL` | `ScheduleSlot.EndTime` | 结束时间 |
| `capacity` | `int` | `NOT NULL`，默认 `1` | `ScheduleSlot.Capacity` | 号源容量 |
| `booked_count` | `int` | `NOT NULL`，默认 `0` | `ScheduleSlot.BookedCount` | 已预约人数 |
| `status` | `varchar(16)` | `NOT NULL`，默认 `available` | `ScheduleSlot.Status` | 号源状态 |
| `created_at` | `datetime` | - | `ScheduleSlot.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `ScheduleSlot.UpdatedAt` | 更新时间 |

## `appointments`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `Appointment.ID` | 预约主键 |
| `order_no` | `varchar(32)` | 唯一索引 | `Appointment.OrderNo` | 预约单号 |
| `user_id` | `uint` | `NOT NULL`，普通索引 | `Appointment.UserID` | 下单用户 ID |
| `family_member_id` | `uint` | 普通索引 | `Appointment.FamilyMemberID` | 代预约家庭成员 ID |
| `doctor_id` | `uint` | 普通索引 | `Appointment.DoctorID` | 接诊医生 ID |
| `institution_id` | `uint` | `NOT NULL`，普通索引 | `Appointment.InstitutionID` | 机构 ID |
| `slot_id` | `uint` | 普通索引 | `Appointment.SlotID` | 绑定号源 ID |
| `package_id` | `uint` | `NOT NULL`，普通索引 | `Appointment.PackageID` | 套餐 ID |
| `coupon_id` | `uint` | 普通索引 | `Appointment.CouponID` | 使用优惠规则 ID |
| `appointment_type` | `varchar(32)` | `NOT NULL`，默认 `个人体检` | `Appointment.AppointmentType` | 预约类型 |
| `category` | `varchar(64)` | `NOT NULL`，默认 `综合体检`，普通索引 | `Appointment.Category` | 体检分类 |
| `date` | `varchar(16)` | `NOT NULL` | `Appointment.Date` | 预约日期 |
| `period` | `varchar(32)` | `NOT NULL` | `Appointment.Period` | 预约时段 |
| `start_time` | `varchar(8)` | - | `Appointment.StartTime` | 开始时间 |
| `end_time` | `varchar(8)` | - | `Appointment.EndTime` | 结束时间 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `booked` | `Appointment.Status` | 预约状态 |
| `note` | `text` | - | `Appointment.Note` | 备注 |
| `payment_status` | `varchar(24)` | `NOT NULL`，默认 `unpaid` | `Appointment.PaymentStatus` | 支付状态 |
| `original_amount` | `float64` | `NOT NULL`，默认 `0` | `Appointment.OriginalAmount` | 原价金额 |
| `discount_amount` | `float64` | `NOT NULL`，默认 `0` | `Appointment.DiscountAmount` | 优惠金额 |
| `payable_amount` | `float64` | `NOT NULL`，默认 `0` | `Appointment.PayableAmount` | 应付金额 |
| `invoice_title` | `varchar(128)` | - | `Appointment.InvoiceTitle` | 发票抬头 |
| `invoice_tax_no` | `varchar(64)` | - | `Appointment.InvoiceTaxNo` | 税号 |
| `invoice_status` | `varchar(24)` | `NOT NULL`，默认 `none`，普通索引 | `Appointment.InvoiceStatus` | 发票状态 |
| `created_at` | `datetime` | - | `Appointment.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `Appointment.UpdatedAt` | 更新时间 |

## `appointment_items`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `AppointmentItem.ID` | 预约项目明细主键 |
| `appointment_id` | `uint` | `NOT NULL`，普通索引 | `AppointmentItem.AppointmentID` | 预约 ID |
| `package_item_id` | `uint` | `NOT NULL`，普通索引 | `AppointmentItem.PackageItemID` | 套餐项目关系 ID |
| `item_id` | `uint` | `NOT NULL`，普通索引 | `AppointmentItem.ItemID` | 体检项目 ID |
| `required` | `bool` | `NOT NULL`，默认 `true` | `AppointmentItem.Required` | 是否必检 |
| `price` | `float64` | `NOT NULL`，默认 `0` | `AppointmentItem.Price` | 预约时快照价格 |
| `created_at` | `datetime` | - | `AppointmentItem.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `AppointmentItem.UpdatedAt` | 更新时间 |

## `waitlist_entries`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `WaitlistEntry.ID` | 候补记录主键 |
| `user_id` | `uint` | `NOT NULL`，普通索引 | `WaitlistEntry.UserID` | 用户 ID |
| `package_id` | `uint` | `NOT NULL`，普通索引 | `WaitlistEntry.PackageID` | 套餐 ID |
| `institution_id` | `uint` | `NOT NULL`，普通索引 | `WaitlistEntry.InstitutionID` | 机构 ID |
| `appointment_type` | `varchar(32)` | `NOT NULL`，默认 `个人体检` | `WaitlistEntry.AppointmentType` | 候补类型 |
| `category` | `varchar(64)` | `NOT NULL`，默认 `综合体检`，普通索引 | `WaitlistEntry.Category` | 体检分类 |
| `date` | `varchar(16)` | `NOT NULL`，普通索引 | `WaitlistEntry.Date` | 目标日期 |
| `period` | `varchar(32)` | `NOT NULL`，普通索引 | `WaitlistEntry.Period` | 目标时段 |
| `start_time` | `varchar(8)` | 普通索引 | `WaitlistEntry.StartTime` | 目标开始时间 |
| `end_time` | `varchar(8)` | - | `WaitlistEntry.EndTime` | 目标结束时间 |
| `note` | `text` | - | `WaitlistEntry.Note` | 候补备注 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `waiting` | `WaitlistEntry.Status` | 候补状态 |
| `created_at` | `datetime` | - | `WaitlistEntry.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `WaitlistEntry.UpdatedAt` | 更新时间 |

## `reports`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `Report.ID` | 报告主键 |
| `report_no` | `varchar(32)` | 唯一索引 | `Report.ReportNo` | 报告编号 |
| `appointment_id` | `uint` | `NOT NULL`，唯一索引 | `Report.AppointmentID` | 对应预约 ID，一单一报告 |
| `user_id` | `uint` | `NOT NULL`，普通索引 | `Report.UserID` | 报告所属用户 ID |
| `doctor_id` | `uint` | `NOT NULL`，普通索引 | `Report.DoctorID` | 录入/负责医生 ID |
| `summary` | `text` | `NOT NULL` | `Report.Summary` | 摘要 |
| `conclusion` | `text` | `NOT NULL` | `Report.Conclusion` | 结论 |
| `recommendation` | `text` | - | `Report.Recommendation` | 建议 |
| `created_at` | `datetime` | - | `Report.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `Report.UpdatedAt` | 更新时间 |

## `service_reviews`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `ServiceReview.ID` | 评价主键 |
| `user_id` | `uint` | `NOT NULL`，普通索引 | `ServiceReview.UserID` | 评价用户 ID |
| `appointment_id` | `uint` | `NOT NULL`，唯一索引 | `ServiceReview.AppointmentID` | 对应预约 ID，一单一评 |
| `package_id` | `uint` | `NOT NULL`，普通索引 | `ServiceReview.PackageID` | 套餐 ID |
| `institution_id` | `uint` | `NOT NULL`，普通索引 | `ServiceReview.InstitutionID` | 机构 ID |
| `doctor_id` | `uint` | 普通索引 | `ServiceReview.DoctorID` | 医生 ID |
| `rating` | `int` | `NOT NULL` | `ServiceReview.Rating` | 评分 |
| `content` | `text` | - | `ServiceReview.Content` | 评价内容 |
| `reply` | `text` | - | `ServiceReview.Reply` | 回复内容 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `published`，普通索引 | `ServiceReview.Status` | 评价状态 |
| `created_at` | `datetime` | - | `ServiceReview.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `ServiceReview.UpdatedAt` | 更新时间 |

## `mail_logs`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `MailLog.ID` | 邮件日志主键 |
| `user_id` | `uint` | 普通索引 | `MailLog.UserID` | 关联用户 ID |
| `to` | `varchar(128)` | `NOT NULL` | `MailLog.To` | 收件地址 |
| `subject` | `varchar(255)` | `NOT NULL` | `MailLog.Subject` | 邮件主题 |
| `body` | `text` | - | `MailLog.Body` | 邮件正文 |
| `status` | `varchar(24)` | `NOT NULL` | `MailLog.Status` | 发送状态 |
| `error` | `text` | - | `MailLog.Error` | 失败原因 |
| `created_at` | `datetime` | - | `MailLog.CreatedAt` | 创建时间 |

## `login_logs`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `LoginLog.ID` | 登录日志主键 |
| `user_id` | `uint` | 普通索引 | `LoginLog.UserID` | 登录用户 ID |
| `email` | `varchar(128)` | 普通索引 | `LoginLog.Email` | 登录邮箱 |
| `role` | `varchar(16)` | - | `LoginLog.Role` | 登录角色 |
| `ip` | `varchar(64)` | 普通索引 | `LoginLog.IP` | 来源 IP |
| `user_agent` | `varchar(255)` | - | `LoginLog.UserAgent` | 浏览器/客户端标识 |
| `status` | `varchar(24)` | `NOT NULL`，普通索引 | `LoginLog.Status` | 登录结果 |
| `reason` | `varchar(255)` | - | `LoginLog.Reason` | 失败原因 |
| `created_at` | `datetime` | - | `LoginLog.CreatedAt` | 创建时间 |

## `operation_logs`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `OperationLog.ID` | 操作日志主键 |
| `user_id` | `uint` | 普通索引 | `OperationLog.UserID` | 操作人 ID |
| `user_name` | `varchar(64)` | - | `OperationLog.UserName` | 操作人姓名 |
| `role` | `varchar(16)` | 普通索引 | `OperationLog.Role` | 操作人角色 |
| `action` | `varchar(64)` | `NOT NULL`，普通索引 | `OperationLog.Action` | 操作动作 |
| `resource` | `varchar(64)` | `NOT NULL`，普通索引 | `OperationLog.Resource` | 资源类型 |
| `resource_id` | `varchar(64)` | - | `OperationLog.ResourceID` | 资源标识 |
| `method` | `varchar(16)` | - | `OperationLog.Method` | HTTP 方法 |
| `path` | `varchar(255)` | - | `OperationLog.Path` | 请求路径 |
| `ip` | `varchar(64)` | 普通索引 | `OperationLog.IP` | 来源 IP |
| `status` | `varchar(24)` | `NOT NULL`，普通索引 | `OperationLog.Status` | 执行结果 |
| `detail` | `text` | - | `OperationLog.Detail` | 详细说明 |
| `created_at` | `datetime` | - | `OperationLog.CreatedAt` | 创建时间 |

## `role_permissions`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `RolePermission.ID` | 权限记录主键 |
| `role` | `varchar(16)` | `NOT NULL`，联合唯一索引 `idx_role_permission` | `RolePermission.Role` | 角色名 |
| `permission` | `varchar(64)` | `NOT NULL`，联合唯一索引 `idx_role_permission` | `RolePermission.Permission` | 权限点编码 |
| `description` | `varchar(255)` | - | `RolePermission.Description` | 权限说明 |
| `enabled` | `bool` | `NOT NULL`，默认 `true`，普通索引 | `RolePermission.Enabled` | 是否启用 |
| `created_at` | `datetime` | - | `RolePermission.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `RolePermission.UpdatedAt` | 更新时间 |

## `notifications`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `Notification.ID` | 通知主键 |
| `user_id` | `uint` | `NOT NULL`，普通索引 | `Notification.UserID` | 接收用户 ID |
| `channel` | `varchar(24)` | `NOT NULL`，默认 `in_app` | `Notification.Channel` | 通知渠道 |
| `type` | `varchar(32)` | `NOT NULL` | `Notification.Type` | 通知类型 |
| `title` | `varchar(128)` | `NOT NULL` | `Notification.Title` | 通知标题 |
| `content` | `text` | - | `Notification.Content` | 通知内容 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `unread`，普通索引 | `Notification.Status` | 阅读状态 |
| `created_at` | `datetime` | - | `Notification.CreatedAt` | 创建时间 |
| `read_at` | `datetime` | 可空 | `Notification.ReadAt` | 读取时间 |

## `system_announcements`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `SystemAnnouncement.ID` | 公告主键 |
| `title` | `varchar(128)` | `NOT NULL` | `SystemAnnouncement.Title` | 公告标题 |
| `content` | `text` | `NOT NULL` | `SystemAnnouncement.Content` | 公告正文 |
| `audience` | `varchar(24)` | `NOT NULL`，默认 `all`，普通索引 | `SystemAnnouncement.Audience` | 面向人群，如用户/医生/全体 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `draft`，普通索引 | `SystemAnnouncement.Status` | 发布状态 |
| `published_at` | `datetime` | 可空 | `SystemAnnouncement.PublishedAt` | 发布时间 |
| `created_at` | `datetime` | - | `SystemAnnouncement.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `SystemAnnouncement.UpdatedAt` | 更新时间 |

## `support_tickets`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `SupportTicket.ID` | 工单主键 |
| `user_id` | `uint` | `NOT NULL`，普通索引 | `SupportTicket.UserID` | 提单用户 ID |
| `subject` | `varchar(128)` | `NOT NULL` | `SupportTicket.Subject` | 工单主题 |
| `content` | `text` | `NOT NULL` | `SupportTicket.Content` | 问题内容 |
| `reply` | `text` | - | `SupportTicket.Reply` | 客服回复 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `open`，普通索引 | `SupportTicket.Status` | 工单状态 |
| `created_at` | `datetime` | - | `SupportTicket.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `SupportTicket.UpdatedAt` | 更新时间 |

## `system_settings`

| 字段名 | 数据类型 | 约束/索引 | 对应实体属性 | 字段说明 |
| --- | --- | --- | --- | --- |
| `id` | `uint` | 主键 | `SystemSetting.ID` | 设置主键 |
| `key` | `varchar(64)` | `NOT NULL`，唯一索引 | `SystemSetting.Key` | 配置键 |
| `value` | `text` | - | `SystemSetting.Value` | 配置值 |
| `value_type` | `varchar(24)` | `NOT NULL`，默认 `string` | `SystemSetting.ValueType` | 值类型 |
| `group` | `varchar(32)` | `NOT NULL`，默认 `system`，普通索引 | `SystemSetting.Group` | 配置分组 |
| `label` | `varchar(128)` | `NOT NULL` | `SystemSetting.Label` | 后台展示名称 |
| `description` | `varchar(255)` | - | `SystemSetting.Description` | 配置说明 |
| `status` | `varchar(24)` | `NOT NULL`，默认 `active`，普通索引 | `SystemSetting.Status` | 状态 |
| `created_at` | `datetime` | - | `SystemSetting.CreatedAt` | 创建时间 |
| `updated_at` | `datetime` | - | `SystemSetting.UpdatedAt` | 更新时间 |
