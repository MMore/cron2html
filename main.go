package main

import (
	"fmt"
	"html/template"
	"net"
	"os"
	"os/user"
	"sort"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/alecthomas/kingpin.v2"
)

const TIMEOUT = 5
const VERSION = "1.0.0"

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
