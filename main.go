package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

var (
	app               = kingpin.New("cronjoboverview", "Get an overview about all your cronjobs.")
	servers           = app.Arg("servers", "server").Required().Strings()
	ssh_user          = app.Flag("user", "SSH user for login").Short('u').Required().String()
	ssh_identity_file = app.Flag("identity", "SSH identity file for login").Short('i').Required().String()
	cron_user         = app.Flag("cron_user", "User of crontab").Short('c').Required().String()
)

func main() {
	app.Version("1.0.0")
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
