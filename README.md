# mysql
MySQL backend for CoreDNS

## Supported Record Types

A, AAAA, CNAME, SOA, TXT, NS, MX, CAA and SRV. This backend doesn't support AXFR requests. It also doesn't support wildcard records yet.

## Setup

Add this as an external plugin in `plugin.cfg` file: 

```
mysql:github.com/cloud66-oss/coredns_mysql
```

then run
 
```shell script
$ go generate
$ go build
```

Add any required modules to CoreDNS code as prompted.

## Configuration

In the Corefile, `mysql` can be configured with the following parameters:

`dsn` DSN for MySQL as per https://github.com/go-sql-driver/mysql examples.
`table_prefix` Prefix for the MySQL tables. Defaults to `coredns_`.
`max_lifetime` Duration (in Golang format) for a SQL connection. Default is 1 minute.
`max_open_connections` Maximum number of open connections to the database server. Default is 10.
`max_idle_connections` Maximum number of idle connections in the database connection pool. Default is 10.
`ttl` Default TTL for records without a specified TTL in seconds. Default is 360 (seconds)

## Database Setup
This plugin doesn't create or migrate database schema for its use yet. To create the database and tables, use the following table structure (note the table name prefix):

```sql
CREATE TABLE `coredns_records` (
    `id` INT NOT NULL AUTO_INCREMENT,
	`zone` VARCHAR(255) NOT NULL,
	`name` VARCHAR(255) NOT NULL,
	`ttl` INT DEFAULT NULL,
	`content` TEXT,
	`record_type` VARCHAR(255) NOT NULL,
	PRIMARY KEY (`id`)
) ENGINE = INNODB AUTO_INCREMENT = 6 DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;
```

## Record setup
Each record served by this plugin, should belong to the zone it is allowed to server by CoreDNS. Here are some examples:

```sql
-- Insert batch #1
INSERT INTO coredns_records (zone, name, ttl, content, record_type) VALUES
('test.', 'foo', 30, '{"ip": "1.1.1.1"}', 'A'),
('test.', 'foo', '60', '{"ip": "1.1.1.0"}', 'A'),
('test.', 'foo', 30, '{"text": "hello"}', 'TXT'),
('test.', 'foo', 30, '{"host" : "foo.test.","priority" : 10}', 'MX');
```

These can be queries using `dig` like this:

```shell script
$ dig A MX foo.test 
```

### Acknowledgements and Credits
This plugin, is inspired by https://github.com/wenerme/coredns-pdsql and https://github.com/arvancloud/redis

### Development 
To develop this plugin further, make sure you can compile CoreDNS locally and get this repo (`go get github.com/cloud66-oss/coredns_mysql`). You can switch the CoreDNS mod file to look for the plugin code locally while you're developing it:

Put `replace github.com/cloud66-oss/coredns_mysql => LOCAL_PATH_TO_THE_SOURCE_CODE` at the end of the `go.mod` file in CoreDNS code. 

Pull requests and bug reports are welcome!

