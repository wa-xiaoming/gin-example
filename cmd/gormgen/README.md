## 数据库 DAO 和 Model 生成工具

### 简介

本工具用于自动化生成数据库的数据访问对象（DAO）和模型（Model），简化数据库操作的编码工作，并提高开发效率。

### 使用指南

要获取工具的帮助信息，运行以下命令：

```shell
go run cmd/gormgen/main.go -h
```

这将显示工具的使用方法和选项。

#### 命令行选项

- `-dsn`：数据库连接字符串（DSN）。详细信息请参考 [GORM 连接数据库文档](https://gorm.io/docs/connecting_to_the_database.html) 。

- `-tables`：指定要生成代码的表名。如果留空，则为所有表生成代码。

### 示例

以下是如何使用该工具为特定表生成 DAO 和 Model 的示例：

```shell
# 为 `order` 表生成代码
go run cmd/gormgen/main.go -dsn "root:123456789@tcp(127.0.0.1:3306)/gin_example?charset=utf8mb4&parseTime=True&loc=Local" -tables "order"

# 为 `order` 和 `user` 表生成代码
go run cmd/gormgen/main.go -dsn "root:123456789@tcp(127.0.0.1:3306)/gin_example?charset=utf8mb4&parseTime=True&loc=Local" -tables "order,user"

# 为所有表生成代码
go run cmd/gormgen/main.go -dsn "root:123456789@tcp(127.0.0.1:3306)/gin_example?charset=utf8mb4&parseTime=True&loc=Local" -tables ""
```

### 注意事项

- 确保在执行命令时，你位于项目的根目录下。
- 替换 DSN 字符串中的用户名、密码、数据库地址和名称等信息为你的实际数据库配置。
- 如果你想要为所有表生成代码，可以省略 `-tables` 参数或将其留空。
- 该工具假设你的项目结构和数据库配置已经正确设置。

通过上述优化，文档更加清晰地指导用户如何使用工具，包括命令行选项的详细说明和具体的使用示例。
