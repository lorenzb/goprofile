package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

func processGoFile(from, to string) (foundMain bool, err error) {
	var fileAst *ast.File
	fs := token.NewFileSet()
	fileAst, err = parser.ParseFile(fs, from, nil, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("Parser error: %s", err)
	}

	if hasMain(fileAst) {
		instrument(fileAst, options.ProfFile)

		outFile, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
		if err != nil {
			return true, fmt.Errorf("Failed to create file: %s", err)
		}
		defer outFile.Close()

		printer.Fprint(outFile, fs, fileAst)

		return true, nil
	} else {
		return false, duplicateFile(from, to)
	}
}

func processFile(from, to string) (foundMain bool, err error) {
	if strings.HasSuffix(from, ".go") {
		foundMain, err := processGoFile(from, to)
		if err != nil {
			return foundMain, fmt.Errorf("Error processing go file %s: %s", from, err)
		}
		return foundMain, nil
	} else {
		if err = duplicateFile(from, to); err != nil {
			return false, fmt.Errorf("Error duplicating file %s: %s", from, err)
		}
		return false, nil
	}
}

func processFileInPlace(path string) (foundMain bool, err error) {
	if !strings.HasSuffix(path, ".go") {
		return false, nil
	}

	var fileAst *ast.File
	fs := token.NewFileSet()
	fileAst, err = parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("Parser error: %s", err)
	}

	if hasMain(fileAst) {
		instrument(fileAst, options.ProfFile)

		outFile, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return true, fmt.Errorf("Failed to truncate file: %s", err)
		}
		defer outFile.Close()

		printer.Fprint(outFile, fs, fileAst)

		return true, nil
	} else {
		return false, nil
	}
}
