package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type testEnv struct {
	t       *testing.T
	envVars map[string]string
	wd      string
}

func NewTestEnv(t *testing.T, name string) *testEnv {
	te := testEnv{
		t:       t,
		envVars: make(map[string]string),
		wd:      "",
	}
	for _, entry := range os.Environ() {
		idx := strings.Index(entry, "=")
		key := entry[:idx]
		val := entry[idx+1:]
		te.envVars[key] = val
	}
	if err := os.Mkdir(name, 0777); err != nil {
		t.Fatal(err)
	}
	te.Run("go", "build", "-o", filepath.Join(name, "goprofile"))
	te.wd = name
	return &te
}

func (te *testEnv) Abs(path string) string {
	abs, err := filepath.Abs(filepath.Join(te.wd, path))
	if err != nil {
		te.t.Fatal(err)
	}
	return abs
}

func (te *testEnv) SetEnv(varname, val string) {
	te.envVars[varname] = val
}

func (te *testEnv) DuplicateFile(from, to string) {
	if err := duplicateFile(filepath.Join(te.wd, from), filepath.Join(te.wd, to)); err != nil {
		te.t.Fatal(te)
	}
}

func (te *testEnv) Run(name string, args ...string) (output []byte) {
	cmd := exec.Command(name, args...)
	for key, val := range te.envVars {
		cmd.Env = append(cmd.Env, key+"="+val)
	}
	cmd.Dir = te.wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		te.t.Fatalf("Call to '%s':%s\nEnvironment:%v\nWd:%s\nOutput:%s", name, err, cmd.Env, cmd.Dir, output)
	}
	return
}

func (te *testEnv) RunCheckOutput(expectedOutput []byte, name string, args ...string) {
	if out := te.Run(name, args...); !reflect.DeepEqual(expectedOutput, out) {
		te.t.Fatalf("Error running %s. Expected %#v; got %#v", name, expectedOutput, out)
	}
}

func (te *testEnv) Dispose() {
	if err := os.RemoveAll(te.wd); err != nil {
		te.t.Fatal(err)
	}
}

func (te *testEnv) CheckNotEmpty(path string) {
	fullpath := filepath.Join(te.wd, path)
	fi, err := os.Stat(fullpath)
	if err != nil {
		te.t.Fatal(err)
	}
	if fi.IsDir() {
		te.t.Fatalf("Found directory instead of file at %s", fullpath)
	}
	if fi.Size() == 0 {
		te.t.Fatalf("File %s is empty", fullpath)
	}
}

func TestFlags(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-flags")
	te.Run("./goprofile", "-o", "foo", "-p", "baz",
		filepath.FromSlash("../test/gopath/src/hello/world/helloworld.go"),
		filepath.FromSlash("../test/gopath/src/hello/world/greeting.go"),
	)
	te.RunCheckOutput([]byte("Hello world!\n"), "./foo")
	te.CheckNotEmpty("baz")
	te.Dispose()
}

func TestBuildFlags(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-german")
	te.SetEnv("GOPATH", te.Abs("../test/gopath"))
	te.Run("./goprofile", "-buildflags", "-tags german", "hello/world")
	te.RunCheckOutput([]byte("Hallo Welt!\n"), "./world.profile")
	te.CheckNotEmpty("world.pprof")
	te.Dispose()
}

func TestFilesEnglish(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-files-english")
	te.Run("./goprofile",
		filepath.FromSlash("../test/gopath/src/hello/world/helloworld.go"),
		filepath.FromSlash("../test/gopath/src/hello/world/greeting.go"),
	)
	te.RunCheckOutput([]byte("Hello world!\n"), "./helloworld.profile")
	te.CheckNotEmpty("helloworld.pprof")
	te.Dispose()
}

func TestFilesGerman(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-files-german")
	te.Run("./goprofile",
		filepath.FromSlash("../test/gopath/src/hello/world/hallowelt.go"),
	)
	te.RunCheckOutput([]byte("Hallo Welt!\n"), "./hallowelt.profile")
	te.CheckNotEmpty("hallowelt.pprof")
	te.Dispose()
}

func TestPackage(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-package")
	te.SetEnv("GOPATH", te.Abs("../test/gopath"))
	te.Run("./goprofile", "hello/world")
	te.RunCheckOutput([]byte("Hello world!\n"), "./world.profile")
	te.CheckNotEmpty("world.pprof")
	te.Dispose()
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-empty")
	te.DuplicateFile(filepath.FromSlash("../test/gopath/src/hello/world/hallowelt.go"), "hallowelt.go")
	te.DuplicateFile(filepath.FromSlash("../test/gopath/src/hello/world/helloworld.go"), "helloworld.go")
	te.DuplicateFile(filepath.FromSlash("../test/gopath/src/hello/world/greeting.go"), "greeting.go")
	te.Run("./goprofile")
	te.RunCheckOutput([]byte("Hello world!\n"), "./temp_test-hello-empty.profile")
	te.CheckNotEmpty("temp_test-hello-empty.pprof")
	te.Dispose()
}

func TestSelf(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-self")
	te.DuplicateFile(filepath.FromSlash("../ast.go"), "ast.go")
	te.DuplicateFile(filepath.FromSlash("../cmd.go"), "cmd.go")
	te.DuplicateFile(filepath.FromSlash("../process.go"), "process.go")
	te.DuplicateFile(filepath.FromSlash("../util.go"), "util.go")
	te.Run("./goprofile")
	te.Run("./temp_test-self.profile", "-o", "temp_test-self.profile.profile", "-p", "temp_test-self.profile.pprof")
	te.CheckNotEmpty("temp_test-self.pprof")
	te.Run("./temp_test-self.profile.profile", "-o", "temp_test-self.profile.profile.profile", "-p", "temp_test-self.profile.profile.pprof")
	te.CheckNotEmpty("temp_test-self.profile.pprof")
	te.Dispose()
}
