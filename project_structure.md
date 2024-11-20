crowdfunding-backend/
├── cmd/
│   └── main.go                 # 应用程序入口
├── config/
│   └── config.go               # 配置文件（从.env加载配置）
├── .env                        # 环境变量配置文件
├── .env.example                # 环境变量示例文件
├── internal/
│   ├── api/
│   │   ├── user/
│   │   │   ├── auth.go         # 用户认证相关处理器
│   │   │   ├── profile.go      # 用户资料相关处理器
│   │   │   └── verification.go # 用户验证相关处理器
│   │   ├── project/
│   │   │   ├── create.go       # 创建项目处理器
│   │   │   ├── review.go       # 项目审核处理器
│   │   │   ├── list.go         # 项目列表处理器
│   │   │   ├── detail.go       # 项目详情处理器
│   │   │   ├── update.go       # 项目更新处理器
│   │   │   ├── goal.go         # 项目目标金额处理器
│   │   │   └── image.go        # 项目图片处理器
│   │   ├── order/
│   │   │   ├── create.go       # 创建订单处理器
│   │   │   ├── list.go         # 订单列表处理器
│   │   │   ├── detail.go       # 订单详情处理器
│   │   │   ├── pay.go          # 订单支付处理器
│   │   │   └── shipping.go     # 订单发货状态处理器
│   │   ├── post/
│   │   │   ├── create.go       # 创建帖子处理器
│   │   │   ├── list.go         # 帖子列表处理器
│   │   │   ├── detail.go       # 帖子详情处理器
│   │   │   ├── like.go         # 帖子点赞处理器
│   │   │   ├── comment.go      # 帖子评论处理器
│   │   │   └── image.go        # 帖子图片处理器
│   │   ├── admin/
│   │   │   ├── project.go      # 管理员项目管理处理器
│   │   │   ├── order.go        # 管理员订单管理处理器
│   │   │   ├── refund.go       # 管理员退款管理处理器
│   │   │   └── system.go       # 系统管理处理器
│   │   └── notification/
│   │       ├── create.go       # 创建通知处理器
│   │       └── list.go         # 通知列表处理器
│   ├── middleware/             # 中间件
│   │   ├── auth.go             # 认证中间件
│   │   ├── admin_auth.go       # 管理员认证中间件
│   │   └── cors.go             # CORS中间件
│   ├── model/
│   │   ├── user.go
│   │   ├── project.go
│   │   ├── project_goal.go
│   │   ├── project_goal_image.go
│   │   ├── project_image.go
│   │   ├── order.go
│   │   ├── refund.go
│   │   ├── transaction.go
│   │   ├── post.go
│   │   ├── comment.go
│   │   ├── notification.go
│   │   └── backup.go
│   ├── repository/
│   │   ├── mysql/
│   │   │   ├── user_repo.go
│   │   │   ├── project_repo.go
│   │   │   ├── project_goal_repo.go
│   │   │   ├── order_repo.go
│   │   │   ├── refund_repo.go
│   │   │   ├── post_repo.go
│   │   │   ├── comment_repo.go
│   │   │   ├── notification_repo.go
│   │   │   └── backup_repo.go
│   │   └── interfaces/
│   │       ├── user_repo.go
│   │       ├── project_repo.go
│   │       ├── project_goal_repo.go
│   │       ├── order_repo.go
│   │       ├── refund_repo.go
│   │       ├── post_repo.go
│   │       ├── comment_repo.go
│   │       ├── notification_repo.go
│   │       └── backup_repo.go
│   ├── service/
│   │   ├── user_service.go
│   │   ├── project_service.go
│   │   ├── order_service.go
│   │   ├── refund_service.go
│   │   ├── post_service.go
│   │   ├── comment_service.go
│   │   ├── admin_service.go
│   │   ├── notification_service.go
│   │   └── backup_service.go
│   └── util/
│       ├── jwt.go
│       ├── image.go
│       ├── pagination.go
│       └── db.go
├── pkg/
│   └── payment/
│       └── gateway.go
├── migrations/
│   └── ... (迁移文件)
├── static/
│   ├── avatars/
│   ├── project_images/
│   └── post_images/
├── tests/
│   ├── unit/
│   └── integration/
├── docs/
│   ├── api_documentation.md
│   └── database_schema.md
├── scripts/
├── Dockerfile
├── .gitignore
├── go.mod
└── go.sum