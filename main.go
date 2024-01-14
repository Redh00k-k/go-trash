package main

import (
	"fmt"
	"os"

	"github.com/pborman/getopt/v2"
)

func main() {
	var (
		isList       = false
		isHelp       = false
		undeleteFile = ""
		outputPath   = ""
	)

	getopt.Flag(&isList, 'l', "List trashed files")
	getopt.Flag(&isHelp, 'h', "Show help")
	getopt.Flag(&undeleteFile, 'u', "Restore files to original location", "File")
	getopt.Flag(&outputPath, 'o', "Output file to location", "File")
	getopt.Parse()
	args := getopt.Args()

	if len(undeleteFile) != 0 {
		err := RestoreItem(undeleteFile, outputPath)
		if err != nil {
			fmt.Println("go-trash: ", err)
		}
		os.Exit(0)
	}

	if isList == true {
		fmt.Println("")
		fmt.Println("# Trash Box #")
		err := PrintTrashBoxItems()
		if err != nil {
			fmt.Println("go-trash: ", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(args) == 0 || isHelp {
		getopt.Usage()
		os.Exit(1)
	}

	for _, path := range args {
		err := MoveToTrashBox(path)
		if err != nil {
			fmt.Println("go-trash: ", err)
		}
	}
}
