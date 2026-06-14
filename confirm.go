package main

import (
	"syscall"
	"unsafe"

	"github.com/sqweek/dialog"
)

const (
	tdCommonButtonOK     = 0x0001
	tdCommonButtonCancel = 0x0008

	tdfAllowDialogCancellation = 0x0008

	idOK     = 1
	idCancel = 2
)

// taskDialogConfig mirrors the Win32 TASKDIALOGCONFIG struct.
type taskDialogConfig struct {
	cbSize                  uint32
	hwndParent              uintptr
	hInstance               uintptr
	dwFlags                 uint32
	dwCommonButtons         uint32
	pszWindowTitle          *uint16
	pszMainIcon             uintptr
	pszMainInstruction      *uint16
	pszContent              *uint16
	cButtons                uint32
	pButtons                uintptr
	nDefaultButton          int32
	cRadioButtons           uint32
	pRadioButtons           uintptr
	nDefaultRadioButton     int32
	pszVerificationText     *uint16
	pszExpandedInformation  *uint16
	pszExpandedControlText  *uint16
	pszCollapsedControlText *uint16
	pszFooterIcon           uintptr
	pszFooter               *uint16
	pfCallback              uintptr
	lpCallbackData          uintptr
	cxWidth                 uint32
}

var (
	comctl32               = syscall.NewLazyDLL("comctl32.dll")
	procTaskDialogIndirect = comctl32.NewProc("TaskDialogIndirect")
)

// confirmFolderDeletion asks the user to check a confirmation box before the
// contents of the given folder are permanently deleted. It returns true only
// if the user checks the box and clicks OK; false if they cancel.
func confirmFolderDeletion(folder string) bool {
	title, err1 := syscall.UTF16PtrFromString(dialogTitle)
	instruction, err2 := syscall.UTF16PtrFromString("Warning: this will permanently delete folder contents")
	content, err3 := syscall.UTF16PtrFromString(
		"Everything inside the following folder will be permanently deleted every 2 hours, starting now:\n\n" + folder + "\n\nThis cannot be undone.")
	verification, err4 := syscall.UTF16PtrFromString("I understand that the contents of this folder will be permanently deleted")

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return dialog.Message("Permanently delete everything inside\n%s\nevery 2 hours?", folder).Title(dialogTitle).YesNo()
	}

	for {
		cfg := taskDialogConfig{
			dwFlags:             tdfAllowDialogCancellation,
			dwCommonButtons:     tdCommonButtonOK | tdCommonButtonCancel,
			pszWindowTitle:      title,
			pszMainInstruction:  instruction,
			pszContent:          content,
			pszVerificationText: verification,
			nDefaultButton:      idCancel,
		}
		cfg.cbSize = uint32(unsafe.Sizeof(cfg))

		var button int32
		var checked int32

		ret, _, _ := procTaskDialogIndirect.Call(
			uintptr(unsafe.Pointer(&cfg)),
			uintptr(unsafe.Pointer(&button)),
			0,
			uintptr(unsafe.Pointer(&checked)),
		)
		if ret != 0 {
			// TaskDialogIndirect failed; fall back to a simple confirmation.
			return dialog.Message("Permanently delete everything inside\n%s\nevery 2 hours?", folder).Title(dialogTitle).YesNo()
		}

		if button != idOK {
			return false
		}
		if checked != 0 {
			return true
		}

		dialog.Message("Please check the confirmation box to continue, or click Cancel.").Title(dialogTitle).Info()
	}
}
