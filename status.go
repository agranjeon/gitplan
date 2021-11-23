package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
)

func Status() {
	files, err := ioutil.ReadDir(".gitplan/commits")
	if err != nil || len(files) == 0 {
		// Should never happen, but who knows
		color.Error.Println("I guess you don't have any commit yet huh")
		return
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Date", "Branch", "Message"})
	for _, file := range files {
		if !strings.Contains(file.Name(), ".info") {
			continue
		}
		content, err := os.ReadFile(".gitplan/commits/" + file.Name())
		if err != nil {
			return
		}
		fileContent := string(content)
		s := strings.Split(fileContent, "\n")
		date, branchName, message := humanDate(s[0]), s[1], s[2]

		t.AppendRow(table.Row{date, branchName, message})
	}
	t.Render()
}

func humanDate(date string) string {
	i, err := strconv.ParseInt(date, 10, 64)
	if err != nil {
		panic(err)
	}
	humanDate := time.Unix(i, 0)

	return humanDate.Format("2006-01-02 15:04")
}
