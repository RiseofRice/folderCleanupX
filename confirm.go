package main

import (
	"github.com/sqweek/dialog"
)

// confirmFolderDeletion warns the user that everything inside the given
// folder will be permanently deleted every 2 hours, and requires two
// explicit confirmations before returning true. Returns false if the user
// declines either confirmation.
func confirmFolderDeletion(folder string) bool {
	ok := dialog.Message(
		"Warning: everything inside the following folder will be permanently deleted, starting now and then every 2 hours:\n\n%s\n\nThis cannot be undone. Continue?",
		folder,
	).Title(dialogTitle).YesNo()
	if !ok {
		return false
	}

	return dialog.Message(
		"Please confirm again: all files and subfolders in\n%s\nwill be permanently deleted, repeatedly, every 2 hours.\n\nClick Yes to confirm.",
		folder,
	).Title(dialogTitle).YesNo()
}
