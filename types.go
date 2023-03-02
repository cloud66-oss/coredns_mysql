package coredns_mysql

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

//type Record struct {
//	Zone       string
//	Name       string
//	RecordType string
//	Ttl        uint32
//	Content    string
//
//	handler *CoreDNSMySql
//}

type Record struct {
	Host string
	Zone string
	Type string
	Data string
	TTL  uint32

	Priority uint32
	Weight   uint32
	Port     uint32
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

type ARecord struct {
	Ip net.IP `json:"ip"`
}

type AAAARecord struct {
	Ip net.IP `json:"ip"`
}

type TXTRecord struct {
	Text string `json:"text"`
}

type CNAMERecord struct {
	Host string `json:"host"`
}

type NSRecord struct {
	Host string `json:"host"`
}

type MXRecord struct {
	Host       string `json:"host"`
	Preference uint16 `json:"preference"`
}

type SRVRecord struct {
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
	Port     uint16 `json:"port"`
	Target   string `json:"target"`
}

type SOARecord struct {
	Ns      string `json:"ns"`
	MBox    string `json:"MBox"`
	Refresh uint32 `json:"refresh"`
	Retry   uint32 `json:"retry"`
	Expire  uint32 `json:"expire"`
	MinTtl  uint32 `json:"minttl"`
}

type CAARecord struct {
	Flag  uint8  `json:"flag"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

func (rec *Record) AsARecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.A)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if rec.Data == nil {
		return nil, nil, nil
	}
	r.A = rec.Data
	return r, nil, nil
}

func (rec *Record) AsAAAARecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.AAAA)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeAAAA,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if rec.Data == nil {
		return nil, nil, nil
	}
	r.AAAA = rec.Data
	return r, nil, nil
}

func (rec *Record) AsTXTRecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.TXT)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeTXT,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}
	if len(rec.Data) == 0 {
		return nil, nil, nil
	}

	r.Txt = split255(rec.Data)
	return r, nil, nil
}

func (rec *Record) AsCNAMERecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.CNAME)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeCNAME,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Data) == 0 {
		return nil, nil, nil
	}
	r.Target = dns.Fqdn(rec.Data)
	return r, nil, nil
}

func (rec *Record) AsNSRecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.NS)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeNS,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Data) == 0 {
		return nil, nil, nil
	}

	r.Ns = rec.Data
	extras, err = rec.handler.hosts(rec.Zone, r.Ns)
	if err != nil {
		return nil, nil, err
	}
	return r, extras, nil
}

func (rec *Record) AsMXRecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.MX)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeMX,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Data) == 0 {
		return nil, nil, nil
	}

	r.Mx = rec.Data
	r.Preference = rec.Priority
	extras, err = rec.handler.hosts(rec.Zone, rec.Data)
	if err != nil {
		return nil, nil, err
	}

	return r, extras, nil
}

func (rec *Record) AsSRVRecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.SRV)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeSRV,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if len(rec.Target) == 0 {
		return nil, nil, nil
	}

	r.Priority = rec.Priority
	r.Weight = rec.Weight
	r.Port = rec.Port
	r.Target = rec.Target
	return r, nil, nil
}

func (rec *Record) AsSOARecord() (record dns.RR, extras []dns.RR, err error) {
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

	return r, nil, nil
}

func (rec *Record) AsCAARecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.CAA)
	r.Hdr = dns.RR_Header{
		Name:   dns.Fqdn(rec.fqdn()),
		Rrtype: dns.TypeCAA,
		Class:  dns.ClassINET,
		Ttl:    rec.minTtl(),
	}

	if rec.Data == "" || rec.Tag == "" {
		return nil, nil, nil
	}

	r.Flag = rec.Flag
	r.Tag = rec.Tag
	r.Value = rec.Data

	return r, nil, nil
}

func (rec *Record) minTtl() uint32 {
	if rec.TTL == 0 {
		return defaultTtl
	}
	return rec.TTL
}

func (rec *Record) serial() uint32 {
	return uint32(time.Now().Unix())
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
