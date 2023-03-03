package coredns_mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func (handler *CoreDNSMySql) findRecord(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, zone string, name string, types ...string) ([]*Record, error) {
	dbConn := handler.dbConn
	query := "@"
	if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}

	sqlQuery := fmt.Sprintf("SELECT host, zone, type, data, ttl, "+
		"priority, weight, port, target, flag, tag, "+
		"primary_ns, resp_person, serial, refresh, retry, expire, minimum, "+
		"remark	FROM %s WHERE zone = ? AND host = ? AND type IN ('%s')",
		handler.TableName, strings.Join(types, "','"))
	results, err := dbConn.Query(sqlQuery, zone, query)
	if err != nil {
		return nil, err
	}
	records, err := handler.getRecordsFromQueryResults(results)
	if err != nil {
		return nil, err
	}

	// maybe this is a CNAME or MX record, should resolve this record
	if len(records) == 0 {
		sqlQueryANY := fmt.Sprintf("SELECT host, zone, type, data, ttl, "+
			"priority, weight, port, target, flag, tag, "+
			"primary_ns, resp_person, serial, refresh, retry, expire, minimum, "+
			"remark	FROM %s WHERE zone = ? AND host = ?", handler.TableName)
		results, err := dbConn.Query(sqlQueryANY, zone, query)
		if err != nil {
			return nil, err
		}
		records, err = handler.getRecordsFromQueryResults(results)
		if err != nil {
			return nil, err
		}

		allExtRecords := make([]*Record, 0)
		if len(records) > 0 {

			for _, record := range records {
				qZone := plugin.Zones(handler.zones).Matches(record.Data)
				if qZone == "" {
					continue
				}
				extRecords, err := handler.findRecord(ctx, w, r, qZone, record.Data, types...)
				if err != nil {
					return nil, err
				}
				fmt.Printf("%#v\n", extRecords)
				allExtRecords = append(allExtRecords, extRecords...)
			}
		}
		records = append(records, allExtRecords...)
	}

	// If no records found, check for wildcard records.
	if len(records) == 0 && name != zone {
		return handler.findWildcardRecords(ctx, w, r, zone, name, types...)
	}

	return records, nil
}

// findWildcardRecords attempts to find wildcard records
// recursively until it finds matching records.
// e.g. x.y.z -> *.y.z -> *.z -> *
func (handler *CoreDNSMySql) findWildcardRecords(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, zone string, name string, types ...string) ([]*Record, error) {
	const (
		wildcard       = "*"
		wildcardPrefix = wildcard + "."
	)

	if name == wildcard {
		return nil, nil
	}

	name = strings.TrimPrefix(name, wildcardPrefix)

	target := wildcard
	i, shot := dns.NextLabel(name, 0)
	if !shot {
		target = wildcardPrefix + name[i:]
	}

	return handler.findRecord(ctx, w, r, zone, target, types...)
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

func (handler *CoreDNSMySql) hosts(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, zone string, name string) ([]dns.RR, error) {
	recs, err := handler.findRecord(ctx, w, r, zone, name, "A", "AAAA", "CNAME")
	if err != nil {
		return nil, err
	}

	answers := make([]dns.RR, 0)

	for _, rec := range recs {
		switch rec.Type {
		case "A":
			aRec, _, err := rec.AsARecord()
			if err != nil {
				return nil, err
			}
			answers = append(answers, aRec)
		case "AAAA":
			aRec, _, err := rec.AsAAAARecord()
			if err != nil {
				return nil, err
			}
			answers = append(answers, aRec)
		case "CNAME":
			aRec, _, err := rec.AsCNAMERecord()
			if err != nil {
				return nil, err
			}
			answers = append(answers, aRec)
		}
	}

	return answers, nil
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
		}

		records = append(records, record)
	}
	return
}
