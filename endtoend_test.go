package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// testEnv represents a test environment that can be created and disposed
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

func (te *testEnv) CopyFile(from, to string) {
	if err := copyFile(filepath.Join(te.wd, from), filepath.Join(te.wd, to)); err != nil {
		te.t.Fatal(err)
	}
}

func (te *testEnv) DuplicateFile(from, to string) {
	if err := duplicateFile(filepath.Join(te.wd, from), filepath.Join(te.wd, to)); err != nil {
		te.t.Fatal(err)
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

func (te *testEnv) CheckNotTouched(path string) {
	fullpath := filepath.Join(te.wd, path)
	fi, err := os.Stat(fullpath)
	if err != nil {
		te.t.Fatal(err)
	}
	if time.Now().Sub(fi.ModTime()) <= 10*time.Second {
		te.t.Fatalf("File modified in last 10 secs: %s", fullpath)
	}
}

func (te *testEnv) equalContent(path1, path2 string) bool {
	fullpath1 := filepath.Join(te.wd, path1)
	fullpath2 := filepath.Join(te.wd, path2)

	f1, err := os.Open(fullpath1)
	if err != nil {
		te.t.Fatal(err)
	}
	f2, err := os.Open(fullpath2)
	if err != nil {
		te.t.Fatal(err)
	}

	data1, err := ioutil.ReadAll(f1)
	if err != nil {
		te.t.Fatal(err)
	}
	data2, err := ioutil.ReadAll(f2)
	if err != nil {
		te.t.Fatal(err)
	}

	return reflect.DeepEqual(data1, data2)
}

func (te *testEnv) CheckSame(path1, path2 string) {
	if !te.equalContent(path1, path2) {
		te.t.Fatalf("Test folder: %s Files %s and %s differ", te.wd, path1, path2)
	}
}

func (te *testEnv) CheckDifferent(path1, path2 string) {
	if te.equalContent(path1, path2) {
		te.t.Fatalf("Test folder: %s Files %s and %s have the same content", te.wd, path1, path2)
	}
}

var (
	pathHelloworld = filepath.FromSlash("../test/gopath/src/hello/world/helloworld.go")
	pathHallowelt  = filepath.FromSlash("../test/gopath/src/hello/world/hallowelt.go")
	pathGreeting   = filepath.FromSlash("../test/gopath/src/hello/world/greeting.go")
)

func checkOriginalsNotTouched(te *testEnv) {
	te.CheckNotTouched(pathGreeting)
	te.CheckNotTouched(pathHallowelt)
	te.CheckNotTouched(pathHelloworld)
}

func TestFlags(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-flags")
	te.Run("./goprofile", "-o", "foo", "-p", "baz",
		pathHelloworld,
		pathGreeting,
	)
	te.RunCheckOutput([]byte("Hello world!\n"), "./foo")
	te.CheckNotEmpty("baz")
	checkOriginalsNotTouched(te)
	te.Dispose()
}

func TestFlagsInplace(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-flags-inplace")
	te.CopyFile(pathHelloworld, "helloworld.go")
	te.CopyFile(pathGreeting, "greeting.go")
	te.Run("./goprofile", "-inplace", "-o", "foo", "-p", "baz", "helloworld.go", "greeting.go")
	te.RunCheckOutput([]byte("Hello world!\n"), "./foo")
	te.CheckNotEmpty("baz")
	te.CheckDifferent(pathHelloworld, "helloworld.go")
	te.CheckSame(pathGreeting, "greeting.go")
	te.Dispose()
}

func TestBuildFlags(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-german")
	te.SetEnv("GOPATH", te.Abs("../test/gopath"))
	te.Run("./goprofile", "-buildflags", "-tags german", "hello/world")
	te.RunCheckOutput([]byte("Hallo Welt!\n"), "./world.profile")
	te.CheckNotEmpty("world.pprof")
	checkOriginalsNotTouched(te)
	te.Dispose()
}

func TestFilesEnglish(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-files-english")
	te.Run("./goprofile",
		pathHelloworld,
		pathGreeting,
	)
	te.RunCheckOutput([]byte("Hello world!\n"), "./helloworld.profile")
	te.CheckNotEmpty("helloworld.pprof")
	checkOriginalsNotTouched(te)
	te.Dispose()
}

func TestFilesGerman(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-files-german")
	te.Run("./goprofile",
		pathHallowelt,
	)
	te.RunCheckOutput([]byte("Hallo Welt!\n"), "./hallowelt.profile")
	te.CheckNotEmpty("hallowelt.pprof")
	checkOriginalsNotTouched(te)
	te.Dispose()
}

func TestPackage(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-package")
	te.SetEnv("GOPATH", te.Abs("../test/gopath"))
	te.Run("./goprofile", "hello/world")
	te.RunCheckOutput([]byte("Hello world!\n"), "./world.profile")
	te.CheckNotEmpty("world.pprof")
	checkOriginalsNotTouched(te)
	te.Dispose()
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-empty")
	te.DuplicateFile(pathHallowelt, "hallowelt.go")
	te.DuplicateFile(pathHelloworld, "helloworld.go")
	te.DuplicateFile(pathGreeting, "greeting.go")
	te.Run("./goprofile")
	te.RunCheckOutput([]byte("Hello world!\n"), "./temp_test-hello-empty.profile")
	te.CheckNotEmpty("temp_test-hello-empty.pprof")
	checkOriginalsNotTouched(te)
	te.Dispose()
}

func TestEmptyInplace(t *testing.T) {
	t.Parallel()
	te := NewTestEnv(t, "temp_test-hello-empty-inplace")
	te.CopyFile(pathHallowelt, "hallowelt.go")
	te.CopyFile(pathHelloworld, "helloworld.go")
	te.CopyFile(pathGreeting, "greeting.go")
	te.Run("./goprofile", "-inplace")
	te.RunCheckOutput([]byte("Hello world!\n"), "./temp_test-hello-empty-inplace.profile")
	te.CheckNotEmpty("temp_test-hello-empty-inplace.profile")
	te.CheckDifferent(pathHallowelt, "hallowelt.go")
	te.CheckDifferent(pathHelloworld, "helloworld.go")
	te.CheckSame(pathGreeting, "greeting.go")
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
