package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/alecthomas/kingpin.v2"
)

// How to do it?
// - `ssh -t user@server sudo crontab -l -u www-data`
// - <enter sudo password for all servers>
// - get the raw crontab for every server and persist it in a machine readable datastructure
//   - server
//     - schedule
//     - crontab user
//     - command
//     - opt. human readable description (via `echo "Human readbale" && command`)
// - create a wonderful readable view (html)

const TIMEOUT = 3

type CrontabPerServer struct {
	server     string
	rawCrontab string
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

var (
	app      = kingpin.New("cronjoboverview", "Get an overview about all your cronjobs.")
	servers  = app.Arg("servers", "server").Required().Strings()
	sshUser  = app.Flag("user", "SSH user for login").Short('u').Required().String()
	cronUser = app.Flag("cron_user", "User of crontab").Short('c').Required().String()
)

func main() {
	app.Version("1.0.0")
	kingpin.MustParse(app.Parse(os.Args[1:]))

	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		fmt.Println("error agent ", err)
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
			output := executeCmd("crontab -l", hostname, clientConfig)
			collector <- CrontabPerServer{server: hostname, rawCrontab: output}
		}(hostname)
	}

	go func() {
		for {
			select {
			case result := <-collector:
				fmt.Println("Result for " + result.server)
				fmt.Println(result.rawCrontab)
			case <-time.After(TIMEOUT * time.Second):
				fmt.Println("Timeout")
				os.Exit(1)
			}
		}
	}()

	var input string
	fmt.Scanln(&input)
}
