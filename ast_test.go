package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

var tests = []struct {
	src            string
	hasMain        bool
	hasPProfImport bool
}{
	{`
	package main

	func main() {}
	`, true, false},
	{`
	package moo

	import "runtime/pprof"

	func main() {}
	`, false, true},
	{`
	package main

	import (
		"foo"
		"runtime/pprof"
		"baz"
	)

	func main(args []string) {}
	`, false, true},
	{`
	package main

	import "pprof"

	func main() int {}
	`, false, false},
	{`
	package main

	import (
		"foo"
		"baz"
		"bar"
	)
	import "heh"
	import "runtime/pprof"

	type Bar struct{}

	func (Bar b) main() {}
	`, false, true},
}

func parse(t *testing.T, src string) *ast.File {
	fileset := token.NewFileSet()
	ast, err := parser.ParseFile(fileset, "", strings.NewReader(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return ast
}

func TestHasMain(t *testing.T) {
	t.Parallel()
	for _, test := range tests {
		ast := parse(t, test.src)
		if hm := hasMain(ast); hm != test.hasMain {
			t.Fatalf("expected %v, got %v\nSource code:%s", test.hasMain, hm, test.src)
		}
	}
}

func TestHasImport(t *testing.T) {
	t.Parallel()
	for _, test := range tests {
		ast := parse(t, test.src)
		if hi := hasImport(ast, `"runtime/pprof"`); hi != test.hasPProfImport {
			t.Fatalf("expected %v, got %v\nSource code:%s", test.hasPProfImport, hi, test.src)
		}
	}
}

func testInstrument(t *testing.T, proffile, srcOrig, srcExpected string) {
	bufExpected := &bytes.Buffer{}
	bufActual := &bytes.Buffer{}

	astExpected := parse(t, srcExpected)
	astActual := parse(t, srcOrig)
	instrument(astActual, proffile)

	printer.Fprint(bufExpected, token.NewFileSet(), astExpected)
	printer.Fprint(bufActual, token.NewFileSet(), astActual)
	if !bytes.Equal(bufExpected.Bytes(), bufActual.Bytes()) {
		t.Fatalf("Expected:\n%s\n Actual:\n%s\n", bufExpected.String(), bufActual.String())
	}

}

func TestInstrument1(t *testing.T) {
	t.Parallel()
	proffile := "bla.prof"
	srcOrig := `
	package main

	func main() {
		fmt.Println("abc")
	}`
	srcExpected := `
	package main

	import "os"
	import "runtime/pprof"
	func main() {
		{
 			f, err := os.Create("bla.prof")
		 	if err != nil {
		 		os.Stderr.WriteString("Couldn't open bla.prof: "+err.Error()+"\n")
 				return
		 	}
	 		pprof.StartCPUProfile(f)
	 		defer pprof.StopCPUProfile()
		 }
		fmt.Println("abc")
	}`
	testInstrument(t, proffile, srcOrig, srcExpected)
}

func TestInstrumentQuoting(t *testing.T) {
	t.Parallel()
	proffile := "foo\" \"asd.out"
	srcOrig := `
	package main

	func main() {
		fmt.Println("abc")
	}`
	srcExpected := `
	package main

	import "os"
	import "runtime/pprof"
	func main() {
		{
 			f, err := os.Create("foo\" \"asd.out")
		 	if err != nil {
		 		os.Stderr.WriteString("Couldn't open foo\" \"asd.out: "+err.Error()+"\n")
 				return
		 	}
	 		pprof.StartCPUProfile(f)
	 		defer pprof.StopCPUProfile()
		 }
		fmt.Println("abc")
	}`
	testInstrument(t, proffile, srcOrig, srcExpected)
}

func TestInstrumentImportsPresent(t *testing.T) {
	t.Parallel()
	proffile := "foo.pprof"
	srcOrig := `
	package main

	import (
		"bla"
		"os"
		"bork"
		"runtime/pprof"
		"fmt"
	)
	func main() {
		fmt.Println("abc")
	}`
	srcExpected := `
	package main

	import (
		"bla"
		"os"
		"bork"
		"runtime/pprof"
		"fmt"
	)
	func main() {
		{
 			f, err := os.Create("foo.pprof")
		 	if err != nil {
		 		os.Stderr.WriteString("Couldn't open foo.pprof: "+err.Error()+"\n")
 				return
		 	}
	 		pprof.StartCPUProfile(f)
	 		defer pprof.StopCPUProfile()
		 }
		fmt.Println("abc")
	}`
	testInstrument(t, proffile, srcOrig, srcExpected)
}

func TestInstrumentSomeImportsPresent(t *testing.T) {
	t.Parallel()
	proffile := "foo.pprof"
	srcOrig := `
	package main

	import (
		"bla"
		"os"
		"bork"
		"fmt"
	)
	func main() {
		fmt.Println("abc")
	}`
	srcExpected := `
	package main

	import "runtime/pprof"
	import (
		"bla"
		"os"
		"bork"
		"fmt"
	)
	func main() {
		{
 			f, err := os.Create("foo.pprof")
		 	if err != nil {
		 		os.Stderr.WriteString("Couldn't open foo.pprof: "+err.Error()+"\n")
 				return
		 	}
	 		pprof.StartCPUProfile(f)
	 		defer pprof.StopCPUProfile()
		 }
		fmt.Println("abc")
	}`
	testInstrument(t, proffile, srcOrig, srcExpected)
}
