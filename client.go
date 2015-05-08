package gowasd

import (
	"errors"
	"github.com/miekg/dns"
	"sort"
	"strconv"
	"strings"
)

const DefaultPropertyVersion = 1

type VersionedProperties map[int]map[string]string

type Client struct {
	c    *dns.Client
	Addr string
}

func New(c *dns.Client, addr string) (out Client, err error) {
	if addr == "" {
		addr, err = addrFromResolvConf("/etc/resolv.conf")
		if err != nil {
			return out, err
		}
	}
	return Client{c: c, Addr: addr}, nil
}

type Service struct {
	Name     string
	Protocol string
	Domain   string
}

func (srv Service) DnsName() string {
	return dumpDnsName(srv.DnsLabels())
}

func (srv Service) DnsLabels() []string {
	return []string{"_" + srv.Name, "_" + srv.Protocol, srv.Domain}
}

type Instance struct {
	Service
	Description string
	FullName    string
}

func (inst Instance) DnsName() string {
	return dumpDnsName(inst.DnsLabels())
}

func (inst Instance) DnsLabels() []string {
	return append([]string{inst.Description}, inst.Service.DnsLabels()...)
}

type Endpoint struct {
	Host     string
	Port     int
	priority int
}

type EndpointList []Endpoint

func (self EndpointList) Len() int {
	return len(self)
}

func (self EndpointList) Less(i, j int) bool {
	// Note we wanted this sorted descending, so we flip the inequality
	return self[i].priority > self[j].priority
}

func (self EndpointList) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

type InstanceResolution struct {
	Instance
	Targets    EndpointList
	Properties VersionedProperties
}

func (self *Client) ServiceInstances(srv Service) (out []Instance, err error) {
	msg := new(dns.Msg)
	msg.SetQuestion(srv.DnsName(), dns.TypePTR)
	resp, _, err := self.c.Exchange(msg, self.Addr)
	if err != nil {
		return nil, err
	}

	var parts []string
	out = make([]Instance, len(resp.Answer))
	for i, ans := range resp.Answer {
		if ansPtr, ok := ans.(*dns.PTR); ok {
			parts = parseDnsName(ansPtr.Ptr, 4)
			out[i] = Instance{
				Description: parts[0],
				FullName:    ansPtr.Ptr,
				Service: Service{
					Name:     parts[1][1:],
					Protocol: parts[2][1:],
					Domain:   parts[3],
				},
			}
		}
	}

	return
}

func (self *Client) ResolveInstance(inst Instance) (out InstanceResolution, err error) {

	responses := make(chan *dns.Msg)

	name := inst.DnsName()
	record_types := [...]uint16{dns.TypeSRV, dns.TypeTXT}
	for _, record_type := range record_types {
		go func(t uint16, n string) {
			msg := new(dns.Msg)
			msg.SetQuestion(n, t)
			resp, _, err := self.c.Exchange(msg, self.Addr)
			if err != nil {
				return
			}
			responses <- resp
		}(record_type, name)
	}

	out.Instance = inst
	out.Targets = make(EndpointList, 0, 3) // 3 is a fair guess!
	out.Properties = make(VersionedProperties)

	for i := 0; i < len(record_types); i++ {
		r := <-responses
		for _, anyRR := range r.Answer {
			switch rr := anyRR.(type) {
			case *dns.SRV:
				// TODO: weight
				out.Targets = append(out.Targets, Endpoint{
					Host:     rr.Target,
					Port:     int(rr.Port),
					priority: int(rr.Priority),
				})
			case *dns.TXT:
				parseTxtRecordForProperties(rr, &out)
			}
		}
	}

	sort.Sort(out.Targets)

	return
}

func parseTxtRecordForProperties(rr *dns.TXT, out *InstanceResolution) {
	var propParts []string
	var k, v string

	propVersion := DefaultPropertyVersion

	for i, prop := range rr.Txt {
		propParts = strings.SplitN(prop, "=", 2)
		if len(propParts) != 2 || propParts[0] == "" {
			continue
		}
		k, v = propParts[0], propParts[1]

		if i == 0 && k == "txtvers" {
			pv64, err := strconv.ParseInt(v, 0, 64)
			if err == nil {
				propVersion = int(pv64)
				continue
			}
		}

		if _, ok := out.Properties[propVersion]; !ok {
			out.Properties[propVersion] = make(map[string]string)
		}

		out.Properties[propVersion][k] = v
	}

}

func parseDnsName(s string, n int) (out []string) {
	// must be .-terminated

	var escaped bool
	var label string
	out = make([]string, 0, 10)

	for i, c := range s {
		if c == '\\' {
			escaped = true
			continue
		} else if c == '.' && !escaped {
			out = append(out, label)
			label = ""
			escaped = false
			if n > 0 && len(out) == n-1 {
				out = append(out, s[i+1:len(s)-1])
				break
			}
		} else {
			escaped = false
			label += string(c)
		}
	}

	return
}

func dumpDnsName(n []string) (out string) {
	for i, l := range n {
		if i < len(n)-1 {
			l = strings.Replace(l, "\\", "\\\\", -1)
			l = strings.Replace(l, ".", "\\.", -1)
			l = strings.Replace(l, " ", "\\ ", -1)
		}
		out += l + "."
	}
	return
}

func addrFromResolvConf(fn string) (out string, err error) {
	clientConfig, err := dns.ClientConfigFromFile(fn)
	if err != nil {
		return
	}
	if len(clientConfig.Servers) == 0 {
		return out, errors.New("no DNS servers found in " + fn)
	}
	return clientConfig.Servers[0] + ":" + clientConfig.Port, nil
}
