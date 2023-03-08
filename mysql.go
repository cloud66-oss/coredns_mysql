package coredns_mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/miekg/dns"
)

func (handler *CoreDNSMySql) dbQuery(zone, host, qType string) ([]*Record, error) {
	sql := fmt.Sprintf("SELECT host, zone, type, data, ttl, "+
		"priority, weight, port, target, flag, tag, "+
		"primary_ns, resp_person, serial, refresh, retry, expire, minimum, "+
		"remark	FROM %s WHERE zone = ? AND host = ? AND type = ?", handler.TableName)
	results, err := handler.dbConn.Query(sql, zone, host, qType)
	if err != nil {
		return nil, err
	}
	records, err := handler.getRecordsFromQueryResults(results)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (handler *CoreDNSMySql) findRecord(zone string, name string, qType string) ([]*Record, int, error) {
	// 如果不能匹配，则转给下一个 coredns 插件
	if zone == "" {
		clog.Debug("coredns-mysql: Not fount matched zone, retrun request to next plugin")
		// code, err := plugin.NextOrFailure(handler.Name(), handler.Next, ctx, w, r)
		return nil, RcodeNextPlugin, nil
	}

	// 处理确定查询的是域本身？亦或是域名
	query := "@"
	if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}
	// 从数据库中查询该记录
	clog.Debug("coredns-mysql: Use zone ", zone, " name ", name, " type ", qType, " to query db ")
	records, err := handler.dbQuery(zone, query, qType)
	if err != nil {
		clog.Debug("coredns-mysql: DB query error ", err)
		return nil, dns.RcodeServerFailure, err
	}
	clog.Debug("coredns-mysql: DB query records are ", records)

	// 如果DB中没有该域名对应查询类型的记录，则尝试查询该域名的所有类型的记录
	// 比如: 可能该域名本事其实是一个CNAME记录或者MX等等，
	if len(records) == 0 {
		clog.Debug("coredns-mysql: Query not have record, query type is ", qType)
		// 判断查询类型是否为 A 或 AAAA，如果是则对该域名的CNAME记录进行查询
		switch qType {
		case RecordType.A, RecordType.AAAA:
			// 查询 CNAME 类型的记录，看是否存在
			records, err = handler.dbQuery(zone, query, RecordType.CNAME)
			clog.Debug("coredns-mysql: Query CNAME records are ", records)
			if err != nil {
				clog.Debug("coredns-mysql: DB query error ", err)
				return nil, dns.RcodeServerFailure, err
			}
			// 如果存在 CNAME 记录，则查询 CNAME 指向的域名的 A 或 AAAA 类型的记录
			if len(records) != 0 {
				clog.Debug("coredns-mysql: Recursive call findrecord method ", records)
				for _, record := range records {
					qZone := plugin.Zones(handler.zones).Matches(record.Data)
					recordsIP, _, err := handler.findRecord(qZone, record.Data, qType)
					if err != nil {
						return nil, 0, err
					}
					records = append(records, recordsIP...)
				}
			}
		}
	}

	return records, 0, nil
}

func (handler *CoreDNSMySql) loadZones() error {
	dbConn := handler.dbConn
	result, err := dbConn.Query("SELECT DISTINCT zone FROM " + handler.TableName)
	if err != nil {
		return err
	}

	var zone string
	zones := make([]string, 0)
	for result.Next() {
		err = result.Scan(&zone)
		if err != nil {
			return err
		}

		zones = append(zones, zone)
	}
	handler.lastZoneUpdate = time.Now()
	handler.zones = zones

	return nil
}

func (handler *CoreDNSMySql) getRecordsFromQueryResults(results *sql.Rows) (records []*Record, err error) {
	var (
		rHost string
		rZone string
		rType string
		rData string
		rTTL  uint32

		rPriority uint16
		rWeight   uint16
		rPort     uint16
		rTarget   string
		rFlag     uint8
		rTag      string

		rPrimaryNS  string
		rRespPerson string
		rSerial     uint32
		rRefresh    uint32
		rRetry      uint32
		rExpire     uint32
		rMinimum    uint32

		remark string
	)
	for results.Next() {
		err = results.Scan(
			&rHost, &rZone, &rType, &rData, &rTTL,
			&rPriority, &rWeight, &rPort, &rTarget, &rFlag, &rTag,
			&rPrimaryNS, &rRespPerson, &rSerial, &rRefresh, &rRetry, &rExpire, &rMinimum,
			&remark,
		)
		if err != nil {
			return
		}

		record := &Record{
			Host: rHost,
			Zone: rZone,
			Type: rType,
			Data: rData,
			TTL:  rTTL,

			Priority: rPriority,
			Weight:   rWeight,
			Port:     rPort,
			Target:   rTarget,

			PrimaryNS:  rPrimaryNS,
			RespPerson: rRespPerson,
			Serial:     rSerial,
			Refresh:    rRefresh,
			Retry:      rRetry,
			Expire:     rExpire,
			Minimum:    rMinimum,

			handler: handler,
			remark:  remark,
		}

		records = append(records, record)
	}
	return
}
