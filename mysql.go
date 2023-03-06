package coredns_mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
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

func (handler *CoreDNSMySql) findRecord(zone string, name string, qType string) ([]*Record, []*Record, error) {
	// 处理确定查询的是域本身？亦或是域名

	query := "@"
	if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}
	// 以 host, zone, type 对DB进行查询，并且得到记录
	var allExtRecords = make([]*Record, 0)
	records, err := handler.dbQuery(zone, query, qType)
	if err != nil {
		return nil, nil, err
	}

	// 如果DB中没有该域名对应查询类型的记录，则尝试查询该域名的所有类型的记录
	// 比如: 可能该域名本事其实是一个CNAME记录或者MX等等，
	if len(records) == 0 {
		// 判断查询类型是否为 A 或 AAAA，如果是则对该域名的CNAME记录进行查询
		switch qType {
		case RecordType.A, RecordType.AAAA:
			records, err = handler.dbQuery(zone, query, RecordType.CNAME)
			if err != nil {
				return nil, nil, err
			}
			if len(records) != 0 {
				for _, record := range records {
					recordsIP, _, err := handler.findRecord(strings.Join(strings.Split(record.Data, ".")[1:], "."), record.Data, qType)
					if err != nil {
						return nil, nil, err
					}
					records = append(records, recordsIP...)
				}
			}
		default:

		}
	}

	return records, allExtRecords, nil
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
