package main

import (
	"fmt"
	"os"

	"github.com/pborman/getopt/v2"
)

func main() {
	var (
		isHelp bool
	)

	getopt.Flag(&isHelp, 'h', "Show help")
	getopt.Parse()
	args := getopt.Args()

	if len(args) == 0 || isHelp {
		getopt.Usage()
		os.Exit(1)
	}

	for _, path := range args {
		ret := MoveToTrashBox(path)
		if ret != 0 {
			fmt.Println("go-trash: cannot move to trashbox: ", path)
		}
	}
}
