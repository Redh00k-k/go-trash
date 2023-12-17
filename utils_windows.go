package main

import (
	"syscall"
	"unsafe"

	/*
		#cgo LDFLAGS: -Llib -ltrashbox -lOle32 -lOleAut32 -lShlwapi -luuid -lpropsys -lstdc++ -static
		#include "lib/trashbox.h"
		#include <stdlib.h>
	*/
	"C"

	"golang.org/x/sys/windows"
)
import (
	"errors"
)

// https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shfileoperationw
var (
	modshell32           = syscall.NewLazyDLL("Shell32.dll")
	procSHFileOperationW = modshell32.NewProc("SHFileOperationW")
)

// https://pinvoke.net/default.aspx/Enums/FileFuncFlags.html
const (
	FO_MOVE   = 0x1
	FO_COPY   = 0x2
	FO_DELETE = 0x3
	FO_RENAME = 0x4
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

func PrintTrashBoxItems() (ret int) {
	ret = int(C.PrintTrashBox())
	return
}

func RestoreItem(file string) (ret int) {
	cStr := C.CString(file)
	defer C.free(unsafe.Pointer(cStr))
	ret = int(C.RestoreItem(cStr))
	return
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
