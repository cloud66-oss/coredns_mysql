module github.com/cloud66-oss/coredns_mysql

go 1.16

require (
	github.com/coredns/caddy v1.1.0
	github.com/coredns/coredns v1.8.4
	github.com/go-sql-driver/mysql v1.6.0
	github.com/miekg/dns v1.1.42
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
)

replace github.com/cloud66-oss/coredns_mysql => github.com/chris-edwards-pub/coredns_mysql latest
