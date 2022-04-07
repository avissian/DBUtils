package main

import (
	"os"

	"github.com/pterm/pterm"
)

func dieOnError(msg string, err error) {
	if err != nil {
		pterm.FgRed.Println(msg, err)
		os.Exit(1)
	}
}
