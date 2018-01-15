package utils

import (
	"bufio"
	"io"
	"log"
	"os"
	s "strings"
)

type State int

const (
	NotInitiated State = iota
	Waiting
	Aborted
	Prepared
	Committed
)

func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func GetCohortCount() int {
	f, err := os.Open("/etc/hosts")
	FailOnError(err, "Failed to open hosts file")

	var line string
	var ioErr error
	cohortCount := 0
	r := bufio.NewReader(f)
	for ioErr != io.EOF {
		line, ioErr = r.ReadString('\n')
		if s.Contains(line, "192.168.10") && s.Contains(line, "cohort") {
			cohortCount += 1
		}
	}

	return cohortCount
}
