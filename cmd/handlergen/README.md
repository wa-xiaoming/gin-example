## 生成 Handler、Routers 和 Swagger 接口文档工具

### 概述

本工具旨在自动化生成 Handler、Routers 和 Swagger 接口文档，简化开发流程，确保代码的一致性和可维护性。

### 使用方法

要查看工具的详细使用说明，可以执行以下命令：

```shell
go run cmd/handlergen/main.go -h
```

这将显示工具的用法和可用选项。

#### 选项说明

- `-table`：此选项用于指定需要生成代码的数据库表名。

### 示例用法

以下是如何使用该工具生成名为 `admin` 的表的 Handler、Routers 和 Swagger 接口文档的示例：

```shell
# 在项目根目录下执行
go run cmd/handlergen/main.go -table "admin"
```

这行命令会根据指定的 `admin` 表生成相应的 Handler、Routers 和 Swagger 文档，帮助你快速启动和维护项目中的 API 开发。

### 注意事项

- 确保在执行命令时，你位于项目的根目录下。
- 替换 `admin` 为你实际需要生成文档的表名。
- 该工具假设你的项目结构和数据库配置已经正确设置。
