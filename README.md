## 项目说明
    基于Gin框架构建的WebAPI服务项目，采用了模块化的设计，具有清晰的分层架构。可以作为GoWeb服务模板，具备了生产环境所需的大部分特性，代码质量较高，架构清晰，非常适合学习和在此基础上开发实际项目。

## 项目结构
    gin-example/
    ├── cmd/                 # 命令行工具
    ├── configs/             # 配置文件（支持多环境）
    ├── docs/                # 文档
    ├── internal/            # 内部模块
    │   ├── api/             # API 控制器
    │   ├── pkg/             # 核心组件库
    │   ├── repository/      # 数据访问层
    │   └── router/          # 路由配置
    ├── logs/                # 日志文件
    ├── scripts/             # 脚本文件
    ├── main.go              # 程序入口
    ├── go.mod               # 依赖管理
    └── README.md            # 项目说明

## 项目介绍
### 1.核心特性
1. 多环境配置支持：
   - 支持 dev、fat、uat、pro 四种环境
   - 使用 TOML 格式配置文件
   - 可通过命令行参数 -env 指定环境
2. 数据库设计：
   - 使用 GORM 作为 ORM 框架
   - 支持读写分离（主从数据库）
   - 集成连接池配置
   - 自动生成 DAO 层代码
3. 缓存系统：
   - 多级缓存架构（本地缓存 + Redis）
   - 自动缓存命中和更新策略
   - 防止缓存穿透机制
4. 中间件系统：
   - CORS 跨域支持
   - Swagger API 文档
   - Prometheus 监控指标
   - PProf 性能分析
   - JWT 认证
   - 限流和熔断
5. 日志系统：
   - 使用 Uber zap 日志库
   - 结构化日志记录
   - 链路追踪支持
6. 服务治理：
   - 健康检查接口
   - 优雅停机
   - Etcd 服务注册与发现
   - 告警通知机制

### 核心模块分析
1. 核心框架 (internal/pkg/core)
   - 封装了 Gin 框架，提供统一的上下文管理
   - 集成错误处理、日志记录、指标收集等功能
   - 支持链路追踪和监控指标
2. 数据访问层 (internal/repository)
   - MySQL 数据库访问（支持读写分离）
   - Redis 缓存访问
   - MongoDB 支持（预留）
3. 缓存系统 (internal/pkg/cache)
   - 多级缓存实现（本地 + Redis）
   - 自动缓存管理和更新策略
4. API 层 (internal/api)
   - 管理员管理接口（增删改查）
   - 系统健康检查
   - 认证接口
5. 配置管理 (configs)
   - 多环境配置支持
   - TOML 格式配置文件
   - 配置热加载

### 技术亮点
1. 优雅的架构设计：
   - 清晰的分层架构，职责分离
   - 高内聚低耦合的模块设计
2. 生产就绪特性：
   - 完善的错误处理和日志记录
   - 健康检查和监控指标
   - 优雅停机和服务注册
3. 高性能设计：
   - 多级缓存架构
   - 连接池管理
   - 读写分离
4. 可观测性：
   - 链路追踪
   - 性能监控
   - 指标收集

## 启动
```
1.初始化数据库（scripts/README.md）
2.启动etcd服务
3.go run main.go -env fat  

// -env 默认为 fat
// -env dev 表示为本地开发环境，使用的配置信息为：configs/dev_configs.toml
// -env fat 表示为测试环境，使用的配置信息为：configs/fat_configs.toml
// -env uat 表示为预上线环境，使用的配置信息为：configs/uat_configs.toml
// -env pro 表示为正式环境，使用的配置信息为：configs/pro_configs.toml
```

## 接口文档

- 接口文档：http://127.0.0.1:9999/swagger/index.html
- 心跳检测：http://127.0.0.1:9999/system/health


