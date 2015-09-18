package main

import (
	"errors"
	"flag"
	"github.com/madebymany/gowasd"
	"github.com/miekg/dns"
	"log"
	"os"
)

var (
	description = flag.String("description", "",
		"Description of the service instance you want to query")
	subtype = flag.String("subtype", "",
		"The subtype record type to retrieve")
	name = flag.String("name", "",
		"The name of the service you wish to query")
	protocol = flag.String("protocol", "tcp",
		"The protocol of the service, either udp or tcp")
	domain = flag.String("domain", "",
		"The top level domain you wish to query the service against")
	version = flag.Int("version", 1,
		"The version of the properties you want to load")
	format = flag.String("format", "default",
		"Output format. Can be 'default' or 'env' (for eval-able environment variables)")
	serverAddr = flag.String("server", "",
		"Address of the server, in ip:port format.")

	// setup log levels
	Info  *log.Logger
	Fatal *log.Logger
)

func main() {
	Info = log.New(os.Stdout, "", 0)
	Fatal = log.New(os.Stderr, "Error: ", 0)

	flag.Parse()
	args := flag.Args()

	fmtr, err := getFormatter()
	if err != nil {
		Fatal.Print(err)
		os.Exit(1)
	}

	sd, err := gowasd.New(new(dns.Client), *serverAddr)
	if err != nil {
		Fatal.Print("gowasd error: ", err)
		os.Exit(1)
	}

	service := gowasd.Service{
		Subtype:  *subtype,
		Name:     *name,
		Protocol: *protocol,
		Domain:   *domain,
	}

	var serviceInstance gowasd.Instance
	if *description != "" {
		serviceInstance.Description = *description
		serviceInstance.Service = service
	}

	if len(args) == 0 {
		Fatal.Print("no command specified")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		if !fmtr.canOutputList() {
			Fatal.Printf("can't output list in %s output format", *format)
			os.Exit(1)
		}

		instances, err := sd.ServiceInstances(service)
		if err != nil {
			Fatal.Print("wasd error: ", err)
			os.Exit(1)
		}

		if len(instances) == 0 {
			Fatal.Print("no instances found")
			os.Exit(1)
		}

		var ri = make([]gowasd.InstanceResolution, 0, len(instances))
		for _, i := range instances {
			ri = append(ri, getResolvedInstance(sd, i))
		}

		fmtr.printResolvedInstances(ri)

	case "show":
		if serviceInstance.Description == "" {
			Fatal.Print("you must give a description if resolving an instance directly")
			os.Exit(1)
		}

		instance, err := sd.ResolveInstance(serviceInstance)
		if err != nil {
			Fatal.Print("wasd error: ", err)
			os.Exit(1)
		}

		fmtr.printResolvedInstance(instance)

	default:
		Fatal.Print("invalid command")
		os.Exit(1)
	}
}

func getFormatter() (out formatter, err error) {
	switch *format {
	case "default":
		out = terminalFormatter{output: Info}
	case "postgres_env":
		out = postgresEnvVarFormatter{output: Info}
	default:
		err = errors.New("invalid format given")
	}
	return
}

func getResolvedInstance(sd gowasd.Client, i gowasd.Instance) (ri gowasd.InstanceResolution) {
	r, err := sd.ResolveInstance(i)

	if err != nil {
		Fatal.Print("Can't resolve instance")
		os.Exit(1)
	}
	return r
}
