# go-trash
The `go-trash` is simple CLI, that move files and folder to the trashbox written in Go.
Works on Linux and Windows 10.

## Usage
```
$ ./go-trash -h
Usage: go-trash [-hl] [-u File] [parameters ...]
 -h       Show help
 -l       List trashed files
 -u File  Restore files to original location
```

### Trash
* Windows
```
C:\Users\user\Desktop> go-trash.exe aaa.txt bbb_dir
```

* Linux
```
~$ ./go-trash aaa.txt bbb_dir
```

### Print list trashed files
* Windows
```
C:\Users\user\Desktop> go-trash.exe -l

# Trash Box #

InFolder        : aaa.txt
Normal          : C:\Users\user\Desktop\aaa.txt
ForParsing      : C:\$RECYCLE.BIN\S-xxx\$RABCD.txt
DateDeleted     : 2023/1/2 12:34:56
Size            : 1234

InFolder        : bbb_dir
Normal          : C:\Users\user\Desktop\bbb_dir
ForParsing      : C:\$RECYCLE.BIN\S-xxx\$R1C0U4Q
DateDeleted     : 2023/1/2 12:34:56
Size            : 0
```

* Linux
```
~$ ./go-trash -l

# Trash Box #

FileName        : aaa.txt
Location        : /home/user/aaa.txt
DeletedDate     : 2023-01-23T12:34:56
Size            : 1234

FileName        : bbb_dir
Location        : /home/user/bbb_dir
DeletedDate     : 2023-01-23T12:34:56
Size            : 0
```


### Restore files
* Windows
```
C:\Users\user\Desktop> go-trash.exe -u *.txt
Restore         : C:\Users\user\Desktop\aaa.txt
```

* Linux
```
~$ ./go-trash -u bbb_dir
Restore: /home/user/bbb_dir
```