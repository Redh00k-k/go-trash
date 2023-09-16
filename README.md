# go-trash
The `go-trash` is simple CLI, that move files and folder to the trashbox written in Go.
Works on Linux and Windows 10.

## Usage
```
$ go-trash -h
Usage: go-trash.exe [-hl] [parameters ...]
 -h    Show help
 -l    List trashed files
```

### Trash
* Windows
```
C:\Users\user\Desktop> trash aaa.txt bbb_dir
```

* Linux
```
~$ go-trash aaa.txt bbb_dir
```

### Print list trashed files
* Windows
```
C:\Users\user\Desktop> trash -l

# Trash Box #
C:\Users\user\Desktop\aaa.txt
C:\Users\user\Desktop\bbb_dir
```

* Linux
```
~$ go-trash -l
/home/user/aaa.txt
/home/user/bbb_dir
```