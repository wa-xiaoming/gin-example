
## 初始化 MySQL 信息

```sql
-- 1. MySQL 数据库连接信息，配置到 ./configs/fat_configs.toml 中 --
[mysql.read]
addr = '127.0.0.1:3306'
name = 'gin_example'
pass = '123456789'
user = 'root'

[mysql.write]
addr = '127.0.0.1:3306'
name = 'gin_example'
pass = '123456789'
user = 'root'

-- 2. 创建数据表 --
CREATE TABLE `admin` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `username` varchar(32) NOT NULL DEFAULT '' COMMENT '用户名',
  `mobile` varchar(20) NOT NULL DEFAULT '' COMMENT '手机号',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员表';

-- 3. 初始化数据 --
INSERT INTO `admin` (`id`, `username`, `mobile`) VALUES
(1, '张三', '13888888888'),
(2, '李四', '13888888888'),
(3, '赵五', '13888888888');
```
