package coredns_mysql

import (
	"database/sql"
	"os"
	"strconv"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

const (
	defaultTableName          = "coredns_records"
	defaultTtl                = 360
	defaultMaxLifeTime        = 24 * time.Hour
	defaultMaxOpenConnections = 10
	defaultMaxIdleConnections = 10
	defaultZoneUpdateTime     = 1 * time.Minute
	defaultDebug              = false
)

func init() {
	caddy.RegisterPlugin("mysql", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	r, err := mysqlParse(c)
	if err != nil {
		return plugin.Error("mysql", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		r.Next = next
		return r
	})

	return nil
}

func mysqlParse(c *caddy.Controller) (*CoreDNSMySql, error) {
	// Use default get a core dns mysql obj
	mysql := CoreDNSMySql{
		TableName:          defaultTableName,
		MaxLifetime:        defaultMaxLifeTime,
		MaxOpenConnections: defaultMaxOpenConnections,
		MaxIdleConnections: defaultMaxIdleConnections,
		Ttl:                defaultTtl,
		debug:              defaultDebug,
	}
	var err error

	c.Next()
	if c.NextBlock() {
		for {
			switch c.Val() {
			case "dsn":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				mysql.Dsn = c.Val()
			case "table_name":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				mysql.TableName = c.Val()
			case "max_lifetime":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val time.Duration
				val, err = time.ParseDuration(c.Val())
				if err != nil {
					val = defaultMaxLifeTime
				}
				mysql.MaxLifetime = val
			case "max_open_connections":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val int
				val, err = strconv.Atoi(c.Val())
				if err != nil {
					val = defaultMaxOpenConnections
				}
				mysql.MaxOpenConnections = val
			case "max_idle_connections":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val int
				val, err = strconv.Atoi(c.Val())
				if err != nil {
					val = defaultMaxIdleConnections
				}
				mysql.MaxIdleConnections = val
			case "zone_update_interval":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val time.Duration
				val, err = time.ParseDuration(c.Val())
				if err != nil {
					val = defaultZoneUpdateTime
				}
				mysql.zoneUpdateTime = val
			case "ttl":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val int
				val, err = strconv.Atoi(c.Val())
				if err != nil {
					val = defaultTtl
				}
				mysql.Ttl = uint32(val)
			case "debug":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val bool
				val, err = strconv.ParseBool(c.Val())
				if err != nil {
					val = false
				}
				if val {
					mysql.debug = true
				}
			default:
				if c.Val() != "}" {
					return &CoreDNSMySql{}, c.Errf("unknown property '%s'", c.Val())
				}
			}

			if !c.Next() {
				break
			}
		}

	}

	dbConn, err := mysql.getDBConn()
	if err != nil {
		return nil, err
	}

	err = dbConn.Ping()
	if err != nil {
		return nil, err
	}
	mysql.dbConn = dbConn

	return &mysql, nil
}

func (handler *CoreDNSMySql) getDBConn() (*sql.DB, error) {
	db, err := sql.Open("mysql", os.ExpandEnv(handler.Dsn))
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(handler.MaxLifetime)
	db.SetMaxOpenConns(handler.MaxOpenConnections)
	db.SetMaxIdleConns(handler.MaxIdleConnections)

	return db, nil
}
