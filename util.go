package main

import (
	"io"
	"os"
	"path/filepath"
)

func copyFile(from, to string) error {
	inFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer inFile.Close()
	outFile, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0666)
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

func duplicateFile(from, to string) error {
	// First try to symlink; if that doesn't work, fall back to copying
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
