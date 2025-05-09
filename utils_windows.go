package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shfileoperationw
var (
	modshell32                     = syscall.NewLazyDLL("Shell32.dll")
	procSHFileOperationW           = modshell32.NewProc("SHFileOperationW")
	procSHGetDesktopFolder         = modshell32.NewProc("SHGetDesktopFolder")
	procSHGetSpecialFolderLocation = modshell32.NewProc("SHGetSpecialFolderLocation")
	procSHGetDisplayName           = modshell32.NewProc("SHGetDisplayName")
	procSHGetDataFromIDList        = modshell32.NewProc("SHGetDataFromIDListW")

	modole32           = syscall.NewLazyDLL("ole32.dll")
	procCoInitialize   = modole32.NewProc("CoInitialize")
	procCoUninitialize = modole32.NewProc("CoUninitialize")
	procCoTaskMemFree  = modole32.NewProc("CoTaskMemFree")

	moduser32            = syscall.NewLazyDLL("User32.dll")
	procCreatePopupMenu  = moduser32.NewProc("CreatePopupMenu")
	procGetMenuItemCount = moduser32.NewProc("GetMenuItemCount")
)

// https://pinvoke.net/default.aspx/Enums/FileFuncFlags.html
const (
	FO_MOVE   = 0x1
	FO_COPY   = 0x2
	FO_DELETE = 0x3
	FO_RENAME = 0x4
)

// https://learn.microsoft.com/ja-jp/windows/win32/api/winuser/nf-winuser-showwindow
const (
	SW_HIDE            = 0x00
	SW_SHOWNORMAL      = 0x01
	SW_NORMAL          = 0x01
	SW_SHOWMINIMIZED   = 0x02
	SW_SHOWMAXIMIZED   = 0x03
	SW_MAXIMIZE        = 0x03
	SW_SHOWNOACTIVATE  = 0x04
	SW_SHOW            = 0x05
	SW_MINIMIZE        = 0x06
	SW_SHOWMINNOACTIVE = 0x07
	SW_SHOWNA          = 0x08
	SW_RESTORE         = 0x09
	SW_SHOWDEFAULT     = 0x10
	SW_FORCEMINIMIZE   = 0x11
	SW_MAX             = 0x11
)

// https://www.pinvoke.net/default.aspx/Enums/FILEOP_FLAGS.html
// https://groups.google.com/g/microsoft.public.vb.winapi/c/htqEx2zQjGo
const (
	FOF_MULTIDESTFILES        = 0x1
	FOF_CONFIRMMOUSE          = 0x2
	FOF_SILENT                = 0x4
	FOF_RENAMEONCOLLISION     = 0x8
	FOF_NOCONFIRMATION        = 0x10
	FOF_WANTMAPPINGHANDLE     = 0x20
	FOF_ALLOWUNDO             = 0x40
	FOF_FILESONLY             = 0x80
	FOF_SIMPLEPROGRESS        = 0x100
	FOF_NOCONFIRMMKDIR        = 0x200
	FOF_NOERRORUI             = 0x400
	FOF_NOCOPYSECURITYATTRIBS = 0x800
	FOF_NORECURSION           = 0x1000
	FOF_NO_CONNECTED_ELEMENTS = 0x2000
	FOF_WANTNUKEWARNING       = 0x4000
	FOF_NORECURSEREPARSE      = 0x8000
	FOF_NO_UI                 = FOF_SILENT | FOF_NOCONFIRMATION | FOF_NOERRORUI | FOF_NOCONFIRMMKDIR
)

const (
	FORMAT_MESSAGE_ALLOCATE_BUFFER = 0x00000100
	FORMAT_MESSAGE_ARGUMENT_ARRAY  = 0x00002000
	FORMAT_MESSAGE_FROM_HMODULE    = 0x00000800
	FORMAT_MESSAGE_FROM_STRING     = 0x00000400
	FORMAT_MESSAGE_FROM_SYSTEM     = 0x00001000
	FORMAT_MESSAGE_IGNORE_INSERTS  = 0x00000200
)

// https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shfileopstructa
type SHFILEOPSTRUCT struct {
	Hwnd                 uintptr
	Func                 uint32
	From                 *uint16
	To                   *uint16
	Flags                uint16
	AnyOperationsAborted int32
	NameMappings         *byte
	ProgressTitle        *uint16
}

type STRRET struct {
	UType uint32
	STRRET_Anonymous
}

type STRRET_Anonymous struct {
	Data [32]uint64
}

type FILE_ATTRIBUTE uint32

const (
	FILE_ATTRIBUTE_INVALID               FILE_ATTRIBUTE = 0xffff_ffff // -1
	FILE_ATTRIBUTE_READONLY              FILE_ATTRIBUTE = 0x0000_0001
	FILE_ATTRIBUTE_HIDDEN                FILE_ATTRIBUTE = 0x0000_0002
	FILE_ATTRIBUTE_SYSTEM                FILE_ATTRIBUTE = 0x0000_0004
	FILE_ATTRIBUTE_DIRECTORY             FILE_ATTRIBUTE = 0x0000_0010
	FILE_ATTRIBUTE_ARCHIVE               FILE_ATTRIBUTE = 0x0000_0020
	FILE_ATTRIBUTE_DEVICE                FILE_ATTRIBUTE = 0x0000_0040
	FILE_ATTRIBUTE_NORMAL                FILE_ATTRIBUTE = 0x0000_0080
	FILE_ATTRIBUTE_TEMPORARY             FILE_ATTRIBUTE = 0x0000_0100
	FILE_ATTRIBUTE_SPARSE_FILE           FILE_ATTRIBUTE = 0x0000_0200
	FILE_ATTRIBUTE_REPARSE_POINT         FILE_ATTRIBUTE = 0x0000_0400
	FILE_ATTRIBUTE_COMPRESSED            FILE_ATTRIBUTE = 0x0000_0800
	FILE_ATTRIBUTE_OFFLINE               FILE_ATTRIBUTE = 0x0000_1000
	FILE_ATTRIBUTE_NOT_CONTENT_INDEXED   FILE_ATTRIBUTE = 0x0000_2000
	FILE_ATTRIBUTE_ENCRYPTED             FILE_ATTRIBUTE = 0x0000_4000
	FILE_ATTRIBUTE_INTEGRITY_STREAM      FILE_ATTRIBUTE = 0x0000_8000
	FILE_ATTRIBUTE_VIRTUAL               FILE_ATTRIBUTE = 0x0001_0000
	FILE_ATTRIBUTE_NO_SCRUB_DATA         FILE_ATTRIBUTE = 0x0002_0000
	FILE_ATTRIBUTE_EA                    FILE_ATTRIBUTE = 0x0004_0000
	FILE_ATTRIBUTE_PINNED                FILE_ATTRIBUTE = 0x0008_0000
	FILE_ATTRIBUTE_UNPINNED              FILE_ATTRIBUTE = 0x0010_0000
	FILE_ATTRIBUTE_RECALL_ON_OPEN        FILE_ATTRIBUTE = 0x0004_0000
	FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS FILE_ATTRIBUTE = 0x0040_0000
)

// https://docs.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime
type FILETIME struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
}

// https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-win32_find_dataw
type WIN32_FIND_DATA struct {
	DwFileAttributes    FILE_ATTRIBUTE
	FtCreationTime      FILETIME
	FtLastAccessTime    FILETIME
	FtLastWriteTime     FILETIME
	NFileSizeHigh       uint32
	NFileSizeLow        uint32
	dwReserved0         uint32
	dwReserved1         uint32
	cFileName           [260]uint16 // MAX_PATH
	cCAlternateFileName [14]uint16
	DwFileType          uint32
	DwCreatorType       uint32
	WFinderFlags        uint16
}

const (
	CSIDL_BITBUCKET     = 0xa
	SHGDN_NORMAL        = 0x0000
	SHGDN_INFOLDER      = 0x1
	SHGDN_FOREDITING    = 0x1000
	SHGDN_FORADDRESSBAR = 0x4000
	SHGDN_FORPARSING    = 0x8000
)

type IShellFolderVtbl struct {
	QueryInterface   uintptr
	AddRef           uintptr
	Release          uintptr
	ParseDisplayName uintptr
	EnumObjects      uintptr
	BindToObject     uintptr
	BindToStorage    uintptr
	CompareIDs       uintptr
	CreateViewObject uintptr
	GetAttributesOf  uintptr
	GetUIObjectOf    uintptr
	GetDisplayNameOf uintptr
	SetNameOf        uintptr
}

type IShellFolder struct {
	lpVtbl *IShellFolderVtbl
}

type IEnumIDListVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Next           uintptr
	Skip           uintptr
	Reset          uintptr
	Clone          uintptr
}

type IEnumIDList struct {
	lpVtbl *IEnumIDListVtbl
}

const (
	SHCONTF_FOLDERS    = 0x0020
	SHCONTF_NONFOLDERS = 0x0040
)

type IID_IShellFolder struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// https://docs.microsoft.com/en-us/windows/win32/api/shtypes/ns-shtypes-itemidlist
type ITEMIDLIST struct {
	ID SHITEMID
}

// https://docs.microsoft.com/en-us/windows/win32/api/shtypes/ns-shtypes-shitemid
type SHITEMID struct {
	CB   uint16
	ABID [1]byte
}

func CoTaskMemFree(pv uintptr) {
	modole32.NewProc("CoTaskMemFree").Call(pv)
}

func (this *STRRET) pOleStr() **uint16 {
	return (**uint16)(unsafe.Pointer(&this.Data[0]))
}
func (this *STRRET) uOffset() *uint32 {
	return (*uint32)(unsafe.Pointer(&this.Data[0]))
}
func (this *STRRET) cStr() *[260]byte {
	return (*[260]byte)(unsafe.Pointer(&this.Data[0]))
}

func (v *IShellFolder) Release() int32 {
	ret, _, _ := syscall.Syscall(
		v.lpVtbl.Release,
		1,
		uintptr(unsafe.Pointer(v)),
		0,
		0)
	return int32(ret)
}

func (v *IEnumIDList) Release() int32 {
	ret, _, _ := syscall.Syscall(
		v.lpVtbl.Release,
		1,
		uintptr(unsafe.Pointer(v)),
		0,
		0)
	return int32(ret)
}

func (v *IShellFolder) BindToObject(pidl uintptr, pbc uintptr, riid *syscall.GUID, ppv **IShellFolder) uintptr {
	ret, _, _ := syscall.Syscall6(
		v.lpVtbl.BindToObject,
		5,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(pidl)),
		uintptr(pbc),
		uintptr(unsafe.Pointer(riid)),
		uintptr(unsafe.Pointer(ppv)),
		0)
	return ret
}

// https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-ishellfolder-enumobjects
func (v *IShellFolder) EnumObjects(hwnd uintptr, grfFlags int, ppenumIDlist **IEnumIDList) uintptr {
	ret, _, _ := syscall.Syscall6(
		v.lpVtbl.EnumObjects,
		4,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(hwnd)),
		uintptr(grfFlags),
		uintptr(unsafe.Pointer(ppenumIDlist)),
		0,
		0)
	return ret
}

func (v *IEnumIDList) Next(celt int, rgelt **ITEMIDLIST, pceltFetched *uint32) uintptr {
	ret, _, _ := syscall.Syscall6(
		v.lpVtbl.Next,
		4,
		uintptr(unsafe.Pointer(v)),
		uintptr(celt),
		uintptr(unsafe.Pointer(rgelt)),
		uintptr(unsafe.Pointer(pceltFetched)),
		0,
		0)
	return ret
}

func (v *IShellFolder) GetDisplayNameOf(pidl *ITEMIDLIST, uFlags uint32, pName *STRRET) uintptr {
	ret, _, _ := syscall.Syscall6(
		v.lpVtbl.GetDisplayNameOf,
		4,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(pidl)),
		uintptr(uFlags),
		uintptr(unsafe.Pointer(pName)),
		0,
		0)
	return ret
}

func _SHGetSpecialFolderLocation(
	hwnd uintptr,
	csidl int,
	ppidl uintptr,
) (r1 uintptr, err error) {
	r1, _, err = procSHGetSpecialFolderLocation.Call(hwnd, uintptr(csidl), ppidl)
	return
}

func _SHGetDesktopFolder(
	ppshf uintptr,
) (r1 uintptr, err error) {
	r1, _, err = procSHGetDesktopFolder.Call(ppshf)

	return
}

func _CoTaskMemFree(
	pv uintptr,
) (r1 uintptr, err error) {
	r1, _, err = procCoTaskMemFree.Call(pv)

	return
}

func _CoUninitialize() (r1 uintptr, err error) {
	r1, _, err = procCoUninitialize.Call()

	return
}

func _CoInitialize(
	pvReserved uintptr,
) (r1 uintptr, err error) {
	r1, _, err = procCoInitialize.Call(pvReserved)

	return
}

// https://stackoverflow.com/questions/39961171/golang-winapi-call-with-struct-parameter
// utf16PtrToString is like UTF16ToString, but takes *uint16
// as a parameter instead of []uint16.
// max is how many times p can be advanced looking for the null terminator.
// If max is hit, the string is truncated at that point.
func utf16PtrToString(p *uint16, max int) string {
	if p == nil {
		return ""
	}
	// Find NUL terminator.
	end := unsafe.Pointer(p)
	n := 0
	for *(*uint16)(end) != 0 && n < max {
		end = unsafe.Pointer(uintptr(end) + unsafe.Sizeof(*p))
		n++
	}
	s := (*[(1 << 30) - 1]uint16)(unsafe.Pointer(p))[:n:n]
	return string(utf16.Decode(s))
}

func CStringToString(cs *uint16) (s string) {
	if cs != nil {
		us := make([]uint16, 0, 256)
		for p := uintptr(unsafe.Pointer(cs)); ; p += 2 {
			u := *(*uint16)(unsafe.Pointer(p))
			if u == 0 {
				return string(utf16.Decode(us))
			}
			us = append(us, u)
		}
	}
	return ""
}

func getDateDelete(rbInternalFormat []byte) time.Time {
	var dd int64
	binary.Read(bytes.NewReader(rbInternalFormat[16:]), binary.LittleEndian, &dd)
	// Convert NT time to Unix epoch
	// Unix: 1/1/1970 00:00, Windows NT: 1/1/1601 00:00
	return time.Unix((dd/10000000)-11644473600, (dd%10000000)*100)
}

func getFileSize(rbInternalFormat []byte) int64 {
	var fsize int64
	binary.Read(bytes.NewReader(rbInternalFormat[8:16]), binary.LittleEndian, &fsize)
	return fsize
}

func GetTrashBoxItems() ([]fi, error) {
	ret, _ := _CoInitialize(uintptr(0))
	if ret != 0 {
		// Call FormatMessage API to display correct errors.
		return nil, _FormatMessage(ret)
	}

	var pRecycleBinFolder *IShellFolder
	ret, _ = GetRecycleBinShellFolder(&pRecycleBinFolder)
	if ret != 0 {
		return nil, _FormatMessage(ret)
	}
	defer pRecycleBinFolder.Release()

	var pEnum *IEnumIDList
	ret = pRecycleBinFolder.EnumObjects(0, SHCONTF_FOLDERS|SHCONTF_NONFOLDERS, &pEnum)
	if ret != 0 {
		fmt.Println("Failed to enumerate Recycle Bin items.")
		return nil, _FormatMessage(ret)
	}
	defer pEnum.Release()

	var files []fi
	var pItemIDL *ITEMIDLIST
	for {
		ret = pEnum.Next(1, &pItemIDL, nil)
		if ret != 0 {
			break
		}
		var file fi

		GetDisplayName(&file, pRecycleBinFolder, pItemIDL, SHGDN_INFOLDER, "InFolder")      // file name
		GetDisplayName(&file, pRecycleBinFolder, pItemIDL, SHGDN_NORMAL, "Normal")          // original Location
		GetDisplayName(&file, pRecycleBinFolder, pItemIDL, SHGDN_FORPARSING, "ForParsing")  // file name in $RECYCLE.BIN
		GetDisplayName(&file, pRecycleBinFolder, pItemIDL, SHGDN_FORPARSING, "DateDeleted") // date deleted
		GetDisplayName(&file, pRecycleBinFolder, pItemIDL, SHGDN_FORPARSING, "Size")        // file size
		files = append(files, file)

		CoTaskMemFree(uintptr(unsafe.Pointer(pItemIDL)))
	}

	_CoUninitialize()
	return files, nil
}

func GetDisplayName(file *fi, psf *IShellFolder, pidl *ITEMIDLIST, uFlags uint32, label string) {
	var pName STRRET
	ret := psf.GetDisplayNameOf(pidl, uFlags, &pName)
	if ret != 0 {
		return
	}

	if strings.Contains(label, "DateDelete") || strings.Contains(label, "Size") {
		recycleDir := filepath.Dir(CStringToString(*pName.pOleStr()))
		ipath := strings.Replace(filepath.Base(CStringToString(*pName.pOleStr())), "$R", "$I", 1)

		// Version 2 (Introduced somewhere in a Windows 10 release)
		// https://github.com/danielmarschall/recyclebinunit/blob/master/FORMAT.md#version-2-introduced-somewhere-in-a-windows-10-release
		buf := make([]byte, 24)
		f, _ := os.Open(recycleDir + "\\" + ipath)
		f.Read(buf)

		if strings.Contains(label, "DateDelete") {
			file.dateDeleted = getDateDelete(buf).Local()
		} else if strings.Contains(label, "Size") {
			file.size = getFileSize(buf)
		}
	} else if strings.Contains(label, "InFolder") {
		file.filename = CStringToString(*pName.pOleStr())
	} else if strings.Contains(label, "Normal") {
		file.location = CStringToString(*pName.pOleStr())
	} else if strings.Contains(label, "ForParsing") {
		file.inTrashBox = CStringToString(*pName.pOleStr())
	}
}

func GetRecycleBinShellFolder(pRecycleBinFolder **IShellFolder) (ret uintptr, err error) {
	var pDesktopFolder *IShellFolder
	ret, err = _SHGetDesktopFolder(uintptr(unsafe.Pointer(&pDesktopFolder)))
	if ret != 0 {
		return ret, err
	}
	defer pDesktopFolder.Release()

	var pRecycleBinIDL uintptr
	ret, err = _SHGetSpecialFolderLocation(uintptr(0), CSIDL_BITBUCKET, uintptr(unsafe.Pointer(&pRecycleBinIDL)))
	if ret != 0 {
		return ret, err
	}

	var IID_IShellFolder = syscall.GUID{0x000214E6, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	ret = pDesktopFolder.BindToObject(pRecycleBinIDL, uintptr(0), &IID_IShellFolder, &(*pRecycleBinFolder))

	return ret, err
}

func PrintTrashBoxItems() error {
	files, err := GetTrashBoxItems()
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Println()
		PrintDisplayName(file.filename, "FileName")
		PrintDisplayName(file.location, "Location")
		PrintDisplayName(file.inTrashBox, "InTrashBox")
		PrintDisplayName(file.dateDeleted.Format("2006-01-02T15:04:05Z07:00"), "DateDeleted")
		PrintDisplayName(strconv.FormatInt(file.size, 10), "Size")
	}

	return nil
}

func PrintDisplayName(line string, label string) {
	fmt.Printf("%-12s: %s\n", label, line)
}

func Undelete(srcPath string, dstPath string) error {
	r := os.Rename(srcPath, dstPath)
	if r != nil {
		return r
	}

	// $I file is still in the trash box. So deleted it.
	recycleDir := filepath.Dir(srcPath)
	ipath := strings.Replace(srcPath, "$R", "$I", 1)
	os.Remove(recycleDir + "\\" + ipath)

	return nil
}

func isMatchFilename(psf *IShellFolder, pidl *ITEMIDLIST, file string) bool {
	var pName STRRET
	ret := psf.GetDisplayNameOf(pidl, SHGDN_NORMAL, &pName)
	if ret != 0 {
		fmt.Println("Failed to get item name.")
		return false
	}

	if strings.Contains(filepath.Base(CStringToString(*pName.pOleStr())), file) {
		return true
	}

	return false
}

func _FormatMessage(errno uintptr) (err error) {
	buf := make([]uint16, 0xff)
	windows.FormatMessage(
		windows.FORMAT_MESSAGE_FROM_SYSTEM,
		uintptr(0),
		uint32(errno),
		0,
		buf,
		nil,
	)

	return errors.New(windows.UTF16ToString(buf))
}

func _SHFileOperation(
	shFileOp *SHFILEOPSTRUCT,
) (r1 uintptr, err error) {
	r1, _, err = procSHFileOperationW.Call(
		uintptr(unsafe.Pointer(shFileOp)),
	)

	return
}

func MoveToTrashBox(path string) (err error) {
	var fileOp SHFILEOPSTRUCT
	fileOp.Hwnd = uintptr(0)
	fileOp.Func = FO_DELETE
	fileOp.From = windows.StringToUTF16Ptr(path)
	fileOp.Flags = FOF_SILENT | FOF_ALLOWUNDO | FOF_NOCONFIRMATION

	// Return error is always "The operation completed successfully."
	ret, _ := _SHFileOperation(&fileOp)
	if ret != 0 {
		// Call FormatMessage API to display correct errors.
		return _FormatMessage(ret)
	}
	return nil
}
