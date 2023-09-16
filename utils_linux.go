package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// ~/.local/share/Trash/info/
type Info struct {
	path         string
	deletionDate time.Time
}

func PrintTrashBoxItems() (ret int) {
	user, err := user.Current()
	if err != nil {
		fmt.Errorf("Failure to get user's home directory: %s", err)
		return 1
	}

	// Contents of a trash directory
	// https://specifications.freedesktop.org/trash-spec/trashspec-1.0.html
	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)
	files, err := filepath.Glob(trashBase + "/info/*")
	if err != nil {
		return 1
	}

	for _, fullpath := range files {
		f, err := os.Open(fullpath)
		if err != nil {
			fmt.Printf("Failure to open file: %s", err)
			continue
		}
		defer f.Close()

		// Read one line at a time, as the order of 'Path' and 'DeletionDate' may be different
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			pl := strings.Split(line, "Path=")
			if len(pl) < 2 {
				continue
			}

			decodedFilePath, err := url.QueryUnescape(pl[1])
			if err != nil {
				fmt.Errorf("Failure to decode: %s", err)
				break
			}
			fmt.Println(decodedFilePath)
			break
		}
	}

	return 0
}

func convertTrashInfo(i Info) string {
	return fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", i.path, i.deletionDate.UTC().Format(time.RFC3339))
}

func MoveToTrashBox(path string) (ret int) {
	filename := filepath.Base(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		fmt.Errorf("Failure to get absolute representation of path: %s", err)
		return 1
	}

	info := Info{abs, time.Now()}
	user, err := user.Current()
	if err != nil {
		fmt.Errorf("Failure to get user's home directory: %s", err)
		return 1
	}

	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)
	err = os.WriteFile(trashBase+"/info/"+filename+".trashinfo", []byte(convertTrashInfo(info)), os.ModePerm)
	if err != nil {
		fmt.Errorf("Failure to writeFile source file: %s", err)
		return 1
	}

	// May not be able to move files or directories between different partitions
	// Occur "Invalid cross-device link error"
	// https://stackoverflow.com/questions/42392600/oserror-errno-18-invalid-cross-device-link
	err = os.Rename(path, trashBase+"/files/"+filename)
	if err != nil {
		fmt.Errorf("Failure to rename source file: %s", err)
		return 1
	}

	return 0
}
