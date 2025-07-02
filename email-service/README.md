# 企业级多账号邮件群发管理系统

本项目是一个基于 Go 语言和阿里云邮件推送服务的企业级邮件群发系统。

## 项目进度追踪

### ✅ 已完成 (Completed)

1.  **项目初始化**
    - [x] 创建了符合 Go 项目最佳实践的目录结构 (`cmd`, `internal`, `pkg`, 等)。
    - [x] 使用 `go mod init` 初始化了 Go 模块。

2.  **数据库设计**
    - [x] 设计了所有核心表的结构 (`accounts`, `senders`, `email_tasks`, 等)。
    - [x] 在 `migrations/` 目录下创建了数据库迁移脚本。
    - [x] 在 `docs/` 目录下生成了 ER 图和数据库设计文档。
    - [x] 使用 `migrate` 工具成功执行了数据库表创建。

3.  **基础架构与核心重构**
    - [x] 使用 `viper` 实现了灵活的配置文件加载 (`config.yaml`)。
    - [x] 使用 `gorm` 建立了数据库连接和初始化逻辑。
    - [x] 使用 `gin` 搭建了 Web 服务器的基础框架。
    - [x] **架构重构**: 全面采用**依赖注入 (Dependency Injection)** 模式重构了 `repository`, `service`, `handler` 层。
        - **优势**: 大幅提升了代码的可测试性、可维护性和灵活性，实现了层与层之间的松散耦合。

4.  **模块A：账号管理 (Account Management)**
    - [x] **Repository 层**: 实现了 `accounts` 表的数据库 CRUD 操作。
    - [x] **Service 层**: 实现了账号管理的业务逻辑。
    - [x] **Handler 层**: 实现了处理账号管理相关 HTTP 请求的 API 端点。
    - [x] **Router**: 将 `/api/v1/accounts` 路由注册到 Web 服务器。

5.  **模块B：发件人管理 (Sender Management)**
    - [x] **数据层**: `senders` 和 `account_senders` 表的 Repository 实现。
    - [x] **业务层**: 发件人信息维护、与账号的关联管理。
    - [x] **接口层**: 实现了发件人及账号关联的 API 端点。
    - [x] **Router**: 将 `/api/v1/senders` 及相关路由注册到 Web 服务器。

6.  **模块F：模板管理 (Template Management)**
    - [x] **数据层/业务层/接口层**: 实现了邮件模板的 CRUD。
    - [x] **Router**: 将 `/api/v1/templates` 路由注册到 Web 服务器。

7.  **模块D：批量任务管理 (Batch Task Management)**
    - [x] **数据层/业务层/接口层**: 实现了批量发送任务的创建与管理。
    - [x] **API 端点**: 实现了 `POST /api/v1/tasks` 接口，用于创建批量发送任务。
    - [x] **数据库**: 为 `email_tasks` 表增加了与发件人的关联。

8.  **模块C：邮件发送与并发处理 (Email Sending & Concurrency)**
    - [x] **架构升级：两阶段消费者模式**
        - **API层 (生产者)**: `EmailTaskService` 负责快速创建任务，并将 `task_id` 推送到专用的 Redis 队列 `tasks:created`，API 响应时间从数十秒降至**毫秒级**。
        - **后台分发层 (消费者/生产者)**: 新增 `TaskDispatcherService`，作为后台 Worker 监听 `tasks:created` 队列。它负责获取任务详情，并将大型任务"分发"成数千个独立的邮件发送作业，推送到 `email:sending` 队列。
        - **后台发送层 (并发消费者)**: `EmailWorkerService` 现在可以根据配置启动**多个并发的 Worker**，共同消费 `email:sending` 队列，将邮件发送的吞吐量提升 **N 倍**（N为配置的并发数）。
    - [x] **SDK 封装**: 在 `pkg/aliyun` 中封装了阿里云邮件推送 SDK。
    - [x] **数据层**: 实现了 `email_send_records` 表的 `repository`，用于记录每一次邮件发送。
    - [x] **配置集成**: 在 `config.yaml` 中添加了阿里云及 Worker 并发数相关配置。

9.  **模块G：收件人管理 (Recipient Management)**
    - [x] **数据库**: 新增了 `recipients` 表，支持自定义元数据，并创建了相应的迁移脚本。
    - [x] **数据层**: `RecipientRepository` 实现了对收件人完整的 CRUD 操作。
    - [x] **业务层**: `RecipientService` 实现了收件人管理的业务逻辑，包括邮箱唯一性校验。
    - [x] **接口层**: `RecipientHandler` 提供了完整的 RESTful API 用于管理收件人。
    - [x] **Router**: 将 `/api/v1/recipients` 路由组注册到 Web 服务器。

### 🚧 待办事项与开发规划 (To-Do & Roadmap)

#### F. 模板管理模块 (Template Management)
- [ ] **功能**: 实现模板变量替换和预览功能。

#### D. 任务调度模块 (Task Scheduling)
- [ ] **核心逻辑**:
  - `[ ]` 实现任务的定时发送功能 (基于 `scheduled_at` 字段)。
  - `[ ]` 实现任务的优先级队列处理。

#### C. 邮件发送模块 (Email Sending)
- [ ] **核心算法**:
  - `[ ] func SelectOptimalSender(...)`: 实现基于权重、历史发送量的智能发件人选择算法。
  - `[ ] func CalculateSendingRate(...)`: 实现发送频率控制算法。
  - `[ ] func DistributeEmailTasks(...)`: 实现在多个发件人间负载均衡的算法。
- [ ] **核心业务**:
  - `[x]` **异步并发处理**: 已基于 Redis List 和并发 Worker 实现。
  - `[ ]` 失败重试机制（如指数退避）。
- [ ] **接口层**:
  - `[x] POST /api/v1/tasks`: 创建批量发送任务 (已实现)。
  - `[ ] POST /api/v1/emails/send`: (可选)保留或重构为更高优先级的单邮件发送。

#### E. 监控统计模块 (Monitoring & Statistics)
- [ ] **核心业务**:
  - `[ ]` 定时同步阿里云邮件发送状态。
  - `[ ]` 处理邮件追踪数据（送达、打开、点击）并更新到数据库。
  - `[ ]` 实现发送结果的回调通知接口。
- [ ] **接口层**: 实现以下 API:
  - `GET /api/v1/emails/tasks/:id`: 获取发送任务状态
  - `GET /api/v1/emails/statistics`: 获取多维度发送统计

#### 系统级功能 (System-Level Features)
- [ ] **安全 (Security)**
  - `[ ]` 实现 API 接口的 JWT 认证中间件。
  - `[ ]` 实现敏感数据（如 AccessKeySecret）的加密存储。
  - `[ ]` 实现 SQL 注入防护 (GORM自带部分防护)。
  - `[ ]` 实现访问日志审计。
- [ ] **监控 (Observability)**
  - `[ ]` 集成 Prometheus, 暴露系统性能指标。
  - `[ ]` 配置 Grafana Dashboard 用于展示监控数据。
- [ ] **代码质量 (Quality)**
  - `[ ]` 编写单元测试和集成测试，目标覆盖率 > 80%。
  - `[ ]` 完善代码注释和文档。
- [ ] **部署 (Deployment)**
  - `[ ]` 编写 `Dockerfile` 用于容器化部署。
  - `[ ]` 编写 Kubernetes 部署和服务文件 (`deployments/`)。
  - `[ ]` 编写详细的部署文档。
- [ ] **文档 (Documentation)**
  - `[ ]` 生成详细的 API 接口文档 (如 Swagger)。

## 如何运行

1.  **确保先决条件已满足:**
    *   Go 1.18+
    *   PostgreSQL 正在运行
    *   Redis 正在运行
    *   RabbitMQ (或 Kafka) 正在运行

2.  **配置:**
    *   复制 `config.yaml.example` (如果存在) 为 `config.yaml`。
    *   修改 `config.yaml` 文件，填入正确的数据库、Redis等连接信息。

3.  **安装依赖:**
    ```bash
    go mod tidy
    ```

4.  **运行数据库迁移:**
    *   首先需要安装 migrate 工具: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
    *   然后执行迁移: `migrate -database "postgres://user:password@localhost:5432/dbname?sslmode=disable" -path migrations up`
      (请将连接字符串替换为您的实际配置)

5.  **启动服务:**
    ```bash
    go run cmd/main.go
    ```
    服务将默认在 `:8080` 端口启动。 