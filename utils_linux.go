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

func GetTrashBoxItems() ([]fi, error) {
	user, err := user.Current()
	if err != nil {
		fmt.Errorf("Failure to get user's home directory: %s", err)
		return nil, err
	}

	// Contents of a trash directory
	// https://specifications.freedesktop.org/trash-spec/trashspec-1.0.html
	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)

	// Generate fullPath from .~/.local/share/Trash/files/
	allFiles, err := ioutil.ReadDir(trashBase + "/files/")
	if err != nil {
		fmt.Errorf("Failure to get files in ~/.local/share/Trash : %s", err)
		return nil, err
	}

	var files []fi
	for _, file := range allFiles {
		infoFilePath := trashBase + "/info/" + file.Name() + ".trashinfo"
		filesFilePath := trashBase + "/files/" + file.Name()

		iFile, err := os.Open(infoFilePath)
		if err != nil {
			fmt.Printf("Failure to open info file: %s\n", err)
			continue
		}
		defer iFile.Close()

		fFile, err := os.Open(filesFilePath)
		if err != nil {
			fmt.Printf("Failure to open files file: %s\n", err)
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

		fs, err := fFile.Stat()
		if err != nil {
			fmt.Printf("Failure to open file: %s\n", err)
		}

		var file fi

		file.filename = filepath.Base(decodedFilePath)
		file.location = decodedFilePath
		file.inTrashBox = filesFilePath
		file.dateDeleted, _ = time.Parse("2006-01-02T15:04:05Z07:00", deletedDate)
		file.size = fs.Size()
		files = append(files, file)
	}

	return files, nil
}

func printDisplayName(line string, label string) {
	fmt.Printf("%-12s: %s\n", label, line)
}

func PrintTrashBoxItems() (ret error) {
	files, err := GetTrashBoxItems()
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Println()
		printDisplayName(file.filename, "FileName")
		printDisplayName(file.location, "Location")
		printDisplayName(file.inTrashBox, "InTrashBox")
		printDisplayName(file.dateDeleted.Format("2006-01-02T15:04:05Z07:00"), "DateDeleted")
		printDisplayName(strconv.FormatInt(file.size, 10), "Size")
	}
	return nil
}

func convertTrashInfo(i Info) string {
	return fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", i.path, i.deletionDate.UTC().Format(time.RFC3339))
}

func MoveToTrashBox(path string) (err error) {
	filename := filepath.Base(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info := Info{abs, time.Now()}
	user, err := user.Current()
	if err != nil {
		return err
	}

	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)
	if _, err := os.Stat(trashBase + "/info/"); err != nil {
		os.MkdirAll(trashBase+"/info/", os.ModePerm)
	}

	if _, err := os.Stat(trashBase + "/files/"); err != nil {
		os.MkdirAll(trashBase+"/files/", os.ModePerm)
	}

	err = os.WriteFile(trashBase+"/info/"+filename+".trashinfo", []byte(convertTrashInfo(info)), os.ModePerm)
	if err != nil {
		return err
	}

	// May not be able to move files or directories between different partitions
	// Occur "Invalid cross-device link error"
	// https://stackoverflow.com/questions/42392600/oserror-errno-18-invalid-cross-device-link
	err = os.Rename(path, trashBase+"/files/"+filename)
	if err != nil {
		return err
	}

	return nil
}

func Undelete(srcPath string, dstPath string) (err error) {
	user, err := user.Current()
	if err != nil {
		return err
	}

	// Contents of a trash directory
	// https://specifications.freedesktop.org/trash-spec/trashspec-1.0.html
	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)
	infoFilePath := trashBase + "/info/" + filepath.Base(srcPath) + ".trashinfo"

	err = os.Rename(srcPath, dstPath)
	if err != nil {
		return err
	}

	// /info/ file is still in the trash box. So deleted it.
	err = os.Remove(infoFilePath)
	if err != nil {
		return err
	}

	return nil
}
