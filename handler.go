package coredns_mysql

import (
	"database/sql"
	"time"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
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
	clog.Debug("coredns-mysql: In ServeDNS method")
	// 包装的一个对象，方便使用
	state := request.Request{W: w, Req: r}

	// 查询的名字，如 dig A qq.com 则 qName 为 qq.com
	qName := state.Name()
	// 查询的类型，为 A
	qType := state.Type()

	// 不支持区域传送
	if qType == RecordType.AXFR {
		clog.Debug("coredns-mysql: AXFR type request not implemented")
		return handler.errorResponse(state, dns.RcodeNotImplemented, nil)
	}

	// coredns-mysql插件会缓存所有的zone，以提高效率，会定时更新zone
	if time.Since(handler.lastZoneUpdate) > handler.zoneUpdateTime {
		clog.Debug("coredns-mysql: Update zones, current zones", handler.zones)
		err := handler.loadZones()
		if err != nil {
			return handler.errorResponse(state, dns.RcodeServerFailure, err)
		}
		clog.Debug("coredns-mysql: Updated zones, current zones", handler.zones)
	}

	// 判断当前 qName 是否能匹配到合适的 zone ，最长匹配原则
	qZone := plugin.Zones(handler.zones).Matches(qName)
	clog.Debug("coredns-mysql: Use", qName, "match zones, matched zones is", qZone)

	// 如果不能匹配，则转给下一个 coredns 插件
	if qZone == "" {
		clog.Debug("coredns-mysql: Not fount matched zone, retrun request to next plugin")
		return plugin.NextOrFailure(handler.Name(), handler.Next, ctx, w, r)
	}

	// 从数据库中查询该记录
	clog.Debug("coredns-mysql: Use zone", qZone, "name", qName, "type", qType, "to query db")
	records, err := handler.findRecord(qZone, qName, qType)
	if err != nil {
		return handler.errorResponse(state, dns.RcodeServerFailure, err)
	}

	// 如果未查到域名，则查询SOA记录
	if len(records) == 0 {
		// 查询SOA记录
		clog.Debug("coredns-mysql: Not query any record, query SOA record")
		records, err = handler.findRecord(qZone, "@", RecordType.SOA)
		if err != nil {
			return handler.errorResponse(state, dns.RcodeServerFailure, err)
		}
	}

	results, err := handler.resolveRecords(records)
	clog.Debug("coredns-mysql: Query all results are", results)
	if err != nil {
		return handler.errorResponse(state, dns.RcodeServerFailure, err)
	}

	// extResults, err := handler.resolveRecords(records)
	// if err != nil {
	// 	return handler.errorResponse(state, dns.RcodeServerFailure, err)
	// }
	// 创建一个DNS结果
	m := new(dns.Msg)
	// 该结果用与响应 r 这个请求
	m.SetReply(r)
	// 设置为权威答案
	m.Authoritative = true
	// 允许递归查询
	m.RecursionAvailable = true
	// 允许压缩
	m.Compress = true

	// 若添加 SOA，则需要添加相关的 NS 信息
	m.Answer = append(m.Answer, results...)
	// 不添加任何额外的DNS信息
	// m.Extra = append(m.Extra, extResults...)

	// 回复响应
	state.SizeAndDo(m)
	m = state.Scrub(m)
	w.WriteMsg(m)
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

func (handler *CoreDNSMySql) resolveRecords(records []*Record) ([]dns.RR, error) {
	var allAnswer = make([]dns.RR, 0)
	var err error
	for _, record := range records {
		var answer []dns.RR

		switch record.Type {
		case "A":
			answer, err = record.AsARecord()
		case "AAAA":
			answer, err = record.AsAAAARecord()
		case "CNAME":
			answer, err = record.AsCNAMERecord()
		case "NS":
			answer, err = record.AsNSRecord()
		case "TXT":
			answer, err = record.AsTXTRecord()
		case "SOA":
			answer, err = record.AsSOARecord()
		case "SRV":
			answer, err = record.AsSRVRecord()
		case "MX":
			answer, err = record.AsMXRecord()
		case "CAA":
			answer, err = record.AsCAARecord()
		default:
			return nil, err
		}
		allAnswer = append(allAnswer, answer...)
	}
	return allAnswer, err
}
