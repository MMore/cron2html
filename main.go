package main

import (
	"fmt"
	"html/template"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/kevinwallace/crontab"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/alecthomas/kingpin.v2"
)

// How to do it?
// - `ssh -t user@server sudo crontab -l -u www-data`
// - <enter sudo password for all servers>
// - get the raw crontab for every server and persist it in a machine readable datastructure
//   - server
//     crontab user
//     schedules:
//       - schedule
//       - command
//       - opt. human readable description (via `echo "Human readbale" && command`)
// - create a wonderful readable view (html)

const TIMEOUT = 5

type CrontabEntry struct {
	Schedule string
	Command  string
	NextRun  time.Time
}

type CrontabPerServer struct {
	Server     string
	User       string
	rawCrontab string
	Entries    []CrontabEntry
}

func executeCmd(cmd string, hostname string, config *ssh.ClientConfig) string {
	fmt.Println("[" + hostname + "] " + cmd)

	// TODO: remove this hack later
	var port string = "22"
	if hostname == "gate.barzahlen.de" {
		port = "88"
	}

	client, err := ssh.Dial("tcp", hostname+":"+port, config)
	if err != nil {
		fmt.Println("Failed to dial: ", err)
	}
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
	}
	defer session.Close()

	output, _ := session.Output(cmd)
	return string(output[:])
}

func (self *CrontabPerServer) parseEntries() {
	regexp := regexp.MustCompile(`(?m)^(@[a-z]+|([0-9\*\/\-\,]+ [0-9\*\/\-\,]+ [0-9\*\/\-\,\?LW]+ [0-9A-Z\*\/\-\,]+ [0-9A-Z\*\/\-\,\?L\#]+[ 0-9\*\/\-\,]*)) (.*)$`)
	result := regexp.FindAllStringSubmatch(self.rawCrontab, -1)

	if self.Entries == nil {
		self.Entries = []CrontabEntry{}
	}

	for _, x := range result {
		// 0: 0 3 * * 1 cmd
		// 1-2: 0 3 * * 1
		// 3: cmd
		if len(x) != 4 {
			fmt.Println("not detected crontab format")
			os.Exit(1)
		}

		var entryForTime crontab.Entry
		entryForTime, err := crontab.ParseEntry(x[1])
		if err != nil {
			fmt.Println("error parsing cron schedule")
			os.Exit(1)
		}
		var entry CrontabEntry = CrontabEntry{Schedule: x[1], Command: x[3], NextRun: entryForTime.Schedule.Next(time.Now())}
		// fmt.Println(entry)

		// fmt.Println("i ", i, ": ", entry)
		self.Entries = append(self.Entries, entry)
	}
}

func writeFile(outputFilename string, results *[]CrontabPerServer) {
	file, err := os.OpenFile(outputFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic("opening file failed: " + err.Error())
	}
	defer file.Close()

	templateFuncs := template.FuncMap{
		"creationTime": templateCreationTime,
		"version":      templateVersion,
	}
	template := template.Must(template.New("overview.tpl.html").Funcs(templateFuncs).ParseFiles("overview.tpl.html"))
	err = template.Execute(file, results)
	if err != nil {
		panic("writing file failed: " + err.Error())
	}
	fmt.Println("... wrote to file", outputFilename)
}

func templateCreationTime() string {
	return time.Now().Format("2006-01-02 15:04 MST")
}

func templateVersion() string {
	return "v1.0.0"
}

var (
	app            = kingpin.New("cronjoboverview", "Get an overview about all your cronjobs.")
	servers        = app.Arg("servers", "server").Required().Strings()
	sshUser        = app.Flag("user", "SSH user for login").Short('u').Required().String()
	cronUser       = app.Flag("cron_user", "User of crontab (default is the SSH user)").Short('c').String()
	outputFilename = app.Flag("output", "Filename of the output").Short('o').Default("output.html").String()
)

func main() {
	app.Version(templateVersion())
	kingpin.MustParse(app.Parse(os.Args[1:]))

	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		fmt.Println("error agent ", err)
		os.Exit(1)
	}
	defer conn.Close()
	ag := agent.NewClient(conn)

	clientConfig := &ssh.ClientConfig{
		User: *sshUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(ag.Signers),
		},
	}

	collector := make(chan CrontabPerServer)
	done := make(chan bool)

	for _, hostname := range *servers {
		fmt.Println("Collecting from " + hostname + "...")
		go func(server string) {
			cmd := "crontab -l"
			if len(*cronUser) > 0 && *sshUser != *cronUser {
				cmd = "sudo " + cmd + " -u " + *cronUser
			} else {
				*cronUser = *sshUser
			}

			output := executeCmd(cmd, server, clientConfig)
			collector <- CrontabPerServer{Server: server, User: *cronUser, rawCrontab: output}
		}(hostname)
	}

	go func() {
		results := []CrontabPerServer{}
		for {
			select {
			case result, more := <-collector:
				if !more {
					done <- true
					return
				}
				result.parseEntries()
				results = append(results, result)
				fmt.Println("... collected crontab with", len(result.Entries), "entries from", result.Server)

				if len(results) == len(*servers) {
					writeFile(*outputFilename, &results)
					close(collector)
				}
			case <-time.After(TIMEOUT * time.Second):
				writeFile(*outputFilename, &results)
				fmt.Println()
				fmt.Println("... Timeout!")
				os.Exit(1)
			}
		}
	}()

	<-done
}
