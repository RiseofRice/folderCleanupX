package main

import (
	"os"

	"golang.org/x/sys/windows"
)

const (
	singleInstanceMutexName = `FolderCleanerX_SingleInstanceMutex`
	singleInstanceEventName = `FolderCleanerX_QuitEvent`
)

// ensureSingleInstance makes sure only one copy of the program runs at a
// time. If another instance is already running, it signals that instance to
// exit and waits briefly for it to do so before this instance continues. It
// also starts a background watcher that exits this process if a newer
// instance later asks it to quit.
func ensureSingleInstance() {
	eventName, err := windows.UTF16PtrFromString(singleInstanceEventName)
	if err != nil {
		return
	}
	hEvent, err := windows.CreateEvent(nil, 0, 0, eventName)
	if err == nil && hEvent != 0 {
		go func() {
			for {
				windows.WaitForSingleObject(hEvent, windows.INFINITE)
				os.Exit(0)
			}
		}()
	}

	mutexName, err := windows.UTF16PtrFromString(singleInstanceMutexName)
	if err != nil {
		return
	}
	hMutex, err := windows.CreateMutex(nil, true, mutexName)
	if hMutex == 0 {
		return
	}

	if err == windows.ERROR_ALREADY_EXISTS {
		if hEvent != 0 {
			windows.SetEvent(hEvent)
		}
		// Wait for the previous instance to exit and release the mutex.
		windows.WaitForSingleObject(hMutex, 5000)
	}
}
