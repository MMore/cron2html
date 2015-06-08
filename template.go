package main

import "time"

func templateCreationTime() string {
	return time.Now().Format("2006-01-02 15:04 MST")
}

func templateVersion() string {
	return "v" + VERSION
}
