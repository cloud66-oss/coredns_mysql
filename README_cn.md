# MySQL

使用mysql作为coredns的后端

[English](./README.md) | 中文

## Name
mysql - MySQL backend for CoreDNS

## Description

该插件使用MySQL作为存储DNS记录的后端，数据库存储的数据可以为coredns提供服务。后端使用简单的单个表数据结构，可以由其他系统共享，从DNS服务器添加和删除记录。由于插件中没有存储的状态，因此可以通过调整数据库连接的个数来实现性能的扩缩容。


## Syntax
```
mysql {
    dsn DSN
    [table_name TABLE_NAME]
    [max_lifetime MAX_LIFETIME]
    [max_open_connections MAX_OPEN_CONNECTIONS]
    [max_idle_connections MAX_IDLE_CONNECTIONS]
    [ttl DEFAULT_TTL]
    [zone_update_interval ZONE_UPDATE_INTERVAL]
    [debug true|false]
}
```

- `dsn` 根据 https://github.com/go-sql-driver/mysql 示例，用于mySQL的DSN。您可以在DSN中使用`$ ENV_NAME`格式，并将其替换为环境变量值。
- `table_name` MySQL 表名. 默认值为 `coredns_records`.
- `max_lifetime` mysql连接池的最大时长(golang的时间格式字符串，如 `3600s`). 默认为 `24h`.
- `max_open_connections` 与DB建立最大连接的个数. 默认为 `10`.
- `max_idle_connections` DB连接池中最大空闲连接个数.  默认为 `5`.
- `ttl` 如果记录没有设置TTL，则使用此值. 默认值为 `360s`.
- `zone_update_interval` 从数据库加载所有域的最大时间间隔，此选项用于提升性能. 默认值为 `1m`.
- `debug` coredns-mysql插件本身的调试功能,可以在关键步骤打印一些debug日志. 默认值为 `false`.

## Supported Record Types

A，AAAA，CNAME，SOA，TXT，NS，MX，CAA和SRV。~~也支持通配符记录~~。此后端不支持AXFR请求。

## Setup (as an external plugin)

将其添加到 `plugin.cfg` 中：

```
mysql:github.com/snail2sky/coredns_mysql
```

运行
 
```shell script
go generate
go get
go build
```

按照提示，将任何必需的模块添加到Coredns代码中

## Database Setup
该插件尚未创建或迁移数据库架构以供其使用。要创建数据库和表，请使用下表结构（请注意表名跟配置文件相对应）：

```sql
CREATE TABLE coredns_records (
  id int(11) NOT NULL AUTO_INCREMENT,
  host varchar(128) NOT NULL,
  zone varchar(512) NOT NULL,
  type varchar(10) NOT NULL DEFAULT 'A',
  data varchar(1024) DEFAULT '',
  ttl int(8) DEFAULT 30,
  
  priority int(4) DEFAULT 0,
  weight int(4) DEFAULT 1,
  port int(5) DEFAULT 0,
  target varchar(256) DEFAULT '',
  flag int(2) DEFAULT 0,
  tag varchar(64) DEFAULT '',

  primary_ns varchar(100) DEFAULT '',
  resp_person varchar(100) DEFAULT '',

  serial int(11) DEFAULT 0,
  refresh int(11) DEFAULT 0,
  retry int(11) DEFAULT 0,
  expire int(11) DEFAULT 0,
  minimum int(11) DEFAULT 0,
  
  online int(2) DEFAULT 0,
  remark varchar(64) DEFAULT '',
  
  PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8;
```

## Record setup
该插件提供的每个记录都应属于核心允许使用的区域。这里有些例子：

```sql
-- Insert SOA record
INSERT INTO coredns_records (host, zone, type, ttl, primary_ns, resp_person, serial, refresh, retry, expire, minimum) VALUES
('@', 'example.org.', 'SOA', 86400, 'ns1.example.org.', 'root.example.org.', 1, 3600, 300, 86400, 300);

-- Insert NS record
INSERT INTO coredns_records (host, zone, type, data, ttl) VALUES
('@', 'example.org.', 'NS', 'ns1.example.org.', 3600);

-- Insert MX records
INSERT INTO coredns_records (host, zone, type, data, ttl, priority) VALUES
('@', 'example.org.', 'MX', 'mx1.svc.tiger.', 3600, 10),
('@', 'example.org.', 'MX', 'mx2.svc.tiger.', 3600, 20);

-- Insert CNAME record
INSERT INTO coredns_records (host, zone, type, data, ttl) VALUES
('www', 'example.org.', 'CNAME', 'web.example.org.', 240);

-- Insert A record
INSERT INTO coredns_records (host, zone, type, data, ttl) VALUES
('web', 'example.org.', 'A', '6.7.8.9', 60);

-- Insert A record
INSERT INTO coredns_records (host, zone, type, data, ttl) VALUES
('ns1', 'example.org.', 'A', '1.2.3.4', 120);

-- Insert AAAA record
INSERT INTO coredns_records (host, zone, type, data, ttl) VALUES
('ns1', 'example.org.', 'AAAA', '2402:4e00:1020:1404:0:9227:71ab:2b74', 240);

-- Insert A record
INSERT INTO coredns_records (host, zone, type, data, ttl) VALUES
('@', 'example.org.', 'A', '5.6.7.8', 60);

-- Insert TXT record
INSERT INTO coredns_records (host, zone, type, data, ttl) VALUES
('@', 'example.org.', 'TXT', 'hello world!', 120);

```

可以使用 dig 命令进行查询：

```bash
# query SOA record
dig example.org SOA

# query NS record
dig example.org NS

# query MX record
dig example.org MX

# query CNAME record
dig www.snail2sky.live CNAME
# or 
dig www.snail2sky.live

# query A and AAAA record
dig ns1.example.org A
dig ns1.example.org AAAA

# query TXT record
dig example.org TXT
```
- 示例Corefile配置文件
```corefile
example.org.:53 {
    mysql {
        dsn username:password@tcp(127.0.0.1:3306)/db_name
        table_name coredns_records
        max_lifetime 24h
        max_open_connections 8
        max_idle_connections 4
        zone_update_interval 60s
        # debug true
    }
}
.:53 {
    forward . 8.8.8.8
}
```

### Acknowledgements and Credits
该插件fork于 https://github.com/cloud66-oss/coredns_mysql.git ,感谢 https://github.com/wenerme/coredns-pdsql 和 https://github.com/arvancloud/redis 灵感

### Development 
要进一步开发此插件，请确保您可以在本地编译Coredns并获取此存储库（`go get github.com/snail2sky/coredns_mysql`）。您可以在开发该文件时切换 coredns mod文件以在本地查找插件代码：

在coredns代码中的 `go.mod` 文件的末尾追加 `replace github.com/snail2sky/coredns_mysql => LOCAL_PATH_TO_THE_SOURCE_CODE`

欢迎大家提PR和ISSUES~
再次感谢

