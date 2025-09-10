# Go Email Service - 前端 API 开发指南

本指南基于项目自动生成的 `swagger.json` 文件，旨在为前端开发人员提供一份清晰、简洁且聚焦于核心功能的 API 参考。它省略了后端的内部实现细节和繁琐的错误码定义，只保留了与前端 UI/UX 开发直接相关的接口、参数和数据结构。

## 目录

1.  [基础信息](#基础信息)
2.  [环境搭建](#环境搭建)
3.  [认证流程](#认证流程)
4.  [API 接口详情](#api-接口详情)
    -   [认证 (Auth)](#认证-auth)
    -   [用户管理 (Users)](#用户管理-users)
    -   [云账号管理 (Accounts)](#云账号管理-accounts)
    -   [发件人管理 (Senders)](#发件人管理-senders)
    -   [收件人管理 (Recipients)](#收件人管理-recipients)
    -   [收件人分群 (Recipient Groups)](#收件人分群-recipient-groups)
    -   [邮件模板 (Templates)](#邮件模板-templates)
    -   [邮件任务 (Tasks)](#邮件任务-tasks)
    -   [数据统计 (Statistics)](#数据统计-statistics)

---

## 基础信息

-   **API 主机 (Host)**: `localhost:8080`
-   **API 基础路径 (Base Path)**: `/`
-   **通用响应结构**: 所有 API 响应都遵循以下格式。如果 `error` 字段不为空字符串，则表示请求失败。

    ```json
    {
      "data": { ... }, // 成功时，业务数据在此对象中
      "error": ""      // 失败时，此处为错误信息
    }
    ```

## 环境搭建

详细的后端环境搭建步骤，包括如何启动 Go 服务、PostgreSQL 数据库、Redis 和 Elasticsearch，请参考项目根目录下的 `README.md` 文件中的 **"如何运行"** 部分。

## 认证流程

系统采用 JWT (JSON Web Token) 进行认证。

1.  **获取 Token**: 用户通过登录接口获取 Token。
2.  **使用 Token**: 在后续所有需要认证的请求中，必须在 HTTP Header 中加入 `Authorization` 字段，其值为 `Bearer <YOUR_TOKEN>`。

    **示例**: `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`

---

## API 接口详情

### 认证 (Auth)

#### `POST /api/v1/login`
**功能**: 用户登录，获取认证 Token。

**请求体 (Request Body):**
```json
{
  "username": "admin",
  "password": "password123"
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImFkbWluIiwiaXNfYWRtaW4iOnRydWUsImV4cCI6MTcwMDAwMDAwMH0.signature"
  },
  "error": ""
}
```

### 用户管理 (Users)

#### `POST /api/v1/users`
**功能**: 创建一个新用户。此接口需要管理员权限。

**请求体 (Request Body):**
```json
{
  "username": "newuser",
  "email": "new.user@example.com",
  "password": "strongpassword",
  "is_admin": false
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 2,
    "username": "newuser",
    "email": "new.user@example.com",
    "is_active": true,
    "is_admin": false,
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-01T10:00:00Z"
  },
  "error": ""
}
```

#### `GET /api/v1/users` (待开发)
**功能**: 分页获取用户列表。需要管理员权限。

#### `GET /api/v1/users/{id}` (待开发)
**功能**: 根据 ID 获取单个用户详情。

#### `PUT /api/v1/users/{id}` (待开发)
**功能**: 根据 ID 更新用户信息。

#### `DELETE /api/v1/users/{id}` (待开发)
**功能**: 根据 ID 删除一个用户。

### 云账号管理 (Accounts)

云账号指阿里云等邮件服务提供商的账号。

#### `POST /api/v1/accounts`
**功能**: 添加一个新的云账号配置。

**请求体 (Request Body):**
```json
{
  "name": "Aliyun Account 1",
  "domain": "mail.example.com",
  "access_key_id": "LTAI5txxxxxxxxxxxxxx",
  "access_key_secret": "P1aXxxxxxxxxxxxxxxxxxxxxxx",
  "daily_send_limit": 5000
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Aliyun Account 1",
    "access_key_id": "LTAI5txxxxxxxxxxxxxx",
    "domain": "mail.example.com",
    "daily_send_limit": 5000,
    "status": "active",
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-01T10:00:00Z"
  },
  "error": ""
}
```

#### `GET /api/v1/accounts`
**功能**: 获取所有已配置的云账号列表。

**成功响应 (Success Response):**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Aliyun Account 1",
      "access_key_id": "LTAI5txxxxxxxxxxxxxx",
      "domain": "mail.example.com",
      "daily_send_limit": 5000,
      "status": "active",
      "created_at": "2023-11-01T10:00:00Z",
      "updated_at": "2023-11-01T10:00:00Z"
    },
    {
      "id": 2,
      "name": "Backup Account",
      "access_key_id": "LTAI5tyyyyyyyyyyyyyy",
      "domain": "backup.example.com",
      "daily_send_limit": 1000,
      "status": "inactive",
      "created_at": "2023-11-02T11:00:00Z",
      "updated_at": "2023-11-02T11:00:00Z"
    }
  ],
  "error": ""
}
```

#### `GET /api/v1/accounts/{id}`
**功能**: 根据 ID 获取单个云账号详情。

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Aliyun Account 1",
    "access_key_id": "LTAI5txxxxxxxxxxxxxx",
    "domain": "mail.example.com",
    "daily_send_limit": 5000,
    "status": "active",
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-01T10:00:00Z"
  },
  "error": ""
}
```

#### `PUT /api/v1/accounts/{id}`
**功能**: 根据 ID 更新单个云账号信息。

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Aliyun Account 1 (Updated)",
    "access_key_id": "LTAI5txxxxxxxxxxxxxx",
    "domain": "mail.example.com",
    "daily_send_limit": 6000,
    "status": "active",
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-05T14:30:00Z"
  },
  "error": ""
}
```

#### `DELETE /api/v1/accounts/{id}`
**功能**: 根据 ID 删除一个云账号。

**成功响应 (Success Response):**
```json
{
  "data": {
    "message": "Account deleted successfully"
  },
  "error": ""
}
```

### 发件人管理 (Senders)

发件人是发送邮件时"From"字段的实体，它必须关联到一个云账号上。

#### `POST /api/v1/senders`
**功能**: 创建一个发件人实体（如"市场部"、"客服团队"）。

**请求体 (Request Body):**
```json
{
  "name": "Marketing Team",
  "role": "Promotion",
  "contact_info": "marketing@example.com"
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Marketing Team",
    "role": "Promotion",
    "contact_info": "marketing@example.com",
    "created_at": "2023-11-03T10:00:00Z",
    "updated_at": "2023-11-03T10:00:00Z"
  },
  "error": ""
}
```

#### `POST /api/v1/senders/{senderId}/accounts/{accountId}`
**功能**: 将一个发件人实体与云账号关联，创建一个可用的发送地址。

**请求体 (Request Body):**
```json
{
  "email_address": "noreply@example.com",
  "daily_send_limit": 1000
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "account_id": 1,
    "sender_id": 1,
    "email_address": "noreply@example.com",
    "weight": 100,
    "daily_send_limit": 1000,
    "status": "active",
    "created_at": "2023-11-04T10:00:00Z",
    "updated_at": "2023-11-04T10:00:00Z"
  },
  "error": ""
}
```

#### `GET /api/v1/senders` (待开发)
**功能**: 获取所有发件人实体列表。

#### `GET /api/v1/senders/{id}` (待开发)
**功能**: 获取单个发件人实体详情。

#### `GET /api/v1/senders/{senderId}/accounts` (待开发)
**功能**: 获取一个发件人实体下关联的所有发送地址列表。

### 收件人管理 (Recipients)

#### `POST /api/v1/recipients`
**功能**: 添加一个新的收件人。

**请求体 (Request Body):**
```json
{
  "email": "john.doe@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "metadata": {
    "country": "USA",
    "level": "VIP"
  }
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "status": "subscribed",
    "metadata": {
      "country": "USA",
      "level": "VIP"
    },
    "created_at": "2023-11-06T10:00:00Z",
    "updated_at": "2023-11-06T10:00:00Z"
  },
  "error": ""
}
```

#### `GET /api/v1/recipients`
**功能**: 分页获取收件人列表，并支持按姓名、邮箱和元数据进行筛选。支持传统分页和search_after深度分页两种模式。

**分页模式:**

**1. 传统分页 (适用于浅层页面，页码 × 每页数量 < 10000)**
**查询参数 (Query Params)**: 
- `page` (页码, 默认为 1)
- `pageSize` (每页数量, 默认为 10)
- `name` (可选, 字符串): 按姓名进行模糊搜索 (匹配姓或名)。
- `email` (可选, 字符串): 按邮箱地址进行精确匹配。
- `metadata.<key>` (可选, 字符串): 按元数据字段进行精确匹配，例如 `metadata.country=USA`。

**传统分页响应 (Traditional Pagination Response):**
```json
{
  "data": [
    {
      "id": 1,
      "email": "john.doe@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "status": "subscribed",
      "metadata": { "country": "USA", "level": "VIP" },
      "created_at": "2023-11-06T10:00:00Z",
      "updated_at": "2023-11-06T10:00:00Z"
    },
    {
      "id": 2,
      "email": "jane.doe@example.com",
      "first_name": "Jane",
      "last_name": "Doe",
      "status": "subscribed",
      "metadata": { "country": "Canada", "level": "Gold" },
      "created_at": "2023-11-05T11:00:00Z",
      "updated_at": "2023-11-05T11:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total": 120,
    "total_pages": 12,
    "has_next": true
  },
  "error": ""
}
```

**2. Search_After分页 (适用于深层页面，无限制)**
**查询参数 (Query Params)**: 
- `search_after` (可选, 字符串): Base64编码的search_after值，用于获取下一页
- `page_size` (每页数量, 默认为 50)
- `name` (可选, 字符串): 按姓名进行模糊搜索 (匹配姓或名)。
- `email` (可选, 字符串): 按邮箱地址进行精确匹配。
- `metadata.<key>` (可选, 字符串): 按元数据字段进行精确匹配，例如 `metadata.country=USA`。

**Search_After分页响应 (Search_After Pagination Response):**
```json
{
  "data": [
    {
      "id": 100001,
      "email": "user100001@example.com",
      "first_name": "User",
      "last_name": "100001",
      "status": "subscribed",
      "metadata": { "country": "USA", "level": "Standard" },
      "created_at": "2023-11-01T10:00:00Z",
      "updated_at": "2023-11-01T10:00:00Z"
    }
  ],
  "pagination": {
    "page_size": 50,
    "total": 1234567,
    "next_search_after": "WzEwMDAwMV0=",
    "has_next": true
  },
  "error": ""
}
```

**分页最佳实践:**
- 对于前几页的数据浏览，使用传统分页 (`page` + `pageSize`)
- 当请求深层页面时(如第1000页以后)，系统会返回错误提示，建议使用search_after分页
- Search_after分页不支持跳页，只能按顺序翻页，但支持无限深度
- 使用search_after时，将上一页响应中的`next_search_after`值作为下一页请求的`search_after`参数
```

#### `GET /api/v1/recipients/{id}`
**功能**: 根据 ID 获取单个收件人详情。

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "status": "subscribed",
    "metadata": {
      "country": "USA",
      "level": "VIP"
    },
    "created_at": "2023-11-06T10:00:00Z",
    "updated_at": "2023-11-06T10:00:00Z"
  },
  "error": ""
}
```

#### `PUT /api/v1/recipients/{id}`
**功能**: 根据 ID 更新收件人信息。

**请求体 (Request Body):**
```json
{
  "email": "john.doe.updated@example.com",
  "first_name": "Johnathan",
  "last_name": "Doe",
  "status": "unsubscribed",
  "metadata": {
    "country": "USA",
    "level": "Gold"
  }
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "email": "john.doe.updated@example.com",
    "first_name": "Johnathan",
    "last_name": "Doe",
    "status": "unsubscribed",
    "metadata": {
      "country": "USA",
      "level": "Gold"
    },
    "created_at": "2023-11-06T10:00:00Z",
    "updated_at": "2023-11-06T14:30:00Z"
  },
  "error": ""
}
```

#### `DELETE /api/v1/recipients/{id}`
**功能**: 根据 ID 删除一个收件人。

**成功响应 (Success Response):**
```json
{
  "data": {
    "message": "Recipient deleted successfully"
  },
  "error": ""
}
```

### 收件人分群 (Recipient Groups)

收件人分群是动态（基于规则）或静态（基于列表）的收件人集合。

#### `POST /api/v1/recipient-groups`
**功能**: 创建一个新的收件人分群。
- **动态分群**: 提供 `rules` 数组。
- **静态分群**: 提供 `member_ids` 数组。

**请求体 (Request Body for Dynamic Group):**
```json
{
  "name": "Active US Users",
  "description": "Users in the USA who are active.",
  "group_type": "dynamic",
  "rules": [
    {
      "field": "metadata.country",
      "operator": "equals",
      "value": "USA"
    },
    {
      "field": "status",
      "operator": "equals",
      "value": "active"
    }
  ]
}
```

#### `GET /api/v1/recipient-groups`
**功能**: 分页获取收件人分群列表。
**查询参数 (Query Params)**: `page`, `pageSize`.

#### `GET /api/v1/recipient-groups/{id}`
**功能**: 根据 ID 获取分群详情。

#### `PUT /api/v1/recipient-groups/{id}`
**功能**: 根据 ID 更新分群信息。

#### `DELETE /api/v1/recipient-groups/{id}`
**功能**: 根据 ID 删除分群。

### 邮件模板 (Templates)

#### `POST /api/v1/templates`
**功能**: 创建一个新的邮件模板。模板变量使用 Go 的 `text/template` 语法，如 `{{.FirstName}}`。

**请求体 (Request Body):**
```json
{
  "name": "Welcome Email",
  "subject": "Welcome to Our Service, {{.FirstName}}!",
  "body": "<h1>Hello {{.FirstName}}!</h1><p>Thank you for joining us. Your registered email is {{.Email}}.</p>"
}
```

#### `GET /api/v1/templates`
**功能**: 获取所有邮件模板列表。

#### `POST /api/v1/templates/{id}/preview`
**功能**: 使用示例数据预览模板渲染效果。

**请求体 (Request Body):**
```json
{
  "email": "preview@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "metadata": {
    "custom_field": "some value"
  }
}
```
**成功响应 (Success Response):**
```json
{
    "data": {
        "subject": "Welcome to Our Service, John!",
        "body": "<h1>Hello John!</h1><p>Thank you for joining us. Your registered email is preview@example.com.</p>"
    },
    "error": ""
}
```

### 邮件任务 (Tasks)

#### `POST /api/v1/tasks`
**功能**: 创建一个批量发送邮件的任务。这是一个异步接口，会立即返回任务ID。
- 任务内容可以来自 `template_id`，也可以直接提供 `subject` 和 `body`。
- 发送对象由 `recipient_group_id` 指定。
- 发件人由系统根据负载均衡策略自动选择，无需前端指定。

**请求体 (Request Body):**
```json
{
  "task_name": "Q4 Promotion",
  "recipient_group_id": 5,
  "template_id": 10,
  "subject": "Special Offer Inside!",
  "body": "<p>This is an alternative body if no template is used.</p>",
  "scheduled_at": "2025-01-01T12:00:00Z"
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "message": "Email task created and scheduled for dispatch",
    "task_id": 123
  },
  "error": ""
}
```

#### 任务状态说明 (Task Status Explanation)
一个邮件任务在系统中会经历以下几种状态，前端可以根据不同状态展示不同的 UI 提示：
-   **`pending`**: 待处理。任务已创建，等待调度系统派发。
-   **`dispatching`**: 分发中。调度器正在处理该任务，计算收件人列表，并生成独立的邮件发送作业。
-   **`sending`**: 发送中。邮件发送作业已被推送到队列，后台 Worker 正在并发发送邮件。
-   **`completed`**: 已完成。所有邮件发送作业都已执行完毕。
-   **`failed`**: 失败。任务在分发或处理过程中遇到严重错误导致无法继续。

### 数据统计 (Statistics)

#### `GET /api/v1/statistics`
**功能**: 获取多维度的全局邮件发送统计数据。

**查询参数 (Query Params):**
- `start_date`: `YYYY-MM-DD`
- `end_date`: `YYYY-MM-DD`
- `account_id`: (可选) 按云账号过滤
- `account_sender_id`: (可选) 按发件人过滤
- `group_by`: `day`, `week`, `month`, `sender`

**成功响应 (Success Response Example):**
```json
{
  "data": {
    "period": { "start_date": "2023-10-01", "end_date": "2023-10-31", "days": 30 },
    "summary": {
      "total_sent": 10000,
      "total_opened": 2500,
      "total_unique_opened": 1800,
      // ... more summary fields
    },
    "time_series": [
      { "date": "2023-10-01", "sent_count": 500, "open_rate": 0.25, ... },
      { "date": "2023-10-02", "sent_count": 600, "open_rate": 0.28, ... }
    ],
    "by_sender": [
      { "sender_email": "a@example.com", "sent_count": 5000, "open_rate": 0.3, ... },
      { "sender_email": "b@example.com", "sent_count": 5000, "open_rate": 0.2, ... }
    ]
  },
  "error": ""
}
```

#### `GET /api/v1/tasks/{id}`
**功能**: 获取单个任务的发送统计摘要。

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 123,
    "task_name": "Q4 Promotion",
    "status": "completed",
    "total_recipients": 5000,
    "open_count": 1500,
    "open_rate": 0.3,
    // ... more summary fields
  },
  "error": ""
}
```

#### `GET /api/v1/tasks/{id}/records`
**功能**: 获取指定任务下所有邮件的详细发送记录。 

**查询参数 (Query Params)**: `page` (页码), `pageSize` (每页数量)。

**成功响应 (Success Response):**
```json
{
  "data": {
    "records": [
      {
        "id": 1001,
        "task_id": 123,
        "account_sender_id": 5,
        "recipient_email": "user1@example.com",
        "subject": "Special Offer Inside!",
        "status": "delivered",
        "error_message": null,
        "sent_at": "2025-01-01T12:05:10Z",
        "last_status_update_at": "2025-01-01T12:05:15Z"
      },
      {
        "id": 1002,
        "task_id": 123,
        "account_sender_id": 5,
        "recipient_email": "user2@example.com",
        "subject": "Special Offer Inside!",
        "status": "failed",
        "error_message": "Invalid recipient email address",
        "sent_at": "2025-01-01T12:05:11Z",
        "last_status_update_at": "2025-01-01T12:05:11Z"
      },
      {
        "id": 1003,
        "task_id": 123,
        "account_sender_id": 6,
        "recipient_email": "user3@example.com",
        "subject": "Special Offer Inside!",
        "status": "bounce",
        "error_message": "Recipient inbox is full",
        "sent_at": "2025-01-01T12:05:12Z",
        "last_status_update_at": "2025-01-01T12:10:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 20,
      "total_records": 5000,
      "total_pages": 250
    }
  },
  "error": ""
}
```

#### 记录状态说明 (Record Status Explanation)
每一封邮件的发送记录（Record）都有自己的状态，这些状态由云服务商回调更新：
-   **`pending`**: 排队中，等待发送。
-   **`sending`**: 发送中，已交给云服务商。
-   **`sent`**: 发送成功，对方邮件服务器已接收。
-   **`failed`**: 发送失败，发送请求阶段就失败了（如格式错误）。
-   **`delivered`**: 投递成功，邮件已成功放入收件人邮箱。
-   **`opened`**: 已打开。
-   **`clicked`**: 已点击邮件中的链接。
-   **`bounce`**: 邮件被弹回，无法送达（如邮箱不存在、邮箱已满）。

### 通用功能

#### 分页 (Pagination)

对于返回列表的接口（如 `GET /api/v1/recipients`, `GET /api/v1/accounts`），如果支持分页，我们推荐在响应中包含分页信息，但这部分后端尚未统一实现。当前，请使用 `page` 和 `pageSize` 查询参数。

```json
{
  "data": {
    "records": [ ... ], // 当前页的数据列表
    "pagination": {
      "page": 1,
      "pageSize": 20,
      "total_records": 150,
      "total_pages": 8
    }
  },
  "error": ""
}
```
*注意：上述 `pagination` 结构为推荐实现，具体接口的返回请以实际为准。*

#### 状态值 (Status Fields)

系统中多个实体（如任务、记录、账户）都包含 `status` 字段。为了前端能正确展示，每个相关模块都会对可能的状态值进行说明。

## 环境搭建
// ... existing code ...
// ... existing code ...
  "error": ""
}
```

### 用户管理 (Users)

#### `POST /api/v1/users`
// ... existing code ...
// ... existing code ...
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-01T10:00:00Z"
  },
  "error": ""
}
```

#### `GET /api/v1/users` (待开发)
**功能**: 分页获取用户列表。需要管理员权限。

#### `GET /api/v1/users/{id}` (待开发)
**功能**: 根据 ID 获取单个用户详情。

#### `PUT /api/v1/users/{id}` (待开发)
**功能**: 根据 ID 更新用户信息。

#### `DELETE /api/v1/users/{id}` (待开发)
**功能**: 根据 ID 删除一个用户。

### 云账号管理 (Accounts)

云账号指阿里云等邮件服务提供商的账号。
// ... existing code ...
  "access_key_id": "LTAI5txxxxxxxxxxxxxx",
  "access_key_secret": "P1aXxxxxxxxxxxxxxxxxxxxxxx",
  "daily_send_limit": 5000
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Aliyun Account 1",
    "access_key_id": "LTAI5txxxxxxxxxxxxxx",
    "domain": "mail.example.com",
    "daily_send_limit": 5000,
    "status": "active",
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-01T10:00:00Z"
  },
  "error": ""
}
```

#### `GET /api/v1/accounts`
**功能**: 获取所有已配置的云账号列表。

**成功响应 (Success Response):**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Aliyun Account 1",
      "access_key_id": "LTAI5txxxxxxxxxxxxxx",
      "domain": "mail.example.com",
      "daily_send_limit": 5000,
      "status": "active",
      "created_at": "2023-11-01T10:00:00Z",
      "updated_at": "2023-11-01T10:00:00Z"
    },
    {
      "id": 2,
      "name": "Backup Account",
      "access_key_id": "LTAI5tyyyyyyyyyyyyyy",
      "domain": "backup.example.com",
      "daily_send_limit": 1000,
      "status": "inactive",
      "created_at": "2023-11-02T11:00:00Z",
      "updated_at": "2023-11-02T11:00:00Z"
    }
  ],
  "error": ""
}
```

#### `GET /api/v1/accounts/{id}`
**功能**: 根据 ID 获取单个云账号详情。

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Aliyun Account 1",
    "access_key_id": "LTAI5txxxxxxxxxxxxxx",
    "domain": "mail.example.com",
    "daily_send_limit": 5000,
    "status": "active",
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-01T10:00:00Z"
  },
  "error": ""
}
```

#### `PUT /api/v1/accounts/{id}`
**功能**: 根据 ID 更新单个云账号信息。

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Aliyun Account 1 (Updated)",
    "access_key_id": "LTAI5txxxxxxxxxxxxxx",
    "domain": "mail.example.com",
    "daily_send_limit": 6000,
    "status": "active",
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-11-05T14:30:00Z"
  },
  "error": ""
}
```

#### `DELETE /api/v1/accounts/{id}`
**功能**: 根据 ID 删除一个云账号。

**成功响应 (Success Response):**
```json
{
  "data": {
    "message": "Account deleted successfully"
  },
  "error": ""
}
```

### 发件人管理 (Senders)

发件人是发送邮件时"From"字段的实体，它必须关联到一个云账号上。
// ... existing code ...
  "role": "Promotion",
  "contact_info": "marketing@example.com"
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "name": "Marketing Team",
    "role": "Promotion",
    "contact_info": "marketing@example.com",
    "created_at": "2023-11-03T10:00:00Z",
    "updated_at": "2023-11-03T10:00:00Z"
  },
  "error": ""
}
```

#### `POST /api/v1/senders/{senderId}/accounts/{accountId}`
**功能**: 将一个发件人实体与云账号关联，创建一个可用的发送地址。
// ... existing code ...
  "email_address": "noreply@example.com",
  "daily_send_limit": 1000
}
```

**成功响应 (Success Response):**
```json
{
  "data": {
    "id": 1,
    "account_id": 1,
    "sender_id": 1,
    "email_address": "noreply@example.com",
    "weight": 100,
    "daily_send_limit": 1000,
    "status": "active",
    "created_at": "2023-11-04T10:00:00Z",
    "updated_at": "2023-11-04T10:00:00Z"
  },
  "error": ""
}
```

#### `GET /api/v1/senders` (待开发)
**功能**: 获取所有发件人实体列表。

#### `GET /api/v1/senders/{id}` (待开发)
**功能**: 获取单个发件人实体详情。

#### `GET /api/v1/senders/{senderId}/accounts` (待开发)
**功能**: 获取一个发件人实体下关联的所有发送地址列表。

### 收件人管理 (Recipients)

// ... existing code ...
// ... existing code ...
  "task_name": "Q4 Promotion",
  "recipient_group_id": 5,
  "template_id": 10,
// ... existing code ...
// ... existing code ...
    "message": "Email task created and scheduled for dispatch",
    "task_id": 123
  },
  "error": ""
}
```

#### 任务状态说明 (Task Status Explanation)
一个邮件任务在系统中会经历以下几种状态，前端可以根据不同状态展示不同的 UI 提示：
-   **`pending`**: 待处理。任务已创建，等待调度系统派发。
-   **`dispatching`**: 分发中。调度器正在处理该任务，计算收件人列表，并生成独立的邮件发送作业。
-   **`sending`**: 发送中。邮件发送作业已被推送到队列，后台 Worker 正在并发发送邮件。
-   **`completed`**: 已完成。所有邮件发送作业都已执行完毕。
-   **`failed`**: 失败。任务在分发或处理过程中遇到严重错误导致无法继续。

### 数据统计 (Statistics)

#### `GET /api/v1/statistics`
// ... existing code ...
// ... existing code ...
    "task_name": "Q4 Promotion",
    "status": "completed",
    "total_recipients": 5000,
    "open_count": 1500,
    "open_rate": 0.3,
    // ... more summary fields
  },
  "error": ""
}
```

#### `GET /api/v1/tasks/{id}/records`
**功能**: 获取指定任务下所有邮件的详细发送记录。 

**查询参数 (Query Params)**: `page` (页码), `pageSize` (每页数量)。

**成功响应 (Success Response):**
```json
{
  "data": {
    "records": [
      {
        "id": 1001,
        "task_id": 123,
        "account_sender_id": 5,
        "recipient_email": "user1@example.com",
        "subject": "Special Offer Inside!",
        "status": "delivered",
        "error_message": null,
        "sent_at": "2025-01-01T12:05:10Z",
        "last_status_update_at": "2025-01-01T12:05:15Z"
      },
      {
        "id": 1002,
        "task_id": 123,
        "account_sender_id": 5,
        "recipient_email": "user2@example.com",
        "subject": "Special Offer Inside!",
        "status": "failed",
        "error_message": "Invalid recipient email address",
        "sent_at": "2025-01-01T12:05:11Z",
        "last_status_update_at": "2025-01-01T12:05:11Z"
      },
      {
        "id": 1003,
        "task_id": 123,
        "account_sender_id": 6,
        "recipient_email": "user3@example.com",
        "subject": "Special Offer Inside!",
        "status": "bounce",
        "error_message": "Recipient inbox is full",
        "sent_at": "2025-01-01T12:05:12Z",
        "last_status_update_at": "2025-01-01T12:10:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 20,
      "total_records": 5000,
      "total_pages": 250
    }
  },
  "error": ""
}
```

#### 记录状态说明 (Record Status Explanation)
每一封邮件的发送记录（Record）都有自己的状态，这些状态由云服务商回调更新：
-   **`pending`**: 排队中，等待发送。
-   **`sending`**: 发送中，已交给云服务商。
-   **`sent`**: 发送成功，对方邮件服务器已接收。
-   **`failed`**: 发送失败，发送请求阶段就失败了（如格式错误）。
-   **`delivered`**: 投递成功，邮件已成功放入收件人邮箱。
-   **`opened`**: 已打开。
-   **`clicked`**: 已点击邮件中的链接。
-   **`bounce`**: 邮件被弹回，无法送达（如邮箱不存在、邮箱已满）。

</rewritten_file>

