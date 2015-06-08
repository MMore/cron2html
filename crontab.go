package main

import "regexp"

type CrontabEntry struct {
	Schedule string
	Command  string
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

		entry := CrontabEntry{Schedule: x[1], Command: x[3]}

		self.Entries = append(self.Entries, entry)
	}
}
