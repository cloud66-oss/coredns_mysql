package coredns_mysql

import (
	"net"

	"github.com/miekg/dns"
)

type Record struct {
	Host string
	Zone string
	Type string
	Data string
	TTL  uint32

	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string
	Flag     uint8
	Tag      string

	PrimaryNS  string
	RespPerson string
	Serial     uint32
	Refresh    uint32
	Retry      uint32
	Expire     uint32
	Minimum    uint32

	remark string

	handler *CoreDNSMySql
}

var RecordType = struct {
	A     string
	AAAA  string
	CNAME string
	NS    string
	SOA   string
	TXT   string
	MX    string
	SRV   string
	CAA   string
	AXFR  string
}{"A", "AAAA", "CNAME", "NS", "SOA", "TXT", "MX", "SRV", "CAA", "AXFR"}

func (rec *Record) AsARecord() (records []dns.RR, err error) {
	r := new(dns.A)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if rec.Data == "" {
		return nil, nil
	}
	r.A = net.ParseIP(rec.Data).To4()
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsAAAARecord() (records []dns.RR, err error) {
	r := new(dns.AAAA)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeAAAA,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if rec.Data == "" {
		return nil, nil
	}
	r.AAAA = net.ParseIP(rec.Data)
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsCNAMERecord() (records []dns.RR, err error) {
	r := new(dns.CNAME)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeCNAME,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Data) == 0 {
		return nil, nil
	}
	r.Target = dns.Fqdn(rec.Data)
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsNSRecord() (records []dns.RR, err error) {
	r := new(dns.NS)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeNS,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Data) == 0 {
		return nil, nil
	}

	r.Ns = rec.Data
	if err != nil {
		return nil, nil
	}
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsTXTRecord() (records []dns.RR, err error) {
	r := new(dns.TXT)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeTXT,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}
	if len(rec.Data) == 0 {
		return nil, nil
	}

	r.Txt = split255(rec.Data)
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsSOARecord() (records []dns.RR, err error) {
	r := new(dns.SOA)

	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.Zone),
		Rrtype: dns.TypeSOA,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}
	r.Ns = rec.PrimaryNS
	r.Mbox = rec.RespPerson
	r.Refresh = rec.Refresh
	r.Retry = rec.Retry
	r.Expire = rec.Expire
	r.Minttl = rec.minTtl()
	r.Serial = rec.Serial
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsSRVRecord() (records []dns.RR, err error) {
	r := new(dns.SRV)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeSRV,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Target) == 0 {
		return nil, nil
	}

	r.Priority = rec.Priority
	r.Weight = rec.Weight
	r.Port = rec.Port
	r.Target = rec.Target
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsMXRecord() (records []dns.RR, err error) {
	r := new(dns.MX)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeMX,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Data) == 0 {
		return nil, nil
	}

	r.Mx = rec.Data
	r.Preference = rec.Priority
	if err != nil {
		return nil, nil
	}
	records = append(records, r)
	return records, nil
}

func (rec *Record) AsCAARecord() (records []dns.RR, err error) {
	r := new(dns.CAA)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeCAA,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if rec.Data == "" || rec.Tag == "" {
		return nil, nil
	}

	r.Flag = rec.Flag
	r.Tag = rec.Tag
	r.Value = rec.Data
	records = append(records, r)
	return records, nil
}

func (rec *Record) minTtl() uint32 {
	if rec.TTL == 0 {
		return defaultTtl
	}
	return rec.TTL
}

func split255(s string) []string {
	if len(s) < 255 {
		return []string{s}
	}
	sx := []string{}
	p, i := 0, 255
	for {
		if i <= len(s) {
			sx = append(sx, s[p:i])
		} else {
			sx = append(sx, s[p:])
			break

		}
		p, i = p+255, i+255
	}

	return sx
}

func (rec *Record) fqdn() string {
	if rec.Host == "@" {
		return rec.Zone
	}
	return rec.Host + "." + rec.Zone
}
