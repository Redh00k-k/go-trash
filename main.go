package main

import (
	"fmt"
	"os"

	"github.com/pborman/getopt/v2"
)

func main() {
	var (
		isList bool
		isHelp bool
	)

	getopt.Flag(&isList, 'l', "List trashed files")
	getopt.Flag(&isHelp, 'h', "Show help")
	getopt.Parse()
	args := getopt.Args()

	if isList == true {
		fmt.Println("")
		fmt.Println("# Trash Box #")
		if PrintTrashBoxItems() != 0 {
			fmt.Println("go-trash: cannot print items in trashbox")
			os.Exit(1)
		}
		os.Exit(0)
	}

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
