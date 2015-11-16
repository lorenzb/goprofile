package main

import (
	"io"
	"os"
	"path/filepath"
)

// copyFile copies a file to a new location. If the destination file
// already exists, an error occurs.
func copyFile(from, to string) error {
	inFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer inFile.Close()
	outFile, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, inFile)
	if err != nil {
		return err
	}
	return nil
}

// duplicateFile duplicates a file: It tries to create a symlink; if that
// fails it falls back to copying the file. (This is to deal with platforms/filesystems
// that do not support symlinks.) If the destination file already exists, an error
// occurs.
func duplicateFile(from, to string) error {
	absFrom, err := filepath.Abs(from)
	if err != nil {
		return err
	}
	if err := os.Symlink(absFrom, to); err != nil {
		if err := copyFile(from, to); err != nil {
			return err
		}
	}
	return nil
}
