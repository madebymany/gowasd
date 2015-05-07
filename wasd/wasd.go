package main

import (
	"flag"
	"fmt"
	"github.com/madebymany/gowasd"
	"github.com/miekg/dns"
	"log"
	"os"
)

var (
	name = flag.String("name", "",
		"The name of the service you wish to query")
	protocol = flag.String("protocol", "tcp",
		"The protocol of the service, either udp or tcp")
	domain = flag.String("domain", "",
		"The top level domain you wish to query the service against")
	args    []string
	argNum  int = 0
	command string
)

func main() {

	log.SetFlags(0)

	flag.Parse()
	args = flag.Args()

	command = getNextArg("no command given")

	sd, err := gowasd.New(new(dns.Client), "")
	if err != nil {
		log.Fatal("gowasd error: ", err)
	}
	s := gowasd.Service{
		Name:     *name,
		Protocol: *protocol,
		Domain:   *domain,
	}

	i, err := sd.ServiceInstances(s)
	if err != nil {
		log.Fatal("gowasd error: ", err)
	}
	if len(i) > 0 {
		ri, err := sd.ResolveInstance(i[0])
		if err != nil {
			log.Fatal("gowasd error: ", err)
		}
		log.Println(ri.Properties[1])
		log.Println(fmt.Printf("%#v", ri.Properties))
	}
}

func fatalUsageError(errMsg string) {
	fmt.Fprintln(os.Stderr, "fatal: "+errMsg+"\n")
	usage()
	os.Exit(1)
}

func getNextArg(errMsg string) (val string) {
	if len(args) >= (argNum + 1) {
		val = args[argNum]
		argNum += 1
	} else if errMsg != "" {
		fatalUsageError(errMsg)
	}
	return
}
