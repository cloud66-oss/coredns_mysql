package coredns_mysql

import (
	"encoding/json"
	"net"
	"time"

	"github.com/miekg/dns"
)

type Record struct {
	Zone       string
	Name       string
	RecordType string
	Ttl        uint32
	Content    string

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
	var aRec *ARecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if aRec.Ip == nil {
		return nil, nil, nil
	}
	r.A = aRec.Ip
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
	var aRec *AAAARecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if aRec.Ip == nil {
		return nil, nil, nil
	}

	r.AAAA = aRec.Ip
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
	var aRec *TXTRecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if len(aRec.Text) == 0 {
		return nil, nil, nil
	}

	r.Txt = split255(aRec.Text)
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
	var aRec *CNAMERecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if len(aRec.Host) == 0 {
		return nil, nil, nil
	}
	r.Target = dns.Fqdn(aRec.Host)
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
	var aRec *CNAMERecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if len(aRec.Host) == 0 {
		return nil, nil, nil
	}

	r.Ns = aRec.Host
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
	var aRec *MXRecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if len(aRec.Host) == 0 {
		return nil, nil, nil
	}

	r.Mx = aRec.Host
	r.Preference = aRec.Preference
	extras, err = rec.handler.hosts(rec.Zone, aRec.Host)
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
	var aRec *SRVRecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if len(aRec.Target) == 0 {
		return nil, nil, nil
	}

	r.Target = aRec.Target
	r.Weight = aRec.Weight
	r.Port = aRec.Port
	r.Priority = aRec.Priority
	return r, nil, nil
}

func (rec *Record) AsSOARecord() (record dns.RR, extras []dns.RR, err error) {
	r := new(dns.SOA)
	var aRec *SOARecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if aRec.Ns == "" {
		r.Hdr = dns.RR_Header{
			Name:   dns.Fqdn(rec.fqdn()),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    rec.minTtl(),
		}
		r.Ns = "ns1." + rec.Name
		r.Mbox = "hostmaster." + rec.Name
		r.Refresh = 86400
		r.Retry = 7200
		r.Expire = 3600
		r.Minttl = rec.minTtl()
	} else {
		r.Hdr = dns.RR_Header{
			Name:   dns.Fqdn(rec.Zone),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    rec.minTtl(),
		}
		r.Ns = aRec.Ns
		r.Mbox = aRec.MBox
		r.Refresh = aRec.Refresh
		r.Retry = aRec.Retry
		r.Expire = aRec.Expire
		r.Minttl = aRec.MinTtl
	}
	r.Serial = rec.serial()

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
	var aRec *CAARecord
	err = json.Unmarshal([]byte(rec.Content), &aRec)
	if err != nil {
		return nil, nil, err
	}

	if aRec.Value == "" || aRec.Tag == "" {
		return nil, nil, nil
	}

	r.Flag = aRec.Flag
	r.Tag = aRec.Tag
	r.Value = aRec.Value

	return r, nil, nil
}

func (rec *Record) minTtl() uint32 {
	if rec.Ttl == 0 {
		return defaultTtl
	}
	return rec.Ttl
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
	if rec.Name == "" {
		return rec.Zone
	}
	return rec.Name + "." + rec.Zone
}
