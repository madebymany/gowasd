package gowasd

import (
	"github.com/miekg/dns"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func startTestDnsServer() (cmd *exec.Cmd) {
	cmd = exec.Command("dnsmasq",
		"-p", "53533", "-d", "-C", "test_dnsmasq.conf")
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second / 10) // sorry
	return
}

func stopTestDnsServer(cmd *exec.Cmd) {
	err := cmd.Process.Signal(os.Interrupt)
	if err != nil {
		panic(err)
	}
	cmd.Wait()
}

const testDnsAddr = "127.0.0.1:53533"

func TestParseDnsName(t *testing.T) {
	var res []string

	tryExample := func(s string, n int, exp []string) {
		res = parseDnsName(s, n)
		if !reflect.DeepEqual(res, exp) {
			t.Errorf("tried %#v, expected %#v, got %#v", s, exp, res)
		}
	}

	tryExample("_test._tcp.example.com.", 3,
		[]string{"_test", "_tcp", "example.com"})
	tryExample("Woop._test._tcp.example.com.", 4,
		[]string{"Woop", "_test", "_tcp", "example.com"})
	tryExample("Woop hello there._test._tcp.example.com.", 4,
		[]string{"Woop hello there", "_test", "_tcp", "example.com"})
	tryExample("Woop\\ hello\\ there._test._tcp.example.com.", 4,
		[]string{"Woop hello there", "_test", "_tcp", "example.com"})
}

func TestDumpDnsName(t *testing.T) {
	var res string

	tryExample := func(s []string, exp string) {
		res = dumpDnsName(s)
		if res != exp {
			t.Errorf("tried %#v, expected %#v, got %#v", s, exp, res)
		}
	}

	tryExample([]string{"_test", "_tcp", "example.com"},
		"_test._tcp.example.com.")
	tryExample([]string{"Woo", "_test", "_tcp", "example.com"},
		"Woo._test._tcp.example.com.")
	tryExample([]string{"Woo yay", "_test", "_tcp", "example.com"},
		"Woo\\ yay._test._tcp.example.com.")
	tryExample([]string{"1. this is awesome", "_test", "_tcp", "example.com"},
		"1\\.\\ this\\ is\\ awesome._test._tcp.example.com.")
}

func TestAddrFromResolvConf(t *testing.T) {
	addr, err := addrFromResolvConf("test_resolv.conf")
	if err != nil {
		t.Fatal(err)
	}
	expected := "127.1.2.3:53"
	if addr != expected {
		t.Fatalf("expected %#v, got %#v", expected, addr)
	}
}

func TestClient_ServiceInstances(t *testing.T) {
	dnsmasq := startTestDnsServer()
	c, err := New(new(dns.Client), testDnsAddr)
	if err != nil {
		panic(err)
	}

	insts, err := c.ServiceInstances(Service{
		Name:     "test",
		Protocol: "tcp",
		Domain:   "example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected :=
		[]Instance{
			Instance{
				Service:     Service{Name: "test", Protocol: "tcp", Domain: "example.com"},
				Description: "Woop",
				FullName:    "Woop._test._tcp.example.com."},
			Instance{
				Service:     Service{Name: "test", Protocol: "tcp", Domain: "example.com"},
				Description: "Hello There",
				FullName:    "Hello\\ There._test._tcp.example.com.",
			}}

	if !reflect.DeepEqual(insts, expected) {
		t.Errorf("expected %#v, got %#v", expected, insts)
	}

	stopTestDnsServer(dnsmasq)
}

func TestClient_ResolveInstance(t *testing.T) {
	dnsmasq := startTestDnsServer()
	c, err := New(new(dns.Client), testDnsAddr)
	if err != nil {
		panic(err)
	}

	inst := Instance{
		Service:     Service{Name: "test", Protocol: "tcp", Domain: "example.com"},
		Description: "Woop",
		FullName:    "Woop._test._tcp.example.com.",
	}

	resInst, err := c.ResolveInstance(inst)
	if err != nil {
		t.Fatal(err)
	}

	expected := InstanceResolution{
		Instance: Instance{
			Service: Service{
				Name:     "test",
				Protocol: "tcp",
				Domain:   "example.com"},
			Description: "Woop",
			FullName:    "Woop._test._tcp.example.com.",
		},
		Targets: EndpointList{
			Endpoint{
				Host:     "woop.example.com.",
				Port:     49153,
				priority: 0,
			}},
		Properties: VersionedProperties{
			1: map[string]string{
				"hello": "there",
				"this":  "is=fun",
			},
			2: map[string]string{
				"second": "version",
				"gosh":   "wow",
			},
		}}

	if !reflect.DeepEqual(resInst, expected) {
		t.Errorf("expected %#v, got %#v", expected, resInst)
	}

	stopTestDnsServer(dnsmasq)
}
