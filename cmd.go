package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	shellwords "github.com/mattn/go-shellwords"
)

var options struct {
	InPlace    bool
	PrintWork  bool
	Verbose    bool
	Output     string
	ProfFile   string
	BuildFlags []string
}

var flags flag.FlagSet

func main() {
	var buildFlags string
	var help bool

	flags.Init(os.Args[0], flag.ContinueOnError)
	flags.StringVar(&buildFlags, "buildflags", "", "arguments to pass on to the underlying invocation of 'go build'")
	flags.BoolVar(&help, "h", false, "")
	flags.BoolVar(&help, "help", false, "show help")
	flags.BoolVar(&options.InPlace, "inplace", false, "perform instrumentation in-place \n    \tDANGER: This will overwrite your source files! \n    \tOnly use this if your files are under version control.")
	flags.StringVar(&options.Output, "o", "", "path to instrumented output binary")
	flags.StringVar(&options.ProfFile, "p", "", "path to profiling output")
	flags.BoolVar(&options.Verbose, "v", false, "")
	flags.BoolVar(&options.Verbose, "verbose", false, "print verbose output")
	flags.BoolVar(&options.PrintWork, "work", false, "print the name of the temporary work directory")
	flags.Parse(os.Args[1:])

	var err error
	options.BuildFlags, err = shellwords.Parse(buildFlags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse given buildflags.", err)
		os.Exit(1)
	}

	if help {
		h := func(args ...interface{}) {
			fmt.Fprintln(os.Stderr, args...)
		}
		h(`Usage: goprofile [-o output binary] [-p profile] [source files... | package]`)
		h()
		h(`Rule of thumb: 'go build' + profiling instrumentation = goprofile.`)
		h()
		h(`goprofile compiles the given files/package into an instrumented binary.`)
		h(`The instrumented binary is like the vanilla binary created by 'go build' but`)
		h(`outputs profiling information.`)
		h(``)
		h(`If no source files or package are specified, goprofile will attempt to treat`)
		h(`the current directory as a package.`)
		h()
		h(`Flags:`)
		flags.PrintDefaults()
		h()
		h(`Examples:`)
		h(`1)`)
		h(`You have written a simple exploratory program that resides in the file`)
		h(`"myprogram.go" in the current working directory. To profile it, you run`)
		h(`    goprofile myprogram.go && ./myprogram.profile && go tool pprof myprogram.profile myprogram.pprof`)
		h(``)
		h(`2)`)
		h(`You have a complex application that you want to profile on a remote server.`)
		h(`You want the profiling output to be written into your home directory on the`)
		h(`server. You run`)
		h(`    goprofile -p ~/trace.pprof module/path/of/your/complexapp`)
		h(`    # copy file complexapp.profile to the server`)
		h(`    # on the server, execute your complexapp.profile binary`)
		h(`    # copy the ~/trace.pprof from your server to your local machine`)
		h(`    go tool pprof complexapp.profile trace.pprof`)
		h(``)
		h(`Details:`)
		h(`If goprofile receives multiple source files as arguments`)
		h(`(e.g. goprofile foo.go cmd.go), it will name the output after the first file `)
		h(`(e.g. foo.profile). If a package is passed, the output will be named after the`)
		h(`last element of the package path. If nothing is passed, goprofile will name`)
		h(`the output after the current working directory.`)
		h(``)
		return
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Fatal:", err)
		os.Exit(1)
	}
}

func hasRelevantEnding(name string) bool {
	for _, suff := range []string{
		".go", ".c", ".cc", ".cpp", ".cxx", ".m", ".h", ".hh",
		".hpp", "hxx", ".s", ".swig", ".swigcxx", ".syso",
	} {
		if strings.HasSuffix(name, suff) {
			return true
		}
	}
	return false
}

// fileset computes the set of files to be instrumented/copied
// from the arguments passed to the program.
// paths is the set of files. list is a boolean indicating whether
// the set of files was explicitly listed on the command line
// (e.g. "goprofile foo.go bla.go").
func fileset() (paths []string, list bool, err error) {
	dir := func(path string) ([]string, bool, error) {
		fd, err := os.Open(path)
		if err != nil {
			return nil, false, err
		}
		var relevantPaths []string
		fis, err := fd.Readdir(-1)
		if err != nil {
			return nil, false, err
		}
		for _, fi := range fis {
			if !fi.IsDir() && hasRelevantEnding(fi.Name()) {
				relevantPaths = append(relevantPaths, filepath.Join(path, fi.Name()))
			}
		}
		return relevantPaths, false, nil
	}

	switch len(flags.Args()) {
	case 0:
		return dir(".")
	case 1:
		fi, err := os.Stat(flags.Arg(0))
		if err == nil && !fi.IsDir() {
			return flags.Args(), true, nil
		} else {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				return nil, false, errors.New("Empty GOPATH environment variable")
			}
			return dir(filepath.Join(gopath, "src", flags.Arg(0)))
		}
	default:
		var paths []string
		var dir string
		for _, arg := range flags.Args() {
			_, err := os.Stat(arg)
			if err != nil {
				return nil, true, err
			}
			if dir == "" {
				dir = filepath.Dir(arg)
			}
			if dir != filepath.Dir(arg) {
				err := fmt.Errorf("named files must all be in one directory; have '%s' and '%s'",
					dir, filepath.Dir(arg))
				return nil, false, err
			}
			paths = append(paths, arg)
		}
		return paths, true, nil
	}
}

func outputName() (string, error) {
	switch len(flags.Args()) {
	case 0:
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Base(wd), nil
	case 1:
		fallthrough
	default:
		name := filepath.Base(flags.Arg(0))
		if strings.Contains(name, ".") {
			end := strings.LastIndex(name, ".")
			return name[:end], nil
		} else {
			return name, nil
		}
	}
}

func makeworkdir() (string, error) {
	var dir string
	var err error

	if options.InPlace {
		dir = "."
	} else {
		dir, err = ioutil.TempDir("", "goprofile")
		if err != nil {
			return "", err
		}
		dir = filepath.Join(dir, time.Now().Format("2006-02-01T15_04_05"))
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return "", err
		}
	}
	if options.PrintWork {
		fmt.Printf("GOPROFILEWORK='%s'\n", dir)
	}
	return dir, nil
}

func run() error {
	workdir, err := makeworkdir()
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	paths, list, err := fileset()
	if err != nil {
		return err
	}

	name, err := outputName()
	if err != nil {
		return err
	}

	if options.ProfFile == "" {
		options.ProfFile = name + ".pprof"
	}

	if options.Output == "" {
		options.Output = name + ".profile"
	}

	if !filepath.IsAbs(options.Output) {
		options.Output = filepath.Join(wd, options.Output)
	}

	if options.Verbose {
		fmt.Fprintln(os.Stderr, "Will compile to", options.Output)
		fmt.Fprintln(os.Stderr, "Instrumented executable will save cpu profile as", options.ProfFile)
	}

	var tos = make(map[string]string)
	for _, path := range paths {
		tos[path] = filepath.Join(workdir, filepath.Base(path))
	}

	var foundMain bool
	for from, to := range tos {
		var fm bool
		if options.InPlace {
			fm, err = processFileInPlace(from)
		} else {
			fm, err = processFile(from, to)
		}
		if err != nil {
			return err
		}
		foundMain = foundMain || fm
		if fm && options.Verbose {
			fmt.Printf("Found and instrumented main() function in %s.\n", from)
		}
	}

	if !foundMain {
		return errors.New("Couldn't find a main() function to instrument")
	}

	if err := os.Chdir(workdir); err != nil {
		return err
	}

	cmd := []string{"build"}
	cmd = append(cmd, options.BuildFlags...)
	cmd = append(cmd, "-o", options.Output)
	if list {
		for _, to := range tos {
			fmt.Println("to:", to)
			cmd = append(cmd, to)
		}
	}
	gobuild := exec.Command("go", cmd...)
	gobuild.Stdout = os.Stdout
	gobuild.Stderr = os.Stderr

	if options.Verbose {
		fmt.Fprintln(os.Stderr, "Successfully instrumented code. Compiling with go build.")
	}

	if err := gobuild.Run(); err != nil {
		return err
	}

	return nil
}
