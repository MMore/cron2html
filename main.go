package main

import (
	"fmt"
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

const TIMEOUT = 3

type CrontabEntry struct {
	schedule string
	command  string
	nextRun  time.Time
}

type CrontabPerServer struct {
	server     string
	user       string
	rawCrontab string
	entries    []CrontabEntry
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

	if self.entries == nil {
		self.entries = make([]CrontabEntry, 0)
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
		var entry CrontabEntry = CrontabEntry{schedule: x[1], command: x[3], nextRun: entryForTime.Schedule.Next(time.Now())}
		// fmt.Println(entry)

		// fmt.Println("i ", i, ": ", entry)
		self.entries = append(self.entries, entry)
	}
}

var (
	app      = kingpin.New("cronjoboverview", "Get an overview about all your cronjobs.")
	servers  = app.Arg("servers", "server").Required().Strings()
	sshUser  = app.Flag("user", "SSH user for login").Short('u').Required().String()
	cronUser = app.Flag("cron_user", "User of crontab (default is the SSH user)").Short('c').String()
)

func main() {
	app.Version("1.0.0")
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

	for _, hostname := range *servers {
		fmt.Println("Collecting from " + hostname + "...")
		go func(server string) {
			cmd := "crontab -l"
			if len(*cronUser) > 0 && *sshUser != *cronUser {
				cmd = "sudo " + cmd + " -u " + *cronUser
			} else {
				*cronUser = *sshUser
			}

			output := executeCmd(cmd, hostname, clientConfig)
			collector <- CrontabPerServer{server: hostname, user: *cronUser, rawCrontab: output}
		}(hostname)
	}

	go func() {
		for {
			select {
			case result := <-collector:

				result.parseEntries()

				fmt.Println("Result for " + result.server + " (" + result.user + ")")
				for _, entry := range result.entries {
					fmt.Println("ENTRY")
					fmt.Println(entry.schedule)
					fmt.Println(entry.command)
					fmt.Println(entry.nextRun.Format(time.RFC3339))
				}
			case <-time.After(TIMEOUT * time.Second):
				fmt.Println()
				fmt.Println("Timeout")
				os.Exit(1)
			}
		}
	}()

	var input string
	fmt.Scanln(&input)
}
