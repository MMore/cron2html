package main

import (
	"reflect"
	"testing"
)

func TestParseEntries(t *testing.T) {
	rawCrontab := `
0 3 * * 1 uname -a
30 3 * * 1 bundle exec rake RAILS_ENV=test break:rules
10 13-15 * * 1-5 ./do.sh >> /var/log/x.log
10 13-15 * * 1-5 2015 cd /tmp && ./do.sh
@daily uname -a
*/15 4-16 * * 6,7 ./do.sh
`
	crontab := ServerCrontab{
		Server:     "a.example.com",
		User:       "dex",
		rawCrontab: rawCrontab,
	}

	cron1 := CrontabEntry{Schedule: "0 3 * * 1", Command: "uname -a"}
	cron2 := CrontabEntry{Schedule: "30 3 * * 1", Command: "bundle exec rake RAILS_ENV=test break:rules"}
	cron3 := CrontabEntry{Schedule: "10 13-15 * * 1-5", Command: "./do.sh >> /var/log/x.log"}
	cron4 := CrontabEntry{Schedule: "10 13-15 * * 1-5 2015", Command: "cd /tmp && ./do.sh"}
	cron5 := CrontabEntry{Schedule: "@daily", Command: "uname -a"}
	cron6 := CrontabEntry{Schedule: "*/15 4-16 * * 6,7", Command: "./do.sh"}
	cronTabEntries := []CrontabEntry{cron1, cron2, cron3, cron4, cron5, cron6}

	crontab.parseEntries()

	if !reflect.DeepEqual(crontab.Entries, cronTabEntries) {
		t.Errorf("Crontab parsing not correct, got: %#v", crontab.Entries)
	}

}
