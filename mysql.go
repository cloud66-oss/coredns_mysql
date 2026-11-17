package coredns_mysql

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func (handler *CoreDNSMySql) findRecord(zone string, name string, types ...string) ([]*Record, error) {
	db, err := handler.db()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var query string
	if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}
	sqlQuery := fmt.Sprintf("SELECT name, zone, ttl, record_type, content FROM %s WHERE zone = ? AND name = ? AND record_type IN ('%s')",
		handler.tableName,
		strings.Join(types, "','"))
	result, err := db.Query(sqlQuery, zone, query)
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

func (handler *CoreDNSMySql) findCNAMEChain(zone string, name string, types ...string) ([]*Record, error) {
	records := make([]*Record, 0)
	seen := make(map[string]bool)

	for len(seen) < maxCNAMEChain {
		key := strings.ToLower(dns.Fqdn(name))
		if seen[key] {
			break
		}
		seen[key] = true

		cnames, err := handler.findRecord(zone, name, "CNAME")
		if err != nil {
			return nil, err
		}
		if len(cnames) == 0 {
			break
		}
		records = append(records, cnames[0])

		target, err := cnames[0].cnameTarget()
		if err != nil {
			return nil, err
		}
		if target == "" || !dns.IsSubDomain(zone, target) {
			break
		}

		found, err := handler.findRecord(zone, target, types...)
		if err != nil {
			return nil, err
		}
		if len(found) > 0 {
			return append(records, found...), nil
		}
		name = target
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

func (handler *CoreDNSMySql) nameExists(zone string, name string) (bool, error) {
	if strings.EqualFold(dns.Fqdn(name), dns.Fqdn(zone)) {
		return true, nil
	}

	db, err := handler.db()
	if err != nil {
		return false, err
	}
	defer db.Close()

	relative := strings.TrimSuffix(name, "."+zone)
	candidates := []interface{}{zone, relative, "%." + escapeLike(relative)}
	placeholders := make([]string, 0)
	labels := dns.SplitDomainName(relative)
	for i := 1; i <= len(labels); i++ {
		placeholders = append(placeholders, "?")
		candidates = append(candidates, strings.Join(append([]string{"*"}, labels[i:]...), "."))
	}

	sqlQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE zone = ? AND (name = ? OR name LIKE ? OR name IN (%s))",
		handler.tableName,
		strings.Join(placeholders, ","))
	var count int
	err = db.QueryRow(sqlQuery, candidates...).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(s)
}

func (handler *CoreDNSMySql) loadZones() error {
	db, err := handler.db()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Query("SELECT DISTINCT zone FROM " + handler.tableName)
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
