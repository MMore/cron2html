package main

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/alecthomas/kingpin.v2"
)

const TIMEOUT = 3
const VERSION = "1.0.0"

func executeCmd(cmd string, hostport string, config *ssh.ClientConfig) string {
	fmt.Println(formatRemoteCallLog(hostport, "executing command '"+cmd+"'"))

	if !strings.Contains(hostport, ":") {
		hostport = hostport + ":22"
	}

	client, err := ssh.Dial("tcp", hostport, config)
	if err != nil {
		halt(formatRemoteCallLog(hostport, "failed to connect: "+err.Error()))
	}
	session, err := client.NewSession()
	if err != nil {
		halt(formatRemoteCallLog(hostport, "failed to create session: "+err.Error()))
	}
	defer session.Close()

	output, err := session.Output(cmd)
	if err != nil {
		fmt.Println(formatRemoteCallLog(hostport, "failed (empty crontab?): "+err.Error()))
	}
	return string(output[:])
}

func formatRemoteCallLog(hostname, msg string) string {
	return fmt.Sprintf("[%s] %s", hostname, msg)
}

func halt(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func getCurrentUser() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.Username
}

var (
	app              = kingpin.New("cron2html", "Get an overview about all your cronjobs.")
	servers          = app.Arg("servers", "server (define custom SSH port by adding ':<PORT>')").Required().Strings()
	sshUser          = app.Flag("user", "SSH user for login").Short('u').Default(getCurrentUser()).String()
	cronUser         = app.Flag("cron-user", "User of crontab (default is the SSH user)").Short('c').String()
	outputFilename   = app.Flag("output", "Filename of the output").Short('o').Default("output.html").String()
	omitEmptyServers = app.Flag("omit-empty", "Omit servers with empty crontabs").Bool()
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
		skipped := 0
		for {
			select {
			case result, more := <-collector:
				if !more {
					done <- true
					return
				}
				result.parseEntries()

				if !*omitEmptyServers || (*omitEmptyServers && len(result.Entries) > 0) {
					results = append(results, result)
					fmt.Println("... collected crontab with", len(result.Entries), "entries from", result.Server)
				} else {
					skipped++
					fmt.Println("... skipped empty server", result.Server)
				}

				if len(results)+skipped == len(*servers) {
					if len(results) > 0 {
						writeFile(*outputFilename, &results)
					} else {
						halt("... stopped, nothing to save")
					}

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
