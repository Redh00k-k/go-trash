package main

import (
	"fmt"
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
