package main

import (
	"fmt"
	"os"
	"sort"
	"text/template"
	"time"
)

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

func templateCreationTime() string {
	return time.Now().Format("2006-01-02 15:04 MST")
}

func templateVersion() string {
	return "v" + VERSION
}
