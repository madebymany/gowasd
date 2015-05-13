package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/madebymany/gowasd"
	"github.com/miekg/dns"
	"log"
	"os"
	"strconv"
)

var (
	name = flag.String("name", "",
		"The name of the service you wish to query")
	protocol = flag.String("protocol", "tcp",
		"The protocol of the service, either udp or tcp")
	version = flag.Int("version", 1,
		"The version of the properties you want to load")
	domain = flag.String("domain", "",
		"The top level domain you wish to query the service against")
	args              []string
	argNum            int = 0
	commands          []string
	availableVersions []int

	// setup log levels
	Info  *log.Logger
	Fatal *log.Logger
)

func main() {
	Info = log.New(os.Stdout, "", 0)
	Fatal = log.New(os.Stderr, "Error: ", 0)

	flag.Parse()
	args = flag.Args()

	sd, err := gowasd.New(new(dns.Client), "")
	if err != nil {
		Fatal.Print("gowasd error: ", err)
		os.Exit(1)
	}
	s := gowasd.Service{
		Name:     *name,
		Protocol: *protocol,
		Domain:   *domain,
	}

	instances, err := sd.ServiceInstances(s)
	if err != nil {
		Fatal.Print("wasd error: ", err)
		os.Exit(1)
	}

	if len(instances) > 0 {
		runCommand(sd, args, instances)
	} else {
		Fatal.Print("no instances found")
		os.Exit(1)
	}
}

func getResolvedInstance(sd gowasd.Client, i gowasd.Instance) (ri gowasd.InstanceResolution) {
	r, err := sd.ResolveInstance(i)

	if err != nil {
		Fatal.Print("Can't resolve instance")
		os.Exit(1)
	}
	return r
}

func runCommand(sd gowasd.Client, commands []string, instances []gowasd.Instance) {
	if len(commands) == 0 {
		Fatal.Print("no command specified")
		os.Exit(1)
	}
	var subCommand bool = false
	if len(commands) > 1 {
		subCommand = true
	}
	switch commands[0] {
	case "list":
		var ri = make([]gowasd.InstanceResolution, len(instances))
		for _, i := range instances {
			ri = append(ri, getResolvedInstance(sd, i))
		}
		setVersions(ri)

		if !versionExists(*version) {
			Fatal.Print("version doesn't exist")
			os.Exit(1)
		}

		if subCommand {
			switch commands[1] {
			case "versions":
				printVersions()
			case "targets":
				printTargets(ri)
			case "properties":
				printProperties(ri, *version)
			}
		} else {
			printResolvedInstance(ri)
		}
	}

}

func versionExists(version int) (exists bool) {
	exists = false
	for _, v := range availableVersions {
		if v == version {
			exists = true
			return
		}
	}
	return
}

func setVersions(instances []gowasd.InstanceResolution) {
	for _, i := range instances {
		for count, _ := range i.Properties {
			availableVersions = append(availableVersions, count)
		}
	}
}

func printVersions() {
	for _, c := range availableVersions {
		Info.Println(c)
	}
}

func printProperties(instances []gowasd.InstanceResolution, version int) {
	for _, i := range instances {
		var fields = make([][]string, len(i.Properties[version]))
		var count = 0
		for k, v := range i.Properties[version] {
			fields[count] = []string{k, v}
			count++
		}
		Info.Print(formatTable(fields))
	}
}

func printTargets(instances []gowasd.InstanceResolution) {
	for _, i := range instances {
		var fields = make([][]string, len(i.Targets))
		for count, t := range i.Targets {
			fields[count] = []string{t.Host, strconv.Itoa(t.Port)}
		}
		Info.Print(formatTable(fields))
	}
}

func printResolvedInstance(instances []gowasd.InstanceResolution) {

	for _, i := range instances {
		for _, e := range i.Targets {
			port := strconv.Itoa(e.Port)
			formatString := "%s    %-s"
			Info.Print(fmt.Sprintf(formatString, e.Host, port))
		}

		for j, r := range i.Properties {
			var fields = make([][]string, len(i.Properties[j]))

			var count = 0
			for k, v := range r {
				fields[count] = []string{k, v}
				count++
			}
			Info.Print(formatTable(fields))
		}
	}

	return
}

func getNextArg(errMsg string) (val string) {
	if len(args) >= (argNum + 1) {
		val = args[argNum]
		argNum += 1
	} else if errMsg != "" {
		Fatal.Print(errMsg)
	}
	return
}

func formatTable(fields [][]string) (out string) {
	if len(fields) == 0 {
		return
	}
	outBuf := new(bytes.Buffer)
	numFields := len(fields[0])
	maxIndex := numFields - 1
	maxWidths := make([]int, numFields)
	for _, f := range fields {
		for i, c := range f {
			if lenc := len(c); lenc > maxWidths[i] {
				maxWidths[i] = lenc
			}
		}
	}

	for _, f := range fields {
		for i := 0; i < numFields; i++ {
			c := f[i]
			outBuf.WriteString(c)
			if i < maxIndex {
				outBuf.Write(
					bytes.Repeat([]byte(" "), maxWidths[i]-len(c)+2))
			}
		}
		outBuf.WriteRune('\n')
	}

	out = outBuf.String()

	return
}
