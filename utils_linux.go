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

func PrintTrashBoxItems() (ret error) {
	user, err := user.Current()
	if err != nil {
		fmt.Errorf("Failure to get user's home directory: %s", err)
		return err
	}

	// Contents of a trash directory
	// https://specifications.freedesktop.org/trash-spec/trashspec-1.0.html
	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)

	// Generate fullPath from .~/.local/share/Trash/files/
	allFiles, err := ioutil.ReadDir(trashBase + "/files/")
	if err != nil {
		fmt.Errorf("Failure to get files in ~/.local/share/Trash : %s", err)
		return err
	}

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
		fi, err := fFile.Stat()
		if err != nil {
			fmt.Printf("Failure to open file: %s\n", err)
		}

		fmt.Println()
		printDisplayName(filepath.Base(decodedFilePath), "FileName")
		printDisplayName(decodedFilePath, "Location")
		printDisplayName(deletedDate, "DeletedDate")
		printDisplayName(strconv.FormatInt(fi.Size(), 10), "Size\t")
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

type fileInfo struct {
	originalPath      string
	trashboxFilesPath string
	trashboxInfoPath  string
	dateDelete        string
	size              int64
}

func addMatchedFileList(fl []fileInfo, infoFilePath string, filesFilePath string) []fileInfo {
	iFile, err := os.Open(infoFilePath)
	if err != nil {
		fmt.Printf("Failure to open file: %s", err)
		return fl
	}
	defer iFile.Close()

	fFile, err := os.Open(filesFilePath)
	if err != nil {
		fmt.Printf("Failure to open file: %s", err)
		return fl
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
		fmt.Printf("Failure to open file: %s", err)
	}

	// assign value to fileInfo
	var fi fileInfo
	fi.originalPath = decodedFilePath
	fi.trashboxFilesPath = filesFilePath
	fi.trashboxInfoPath = infoFilePath
	fi.dateDelete = deletedDate
	fi.size = fs.Size()

	return append(fl, fi)
}

func unDelete(id uint, fl []fileInfo, outputPath string) (err error) {
	if uint(len(fl)) <= id {
		return fmt.Errorf("Index out of range")
	}
	var path string
	if len(outputPath) == 0 {
		// assign original location to 'path'
		path = fl[id].originalPath
	} else {
		path, _ = filepath.Abs(outputPath)
	}
	fmt.Printf("Restore %s → %s\n", fl[id].trashboxFilesPath, path)

	err = os.Rename(fl[id].trashboxFilesPath, path)
	if err != nil {
		return err
	}

	err = os.Remove(fl[id].trashboxInfoPath)
	if err != nil {
		return err
	}

	return nil
}

func RestoreItem(filename string, outputPath string) (ret error) {
	user, err := user.Current()
	if err != nil {
		return err
	}

	// Contents of a trash directory
	// https://specifications.freedesktop.org/trash-spec/trashspec-1.0.html
	trashBase := strings.Replace("~/.local/share/Trash", "~", user.HomeDir, 1)

	// Generate fullPath from .~/.local/share/Trash/files/
	allFiles, err := os.ReadDir(trashBase + "/files/")
	if err != nil {
		return err
	}

	var fl []fileInfo
	for _, file := range allFiles {
		if !strings.Contains(file.Name(), filename) {
			continue
		}

		infoFilePath := trashBase + "/info/" + file.Name() + ".trashinfo"
		filesFilePath := trashBase + "/files/" + file.Name()

		fl = addMatchedFileList(fl, infoFilePath, filesFilePath)
	}
	var id int
	if len(fl) > 1 {
		fmt.Println("ID\t DateDeleted\t\t FileSize\t Path")
		for i, v := range fl {
			fmt.Printf("%d\t %s\t %d\t\t %s\t\n", i, v.dateDelete, v.size, v.originalPath)
		}

		fmt.Printf("Which one do you want to restore[ID] ? > ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		id, _ = strconv.Atoi(scanner.Text())
		unDelete(uint(id), fl, outputPath)
	} else if len(fl) == 1 {
		unDelete(uint(0), fl, outputPath)
	} else {
		return fmt.Errorf("No such file or director")
	}

	return nil
}
