package coredns_mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	_ "github.com/go-sql-driver/mysql"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

type CoreDNSMySql struct {
	Next               plugin.Handler
	Dsn                string
	TableName          string
	MaxLifetime        time.Duration
	MaxOpenConnections int
	MaxIdleConnections int

	Ttl            uint32
	lastZoneUpdate time.Time
	zoneUpdateTime time.Duration
	zones          []string
	dbConn         *sql.DB
}

// ServeDNS implements the plugin.Handler interface.
func (handler *CoreDNSMySql) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	qName := state.Name()
	qType := state.Type()
	fmt.Println("1----------------")
	if time.Since(handler.lastZoneUpdate) > handler.zoneUpdateTime {
		err := handler.loadZones()
		if err != nil {
			return handler.errorResponse(state, dns.RcodeServerFailure, err)
		}
	}
	fmt.Println("2----------------")

	// TODO 此处可能有性能瓶颈，如果域很多的话，则需要遍历很多次，有不同的域就要遍历一次. 如果域很多，可以考虑采用hash表进行优化
	qZone := plugin.Zones(handler.zones).Matches(qName)
	if qZone == "" {
		return plugin.NextOrFailure(handler.Name(), handler.Next, ctx, w, r)
	}
	fmt.Println("3----------------")

	records, err := handler.findRecord(qZone, qName, qType)
	if err != nil {
		return handler.errorResponse(state, dns.RcodeServerFailure, err)
	}
	fmt.Println("4----------------")

	var appendSOA bool
	if len(records) == 0 {
		appendSOA = true
		// no record found but we are going to return a SOA
		recs, err := handler.findRecord(qZone, "@", "SOA")
		if err != nil {
			return handler.errorResponse(state, dns.RcodeServerFailure, err)
		}
		records = append(records, recs...)
	}

	if qType == "AXFR" {
		return handler.errorResponse(state, dns.RcodeNotImplemented, nil)
	}
	fmt.Println("5----------------")

	answers := make([]dns.RR, 0)
	extras := make([]dns.RR, 0)

	for _, record := range records {
		var answer dns.RR
		switch record.Type {
		case "A":
			answer, extras, err = record.AsARecord()
		case "AAAA":
			answer, extras, err = record.AsAAAARecord()
		case "CNAME":
			answer, extras, err = record.AsCNAMERecord()
		case "SOA":
			answer, extras, err = record.AsSOARecord()
		case "SRV":
			answer, extras, err = record.AsSRVRecord()
		case "NS":
			answer, extras, err = record.AsNSRecord()
		case "MX":
			answer, extras, err = record.AsMXRecord()
		case "TXT":
			answer, extras, err = record.AsTXTRecord()
		case "CAA":
			answer, extras, err = record.AsCAARecord()
		default:
			return handler.errorResponse(state, dns.RcodeNotImplemented, nil)
		}

		if err != nil {
			return handler.errorResponse(state, dns.RcodeServerFailure, err)
		}
		if answer != nil {
			answers = append(answers, answer)
		}
	}
	fmt.Println("6----------------")

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = false
	m.Compress = true

	if !appendSOA {
		m.Answer = append(m.Answer, answers...)
	} else {
		m.Ns = append(m.Ns, answers...)
	}
	m.Extra = append(m.Extra, extras...)
	fmt.Println("7----------------")

	state.SizeAndDo(m)
	m = state.Scrub(m)
	err = w.WriteMsg(m)
	fmt.Println(err.Error(), err)
	fmt.Println("8----------------")
	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (handler *CoreDNSMySql) Name() string { return "handler" }

func (handler *CoreDNSMySql) errorResponse(state request.Request, rCode int, err error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(state.Req, rCode)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, false, true

	state.SizeAndDo(m)
	_ = state.W.WriteMsg(m)
	// Return success as the rCode to signal we have written to the client.
	return dns.RcodeSuccess, err
}
