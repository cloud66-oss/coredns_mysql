# MySQL

MySQL backend for CoreDNS

## Name
mysql - MySQL backend for CoreDNS

## Description

This plugin uses MySQL as a backend to store DNS records. These will then can served by CoreDNS. The backend uses a simple, single table data structure that can be shared by other systems to add and remove records from the DNS server. As there is no state stored in the plugin, the service can be scaled out by spinning multiple instances of CoreDNS backed by the same database.

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

- `dsn` DSN for MySQL as per https://github.com/go-sql-driver/mysql examples. You can use `$ENV_NAME` format in the DSN, and it will be replaced with the environment variable value.
- `table_name` MySQL table name. Defaults to `coredns_records`.
- `max_lifetime` Duration (in Golang format) for a SQL connection. Default is 24 hours.
- `max_open_connections` Maximum number of open connections to the database server. Default is 10.
- `max_idle_connections` Maximum number of idle connections in the database connection pool. Default is 10.
- `ttl` Default TTL for records without a specified TTL in seconds. Default is 360 (seconds)
- `zone_update_interval` Maximum time interval between loading all the zones from the database. Default is 1 minutes.
- `debug` Open coredns-mysql debug model. default false.

## Supported Record Types

A, AAAA, CNAME, SOA, TXT, NS, MX, CAA and SRV.  Wildcard records are supported as well.  This backend doesn't support AXFR requests.

## Setup (as an external plugin)

Add this as an external plugin in `plugin.cfg` file: 

```
mysql:github.com/snail2sky/coredns_mysql
```

then run
 
```shell script
$ go generate
$ go build
```

Add any required modules to CoreDNS code as prompted.

## Database Setup
This plugin doesn't create or migrate database schema for its use yet. To create the database and tables, use the following table structure (note the table name prefix):

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
  
  remark varchar(64) DEFAULT '',
  
  PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8;
```

## Record setup
Each record served by this plugin, should belong to the zone it is allowed to server by CoreDNS. Here are some examples:

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

These can be queries using `dig` like this:

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
- Example Corefile
```corefile
example.org.:53 {
    cache {
        success 65535
        denial 65535
    }
    mysql {
        dsn username:password@tcp(127.0.0.1:3306)/db_name
        table_name coredns_records
        max_lifetime 360000000
        max_open_connections 8
        max_idle_connections 4
        zone_update_interval 60s
        # debug true
    }
}
.:53 {
    cache {
        success 65535
        denial 65535
    }
    forward . 8.8.8.8
}
```

### Acknowledgements and Credits
This plugin, is inspired by https://github.com/cloud66-oss/coredns_mysql.git and https://github.com/wenerme/coredns-pdsql and https://github.com/arvancloud/redis

### Development 
To develop this plugin further, make sure you can compile CoreDNS locally and get this repo (`go get github.com/snail2sky/coredns_mysql`). You can switch the CoreDNS mod file to look for the plugin code locally while you're developing it:

Put `replace github.com/snail2sky/coredns_mysql => LOCAL_PATH_TO_THE_SOURCE_CODE` at the end of the `go.mod` file in CoreDNS code. 

Pull requests and bug reports are welcome!

