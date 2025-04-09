package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/censys/scan-takehome/cmd"

	"github.com/transientvariable/config-go"
)

const (
	exitCodeSuccess = iota
	exitCodeError
)

func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}

	rootCmd, err := cmd.New().ExecuteC()
	if cmdErr(err) {
		if !strings.HasSuffix(err.Error(), "\n") {
			fmt.Println()
		}
		fmt.Println(rootCmd.UsageString())
		os.Exit(exitCodeSuccess)
	}
	fmt.Println(err)
	os.Exit(exitCodeError)
}

func cmdErr(err error) bool {
	if err == nil {
		return false
	}

	keywords := []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
	}

	cause := err.Error()
	for _, k := range keywords {
		if strings.Contains(cause, k) {
			return true
		}
	}
	return false
}
