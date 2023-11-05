package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ~/.local/share/Trash/info/
type Info struct {
	path         string
	deletionDate time.Time
}

func parseLine(line string, delimiter string) []string {
	pl := strings.Split(line, delimiter)
	return pl
}

func decodeLine(pl []string) string {
	decodedFilePath, err := url.QueryUnescape(pl[1])
	if err != nil {
		fmt.Errorf("Failure to decode: %s", err)
		return ""
	}
	return decodedFilePath
}

func printDisplayName(line string, label string) {
	fmt.Printf("%s\t: %s\n", label, line)
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

	// Generate fullPath from .~/.local/share/Trash/files/
	allFiles, err := ioutil.ReadDir(trashBase + "/files/")
	if err != nil {
		fmt.Errorf("Failure to get files in ~/.local/share/Trash : %s", err)
		return 1
	}

	for _, file := range allFiles {
		infoFilePath := trashBase + "/info/" + file.Name() + ".trashinfo"
		filesFilePath := trashBase + "/files/" + file.Name()

		iFile, err := os.Open(infoFilePath)
		if err != nil {
			fmt.Printf("Failure to open file: %s", err)
			continue
		}
		defer iFile.Close()

		fFile, err := os.Open(filesFilePath)
		if err != nil {
			fmt.Printf("Failure to open file: %s", err)
			continue
		}
		defer fFile.Close()

		var decodedFilePath string
		var deletedDate string
		// Read one line at a time, as the order of 'Path' and 'DeletionDate' may be different
		scanner := bufio.NewScanner(iFile)
		for scanner.Scan() {
			line := scanner.Text()
			if pl := parseLine(line, "Path="); len(pl) > 1 {
				decodedFilePath, _ = url.QueryUnescape(pl[1])
			} else if pl := parseLine(line, "DeletionDate="); len(pl) > 1 {
				deletedDate = pl[1]
			} else {
				// "[Trash Info]"
				continue
			}
		}
		fi, err := fFile.Stat()
		if err != nil {
			fmt.Printf("Failure to open file: %s", err)
		}

		fmt.Println()
		printDisplayName(filepath.Base(decodedFilePath), "FileName")
		printDisplayName(decodedFilePath, "Location")
		printDisplayName(deletedDate, "DeletedDate")
		printDisplayName(strconv.FormatInt(fi.Size(), 10), "Size\t")
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

func RestoreItem(filename string) (ret int) {
	user, err := user.Current()
	if err != nil {
		fmt.Errorf("Failure to get user's home directory: %s", err)
		return 1
	}

	// Contents of a trash directory
	// https://specifications.freedesktop.org/trash-spec/trashspec-1.0.html
	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)

	// Generate fullPath from .~/.local/share/Trash/files/
	allFiles, err := os.ReadDir(trashBase + "/files/")
	if err != nil {
		fmt.Errorf("Failure to get files in ~/.local/share/Trash : %s", err)
		return 1
	}

	for _, file := range allFiles {
		if isMatch, _ := filepath.Match(filename, file.Name()); !isMatch {
			continue
		}

		infoFilePath := trashBase + "/info/" + file.Name() + ".trashinfo"
		filesFilePath := trashBase + "/files/" + file.Name()

		iFile, err := os.Open(infoFilePath)
		if err != nil {
			fmt.Printf("Failure to open file: %s", err)
			continue
		}
		defer iFile.Close()

		var decodedFilePath string
		// Read one line at a time, as the order of 'Path' and 'DeletionDate' may be different
		scanner := bufio.NewScanner(iFile)
		for scanner.Scan() {
			line := scanner.Text()
			if pl := parseLine(line, "Path="); len(pl) > 1 {
				decodedFilePath, _ = url.QueryUnescape(pl[1])
			} else {
				// "[Trash Info]"
				continue
			}
		}

		err = os.Rename(filesFilePath, decodedFilePath)
		if err != nil {
			fmt.Errorf("Failure to move file: %s", err)
			return 1
		}

		err = os.Remove(infoFilePath)
		if err != nil {
			fmt.Errorf("Failure to remove info file: %s", err)
			return 1
		}

		fmt.Printf("Restore\t: %s\n", decodedFilePath)
	}

	return 0
}
