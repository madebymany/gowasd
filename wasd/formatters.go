package main

import (
	"bytes"
	"fmt"
	"github.com/madebymany/gowasd"
	"log"
	"strconv"
	"strings"
)

type formatter interface {
	printResolvedInstances([]gowasd.InstanceResolution)
	printResolvedInstance(gowasd.InstanceResolution)
	canOutputList() bool
}

type terminalFormatter struct {
	output *log.Logger
}

type postgresEnvVarFormatter struct {
	output *log.Logger
}

var postgresPropertyNames []string = []string{"database", "user", "password", "passfile", "service", "servicefile", "realm", "options", "appname", "sslmode", "requiressl", "sslcompression", "sslcert", "sslkey", "sslrootcert", "sslcrl", "requirepeer", "krbsrvname", "gsslib", "connect_timeout", "clientencoding", "datestyle", "tz", "geqo", "sysconfdir", "localedir"}

func (self terminalFormatter) printResolvedInstances(instances []gowasd.InstanceResolution) {
	for n, i := range instances {
		self.printResolvedInstance(i)
		if n < len(instances)-1 {
			self.output.Println("")
		}
	}
}

func (self terminalFormatter) printResolvedInstance(i gowasd.InstanceResolution) {

	for _, e := range i.Targets {
		port := strconv.Itoa(e.Port)
		self.output.Print(fmt.Sprintf("⌁  %s\t%s\t%-s", i.DnsName(), e.Host, port))
	}

	for version, r := range i.Properties {
		var fields = make([][]string, len(i.Properties[version]))

		var count = 0
		for k, v := range r {
			fields[count] = []string{strconv.Itoa(version), k, v}
			count++
		}
		self.output.Print(formatTable(fields, "✎  "))
	}

	return
}

func (self terminalFormatter) canOutputList() bool {
	return true
}

func (self postgresEnvVarFormatter) printResolvedInstances(instances []gowasd.InstanceResolution) {
	panic("unreachable")
}

func (self postgresEnvVarFormatter) printEnvVar(k, v string) {
	v = strings.Replace(v, "'", "'\\''", -1)
	self.output.Printf("export PG%s='%s'", strings.ToUpper(k), v)
}

func (self postgresEnvVarFormatter) printResolvedInstance(i gowasd.InstanceResolution) {
	// XXX: no support for choosing a non-primary endpoint
	ep := i.Targets[0]
	self.printEnvVar("host", ep.Host)
	self.printEnvVar("port", strconv.Itoa(ep.Port))

	/* As far as I can see, the only inconsistency between environment variable
	 * and "connection parameter" names is in dbname/database. So I'll cover
	 * that one here.
	 */
	if v, ok := i.Properties[*version]["dbname"]; ok {
		self.printEnvVar("database", v)
	}

	for _, n := range postgresPropertyNames {
		if v, ok := i.Properties[*version][n]; ok {
			self.printEnvVar(n, v)
		}
	}
}

func (self postgresEnvVarFormatter) canOutputList() bool {
	return false
}

func formatTable(fields [][]string, linePrefix string) (out string) {
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
		outBuf.WriteString(linePrefix)
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
