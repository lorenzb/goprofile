#goprofile

In one simple equation:
**'go build' + profiling instrumentation = goprofile.**

goprofile is similar to go build, but instruments your program with profiling code (using the "runtime/pprof" package) before building it.

##How mature is goprofile?
goprofile is **not** mature software. More like "early-alpha".
So if your rely on it in production, and things go wrong...

...well, just don't!

That being said, it has some system and unit tests (which even pass!), so it should be somewhat reliable and useful.


##Usage

```
Usage: goprofile [-o output binary] [-p profile] [source files... | package]

Rule of thumb: 'go build' + profiling instrumentation = goprofile.

goprofile compiles the given files/package into an instrumented binary.
The instrumented binary is like the vanilla binary created by 'go build' but
outputs profiling information.

If no source files or package are specified, goprofile will attempt to treat
the current directory as a package.

Flags:
  -buildflags string
    	arguments to pass on to the underlying invocation of 'go build'
  -h	
  -help
    	show help
  -o string
    	path to instrumented output binary
  -p string
    	path to profiling output
  -v	
  -verbose
    	print verbose output
  -work
    	print the name of the temporary work directory

Examples:
1)
You have written a simple exploratory program that resides in the file
"myprogram.go" in the current working directory. To profile it, you run
    goprofile myprogram.go && ./myprogram.profile && go tool pprof myprogram.profile myprogram.pprof

2)
You have a complex application that you want to profile on a remote server.
You want the profiling output to be written into your home directory on the
server. You run
    goprofile -p ~/trace.pprof module/path/of/your/complexapp
    # copy file complexapp.profile to the server
    # on the server, execute your complexapp.profile binary
    # copy the ~/trace.pprof from your server to your local machine
    go tool pprof complexapp.profile trace.pprof

Details:
If goprofile receives multiple source files as arguments
(e.g. goprofile foo.go cmd.go), it will name the output after the first file 
(e.g. foo.profile). If a package is passed, the output will be named after the
last element of the package path. If nothing is passed, goprofile will name
the output after the current working directory.
```

##License
goprofile is licensed under a 2-clause BSD license:

Copyright (c) 2015, the github user "lorenzb"
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
