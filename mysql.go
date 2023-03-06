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
	fmt.Println(sql, zone, host, qType)
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

func (handler *CoreDNSMySql) dbQueryIP(zone, host string) ([]*Record, error) {
	var allRecords = make([]*Record, 0)

	records, err := handler.dbQuery(zone, host, RecordType.A)
	if err != nil {
		return nil, err
	}
	allRecords = append(allRecords, records...)
	records, err = handler.dbQuery(zone, host, RecordType.AAAA)
	if err != nil {
		return nil, err
	}
	allRecords = append(allRecords, records...)
	return allRecords, nil
}

func (handler *CoreDNSMySql) findRecord(zone string, name string, qType string) ([]*Record, []*Record, error) {
	// 处理确定查询的是域本身？亦或是域名
	fmt.Println("11------")

	query := "@"
	if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}
	fmt.Println("12------")

	// 以 host, zone, type 对DB进行查询，并且得到记录
	var allExtRecords = make([]*Record, 0)
	records, err := handler.dbQuery(zone, query, qType)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("13------", records, len(records) == 0, qType == RecordType.A)

	// 如果DB中没有该域名对应查询类型的记录，则尝试查询该域名的所有类型的记录
	// 比如: 可能该域名本事其实是一个CNAME记录或者MX等等，
	if len(records) == 0 {
		// 判断查询类型是否为 A 或 AAAA，如果是则对该域名的CNAME记录进行查询
		switch qType {
		case RecordType.A, RecordType.AAAA:
			records, err = handler.dbQuery(zone, query, RecordType.CNAME)
			fmt.Println(records, err, zone, query, "-----------------")
			if err != nil {
				return nil, nil, err
			}
			if len(records) != 0 {
				for _, record := range records {
					fmt.Println(record)
					recordsIP, _, err := handler.findRecord(strings.Join(strings.Split(record.Data, ".")[1:], "."), record.Data, qType)
					if err != nil {
						return nil, nil, err
					}
					records = append(records, recordsIP...)
				}
			}
		default:

		}
	} else {

		switch qType {
		case RecordType.MX, RecordType.NS, RecordType.SOA:
			for _, record := range records {
				extRecords, err := handler.dbQueryIP(record.Zone, record.Host)
				if err != nil {
					return nil, nil, err
				}
				allExtRecords = append(allExtRecords, extRecords...)
			}
		}
	}
	fmt.Println("14------")

	// If no records found, check for wildcard records.
	// if len(records) == 0 && name != zone {
	// 	return handler.findWildcardRecords(ctx, w, r, zone, name, qType)
	// }

	return records, allExtRecords, nil
}

// findWildcardRecords attempts to find wildcard records
// recursively until it finds matching records.
// e.g. x.y.z -> *.y.z -> *.z -> *
// func (handler *CoreDNSMySql) findWildcardRecords(zone string, name string, types ...string) ([]*Record, error) {
// 	const (
// 		wildcard       = "*"
// 		wildcardPrefix = wildcard + "."
// 	)

// 	if name == wildcard {
// 		return nil, nil
// 	}
// 	return nil, nil

// 	// name = strings.TrimPrefix(name, wildcardPrefix)

// 	// target := wildcard
// 	// i, shot := dns.NextLabel(name, 0)
// 	// if !shot {
// 	// 	target = wildcardPrefix + name[i:]
// 	// }

// 	// return handler.findRecord(ctx, w, r, zone, target, types...)
// }

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

// func (handler *CoreDNSMySql) hosts(zone string, name string) ([]dns.RR, error) {
// 	recs, _, err := handler.findRecord(zone, name, "A")
// 	if err != nil {
// 		return nil, err
// 	}

// 	// answers := make([]dns.RR, 0)

// 	// for _, rec := range recs {
// 	// 	switch rec.Type {
// 	// 	case "A":
// 	// 		aRec, err := rec.AsARecord()
// 	// 		if err != nil {
// 	// 			return nil, err
// 	// 		}
// 	// 		answers = append(answers, aRec)
// 	// 	case "AAAA":
// 	// 		aRec, err := rec.AsAAAARecord()
// 	// 		if err != nil {
// 	// 			return nil, err
// 	// 		}
// 	// 		answers = append(answers, aRec)
// 	// 	case "CNAME":
// 	// 		aRec, _, err := rec.AsCNAMERecord()
// 	// 		if err != nil {
// 	// 			return nil, err
// 	// 		}
// 	// 		answers = append(answers, aRec)
// 	// 	}
// 	// }

// 	return recs, nil
// }

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
