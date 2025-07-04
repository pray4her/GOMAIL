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
    - [x] **性能优化：解决 N+1 查询问题**
        - **问题**: 分析发现，每个邮件发送 Worker 在处理作业时，都会独立查询数据库以获取发件人（`AccountSender`）、账户（`Account`）等信息。当一个任务包含数万封邮件时，这会导致数万次重复的数据库查询，形成典型的 N+1 性能瓶颈，可能压垮数据库。
        - **解决方案**:
            - **责任转移**: 将查询发件人信息的责任从**执行者 (Worker)** 上移至**分发者 (Dispatcher)**。
            - **丰富作业载荷**: `TaskDispatcherService` 现在会在分发任务时，一次性获取该任务所需的所有发件人完整信息（包括账户凭证）。
            - **创建 `EmailJobPayload`**: 将包含完整上下文（收件人、主题、正文、发件人详情）的 `EmailJobPayload` 结构序列化为 JSON，并推送到 `email:sending` 队列。
            - **简化 Worker**: `EmailWorkerService` 从队列中获取作业后，直接使用载荷中的数据，不再访问数据库查询发件人，从根本上消除了 N+1 问题。
    - [x] **SDK 封装**: 在 `pkg/aliyun` 中封装了阿里云邮件推送 SDK。
    - [x] **数据层**: 实现了 `email_send_records` 表的 `repository`，用于记录每一次邮件发送。
    - [x] **配置集成**: 在 `config.yaml` 中添加了阿里云及 Worker 并发数相关配置。

9.  **模块G：企业级收件人分群管理 (Enterprise Recipient Segmentation)**
    - [x] **核心重构: 从"选择"到"分群"**
        - **问题**: 原有通过收件人ID数组创建任务的方式，在面对海量收件人时（如百万级），存在API性能瓶颈、前端交互困难、且无法按用户属性（如国家、注册时间）动态筛选的问题。
        - **解决方案**: 引入"收件人分群" (`Recipient Groups`) 的概念。管理员可以预先定义一组动态规则（例如：`country = 'USA' AND last_opened_days > 30`），系统会在发送时实时计算出符合条件的收件人列表。
    - [x] **架构升级: 集成 Elasticsearch 实现高性能筛选**
        - **挑战**: 在千万级数据量下，仅依赖 PostgreSQL 进行多维度动态规则筛选，性能无法接受。
        - **新架构**:
            - **数据源分离**: PostgreSQL 继续作为收件人数据的唯一"事实来源"(Source of Truth)。Elasticsearch 作为专门的收件人"查询与筛选引擎"。
            - **实时数据同步**: 搭建了 `PostgreSQL -> Redis Queue -> Elasticsearch` 的异步数据同步管道。`SyncService` 消费队列，将数据库中的变更实时索引到 ES。
            - **毫秒级查询**: `RecipientGroupRepository` 的核心查询逻辑被重构，不再查询 PG，而是将分群规则动态翻译成 ES 查询语句，实现了毫秒级的复杂条件筛选。
    - [x] **服务与接口更新**
        - **分群管理**: 新增 `RecipientGroupService` 和 `RecipientGroupHandler`，提供完整的 `/api/v1/recipient-groups` CRUD API 用于管理分群及其规则。
        - **任务创建流程改造**: 创建任务接口 (`POST /api/v1/tasks`) 不再接收庞大的 `recipient_ids` 数组，而是接收轻量的 `recipient_group_id`。
        - **即时解析 (Just-in-Time)**: `TaskDispatcherService` 在任务分发的最后一刻，才调用分群服务解析出最终的收件人列表，确保了数据的最新鲜。
    - [x] **数据库迁移**
        - **新增**: 创建了 `recipient_groups` 和 `recipient_group_rules` 表。
        - **废弃**: 移除了旧的 `email_task_recipients` 中间表，彻底解耦了任务和收件人的静态绑定。

10. **模块H：用户认证与授权 (User Auth & Permissions)**
    - [x] **认证体系**: 实现了完整的 JWT (JSON Web Token) 用户认证系统。
        - **登录接口**: `POST /api/v1/login` 用于验证用户凭据并签发 Token。
        - **认证中间件**: `internal/middleware/auth_middleware.go` 会检查所有受保护路由的 `Authorization: Bearer <token>` 请求头。
    - [x] **用户管理**: 新增独立的用户管理模块。
        - **数据库**: 新建 `users` 表来存储用户信息，与 `accounts` (阿里云账号) 解耦。
        - **密码安全**: 使用 `bcrypt` 库对用户密码进行安全的哈希存储和比对。
        - **用户接口**: `POST /api/v1/users` (受保护) 用于创建新用户。
    - [x] **权限控制**: 实现了精细化的操作权限管理。
        - **数据库**: 新建 `user_permissions` 表，用于关联用户 (`users`) 和其有权使用的发件人 (`account_senders`)。
        - **服务层检查**: 在创建邮件任务等核心操作前，`EmailTaskService` 会严格校验当前用户是否有权使用指定的发件人。
    - [x] **操作审计**: 关键操作现在会记录执行者。
        - **数据库**: `email_tasks` 表新增 `created_by_user_id` 字段，自动追踪每个任务的创建者。

11. **核心发送逻辑升级：智能负载均衡 (Intelligent Load Balancing)**
    - [x] **问题**: 原有的任务与发件人一对一绑定机制，无法充分利用企业下的所有发件人资源，缺乏横向扩展能力和高可用性。
    - [x] **解决方案: 新增 `LoadBalancerService`**
        - **解耦**: 彻底解除了 `EmailTask` 与 `AccountSender` 的静态绑定。现在，创建任务时不再需要指定发件人。
        - **动态派发计划**: `TaskDispatcherService` 在处理任务时，会实时调用 `LoadBalancerService` 来生成一个动态的派发计划 (`DispatchPlan`)。
        - **核心算法**: `LoadBalancerService` 会综合考虑以下因素，在毫秒级内计算出最优的分配方案：
            1.  **用户权限**: 仅使用当前用户有权操作的发件人。
            2.  **账户/发件人状态**: 自动过滤掉已禁用（`inactive`）的账号或发件人。
            3.  **每日剩余配额**: 实时查询每个发件人当天的发送量，精确计算其剩余配额。
            4.  **权重分配**: 在账户内部，根据发件人设置的权重比例（`Weight`）来分配邮件量，实现精细化控制。
        - **高可用**: 当某个发件人配额用尽或被禁用时，系统会自动将其从计划中排除，由其他可用资源无缝接管，保证了任务的顺利执行。
        - **横向扩展**: 系统总的发送吞吐量现在等于所有可用发件人的配额总和，可以通过增加发件人配置来线性提升。

### 🚧 待办事项与开发规划 (To-Do & Roadmap)

#### F. 模板管理模块 (Template Management)
- [x] **功能**: 实现模板变量替换和预览功能。
  - `[x]` 实现了基于收件人字段 (`Email`, `FirstName` 等) 和自定义 `metadata` 的动态模板渲染。
  - `[x]` 提供了 `POST /api/v1/templates/:id/preview` 接口用于实时预览模板效果。

#### D. 任务调度模块 (Task Scheduling)
- [x] **核心逻辑**:
  - `[x]` 实现任务的定时发送功能 (基于 `scheduled_at` 字段)。
    - `[x]` 采用 Redis ZSET (有序集合) 实现了一个高效的调度队列。
    - `[x]` `TaskDispatcherService` 现已升级为调度中心，可轮询并派发到期任务。
  - `[ ]` 实现任务的优先级队列处理。

#### C. 邮件发送模块 (Email Sending)
- [ ] **核心算法**:
  - `[x]` 实现基于权重、历史发送量的智能发件人选择算法 (`LoadBalancerService` 已完成)。
  - `[ ] func CalculateSendingRate(...)`: 实现发送频率控制算法。
  - `[ ] func DistributeEmailTasks(...)`: 实现在多个发件人间负载均衡的算法 (`LoadBalancerService` 已完成)。
- [ ] **核心业务**:
  - `[x]` **异步并发处理**: 已基于 Redis List 和并发 Worker 实现。
  - `[ ]` 失败重试机制（如指数退避）。
- [ ] **接口层**:
  - `[x] POST /api/v1/tasks`: 创建批量发送任务 (已实现)。
  - `[ ] POST /api/v1/emails/send`: (可选)保留或重构为更高优先级的单邮件发送。

#### E. 监控统计模块 (Monitoring & Statistics)
- [x] **核心业务**:
  - `[x]` 定时同步阿里云邮件发送状态。
  - `[x]` 处理邮件追踪数据（送达、打开、点击）并更新到数据库。
- [x] **接口层**: 实现以下 API:
  - `[x]` `GET /api/v1/tasks/:id/records`: 获取发送任务下所有邮件的详细状态。
  - `[x]` `GET /api/v1/statistics`: 获取多维度发送统计
    - `[x]` 支持时间范围过滤 (`start_date`, `end_date`)
    - `[x]` 支持账号和发件人过滤 (`account_id`, `account_sender_id`)
    - `[x]` 支持多种分组方式 (`group_by`: day, week, month, sender)
    - `[x]` 提供核心指标的统计摘要与分析，包括：发送数、总打开数、独立打开数、总点击数、独立点击数及其比率。
    - `[ ]` **注意**: 由于当前依赖的阿里云接口版本限制，暂无法获取邮件的 `送达`、`失败`、`弹回` 状态。

#### 系统级功能 (System-Level Features)
- [ ] **安全 (Security)**
  - `[x]` 实现 API 接口的 JWT 认证中间件。
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
    *   Elasticsearch 正在运行

2.  **配置:**
    *   复制 `config.yaml.example` (如果存在) 为 `config.yaml`。
    *   修改 `config.yaml` 文件，填入正确的数据库、Redis、Elasticsearch 等连接信息。

3.  **首次运行：创建 Elasticsearch 索引**
    *   应用启动时会自动检查名为 `recipients` 的索引是否存在。如果不存在，它将使用预定义的优化映射（Mapping）自动创建该索引。
    *   如果需要手动创建或自定义映射，请参考 `pkg/elasticsearch/client.go` 中的 `mapping` 变量。

4.  **安装依赖:**
    ```bash
    go mod tidy
    ```

5.  **运行数据库迁移:**
    *   首先需要安装 migrate 工具: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
    *   然后执行迁移: `migrate -database "postgres://user:password@localhost:5432/dbname?sslmode=disable" -path migrations up`
      (请将连接字符串替换为您的实际配置)

6.  **启动服务:**
    ```bash
    go run cmd/main.go
    ```
    服务将默认在 `:8080` 端口启动。

7.  **首次运行与API测试:**
    a. **创建第一个用户**: 由于创建用户的接口本身是受保护的，您需要通过**直连数据库**或**临时修改代码**的方式创建第一个用户。
    b. **登录**: 使用您创建的用户名和密码，调用 `POST http://localhost:8080/api/v1/login` 来获取 JWT Token。
    c. **调用受保护接口**: 在请求其他API时，请在请求头中加入 `Authorization` 字段，值为 `Bearer <YOUR_TOKEN>`。
    d. **授予权限**: 若要使用发件人创建任务，需在 `user_permissions` 表中手动插入记录，将 `user_id` 和 `account_sender_id` 进行关联。 