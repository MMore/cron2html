package main

import (
	"fmt"
	"html/template"
	"net"
	"os"
	"os/user"
	"regexp"
	"sort"
	"time"

	"github.com/kevinwallace/crontab"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/alecthomas/kingpin.v2"
)

const TIMEOUT = 5

type CrontabEntry struct {
	Schedule string
	Command  string
	NextRun  time.Time
}

type ServerCrontab struct {
	Server     string
	User       string
	rawCrontab string
	Entries    []CrontabEntry
}

type ServerCrontabs []ServerCrontab

func (self ServerCrontabs) Len() int {
	return len(self)
}

func (self ServerCrontabs) Less(i, j int) bool {
	return self[i].Server < self[j].Server
}

func (self ServerCrontabs) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func formatRemoteCallLog(hostname, msg string) string {
	return fmt.Sprintf("[%s] %s", hostname, msg)
}

func executeCmd(cmd string, hostname string, config *ssh.ClientConfig) string {
	fmt.Println(formatRemoteCallLog(hostname, "executing command '"+cmd+"'"))

	client, err := ssh.Dial("tcp", hostname+":22", config)
	if err != nil {
		halt(formatRemoteCallLog(hostname, "failed to connect: "+err.Error()))
	}
	session, err := client.NewSession()
	if err != nil {
		halt(formatRemoteCallLog(hostname, "failed to create session: "+err.Error()))
	}
	defer session.Close()

	output, err := session.Output(cmd)
	if err != nil {
		halt(formatRemoteCallLog(hostname, "failed to execute command '"+cmd+"': "+err.Error()))
	}
	return string(output[:])
}

func (self *ServerCrontab) parseEntries() {
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
			halt("crontab format could not be detected")
		}

		var entryForTime crontab.Entry
		entryForTime, err := crontab.ParseEntry(x[1])
		if err != nil {
			halt("parsing cron schedule failed")
		}
		var entry CrontabEntry = CrontabEntry{Schedule: x[1], Command: x[3], NextRun: entryForTime.Schedule.Next(time.Now())}

		self.Entries = append(self.Entries, entry)
	}
}

func halt(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func writeFile(outputFilename string, results *ServerCrontabs) {
	file, err := os.OpenFile(outputFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		halt("opening file failed: " + err.Error())
	}
	defer file.Close()

	templateFuncs := template.FuncMap{
		"creationTime": templateCreationTime,
		"version":      templateVersion,
	}
	template := template.Must(template.New("overview.tpl.html").Funcs(templateFuncs).ParseFiles("overview.tpl.html"))
	sort.Sort(results)
	err = template.Execute(file, results)
	if err != nil {
		halt("writing file failed: " + err.Error())
	}
	fmt.Println("... wrote to file", outputFilename)
}

func getCurrentUser() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.Username
}

func templateCreationTime() string {
	return time.Now().Format("2006-01-02 15:04 MST")
}

func templateVersion() string {
	return "v1.0.0"
}

var (
	app            = kingpin.New("cron2html", "Get an overview about all your cronjobs.")
	servers        = app.Arg("servers", "server").Required().Strings()
	sshUser        = app.Flag("user", "SSH user for login").Short('u').Default(getCurrentUser()).String()
	cronUser       = app.Flag("cron-user", "User of crontab (default is the SSH user)").Short('c').String()
	outputFilename = app.Flag("output", "Filename of the output").Short('o').Default("output.html").String()
)

func main() {
	app.Version(templateVersion())
	kingpin.MustParse(app.Parse(os.Args[1:]))

	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		halt("failed connecting to ssh agent: " + err.Error())
	}
	defer conn.Close()
	ag := agent.NewClient(conn)

	clientConfig := &ssh.ClientConfig{
		User: *sshUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(ag.Signers),
		},
	}

	collector := make(chan ServerCrontab)
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
			collector <- ServerCrontab{Server: server, User: *cronUser, rawCrontab: output}
		}(hostname)
	}

	go func() {
		results := ServerCrontabs{}
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
				halt("... Timeout!")
			}
		}
	}()

	<-done
}
