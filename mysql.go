package coredns_mysql

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func (handler *CoreDNSMySql) findRecord(zone string, name string, types ...string) ([]*Record, error) {
	dbConn := handler.dbConn
	var query string
	if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}
	sqlQuery := fmt.Sprintf("SELECT name, zone, ttl, record_type, content FROM %s WHERE zone = ? AND name = ? AND record_type IN ('%s')",
		handler.TableName,
		strings.Join(types, "','"))
	result, err := dbConn.Query(sqlQuery, zone, query)
	if err != nil {
		return nil, err
	}

	var recordName string
	var recordZone string
	var recordType string
	var ttl uint32
	var content string
	records := make([]*Record, 0)
	for result.Next() {
		err = result.Scan(&recordName, &recordZone, &ttl, &recordType, &content)
		if err != nil {
			return nil, err
		}

		records = append(records, &Record{
			Name:       recordName,
			Zone:       recordZone,
			RecordType: recordType,
			Ttl:        ttl,
			Content:    content,
			handler:    handler,
		})
	}

	// If no records found, check for wildcard records.
	if len(records) == 0 && name != zone {
		return handler.findWildcardRecords(zone, name, types...)
	}

	return records, nil
}

// findWildcardRecords attempts to find wildcard records
// recursively until it finds matching records.
// e.g. x.y.z -> *.y.z -> *.z -> *
func (handler *CoreDNSMySql) findWildcardRecords(zone string, name string, types ...string) ([]*Record, error) {
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

	return handler.findRecord(zone, target, types...)
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

func (handler *CoreDNSMySql) hosts(zone string, name string) ([]dns.RR, error) {
	recs, err := handler.findRecord(zone, name, "A", "AAAA", "CNAME")
	if err != nil {
		return nil, err
	}

	answers := make([]dns.RR, 0)

	for _, rec := range recs {
		switch rec.RecordType {
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
