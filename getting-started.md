# Getting started

This document is a guide to introduce people to the `gb` project structure. A `gb` project is conceptually a `$GOPATH` per project, but saying that doesn't really help explain how to set up a new project; hence this document.

# Creating an empty project

A `gb` project is defined as any directory that has a `src/` subdirectory. The simplest possible `gb` project would be

     % mkdir -p ~/project/src/
     % cd ~/project

`~/project` is therefore a `gb` project.

Source inside a `gb` project follows the same rules as the `go` tool, see the [Workspaces section of the Go getting started document](https://golang.org/doc/code.html#Workspaces). All Go code goes in packages, and packages are subdirectories inside the project's `src/` directory

     % cd ~/project
     % mkdir -p src/cmd/helloworld
     % cat <<EOF > src/cmd/helloworld/helloworld.go
     package main
     
     import "fmt"
      
     func main() {
             fmt.Println("Hello world")
     }
     EOF
     % gb build cmd/helloworld

will build the small `helloworld` command.

*note*: there is a bug currently where binaries are not installed to the project's `bin/` directory as you would expect. This will be fixed shortly

# Converting an existing project

This section shows how to construct a `gb` project using existing code bases.

## Simple example

In this example we'll create a `gb` project from the `github.com/pkg/sftp` codebase. 

First, create a project,

     % mkdir -p ~/devel/sftp
     % cd ~/devel/sfp

Now checkout `github.com/pkg/sftp` to the path it expects

     % mkdir -p src/github.com/pkg/sftp
     % git clone -q https://github.com/pkg/sftp src/github.com/pkg/sftp

Now, let's try to build this

     % gb build all
     2015/04/29 13:39:44 INFO project root "/home/dfc/devel/sftp"
     2015/04/29 13:39:44 INFO build duration: 486.967µs map[]
     2015/04/29 13:39:44 command "build" failed: failed to resolve package "github.com/pkg/sftp": cannot find package "github.com/kr/fs" in any of:
             /home/dfc/go/src/github.com/kr/fs (from $GOROOT)
             /home/dfc/devel/sftp/src/github.com/kr/fs (from $GOPATH)
             /home/dfc/devel/sftp/vendor/src/github.com/kr/fs

The build failed because the dependency, `github.com/kr/fs` was not found in the project, which was expected (ignore the message about `$GOPATH` this is a side effect of reusing the `go/build` package for dependency resolution). So we can use the `gb vendor` plugin to fetch the code for `github.com/kr/fs`, and try again

     % gb vendor github.com/kr/fs
     2015/04/29 13:42:02 INFO project root "/home/dfc/devel/sftp"
     % gb build all                                                                                                                   
     2015/04/29 13:42:06 INFO project root "/home/dfc/devel/sftp"
     2015/04/29 13:42:06 INFO build duration: 701.994µs map[]
     2015/04/29 13:42:06 command "build" failed: failed to resolve package "github.com/pkg/sftp": cannot find package "golang.org/x/crypto/ssh" in any of:
             /home/dfc/go/src/golang.org/x/crypto/ssh (from $GOROOT)
             /home/dfc/devel/sftp/src/golang.org/x/crypto/ssh (from $GOPATH)
             /home/dfc/devel/sftp/vendor/src/golang.org/x/crypto/ssh

Nearly, there, just missing the `golang.org/x/crypto/ssh` package, again we'll use `gb vendor`.

      % gb vendor golang.org/x/crypto/ssh
     2015/04/29 13:44:32 INFO project root "/home/dfc/devel/sftp"
      % gb build all                                                                                                                   
     2015/04/29 13:44:40 INFO project root "/home/dfc/devel/sftp"
     2015/04/29 13:44:40 INFO compile github.com/kr/fs [filesystem.go walk.go]
     2015/04/29 13:44:40 INFO compile golang.org/x/crypto/ssh [buffer.go certs.go channel.go cipher.go client.go client_auth.go common.go connection.go doc.go handshake.go kex.go keys.go mac.go messages.go mux.go server.go session.go tcpip.go transport.go]
     2015/04/29 13:44:40 INFO install compile {fs github.com/kr/fs /home/dfc/devel/sftp/vendor/src/github.com/kr/fs}
     2015/04/29 13:44:41 INFO install compile {ssh golang.org/x/crypto/ssh /home/dfc/devel/sftp/vendor/src/golang.org/x/crypto/ssh}
     2015/04/29 13:44:41 INFO compile golang.org/x/crypto/ssh/agent [client.go forward.go keyring.go server.go]
     2015/04/29 13:44:41 INFO compile github.com/pkg/sftp [attrs.go client.go packet.go release.go sftp.go]
     2015/04/29 13:44:42 INFO install compile {agent golang.org/x/crypto/ssh/agent /home/dfc/devel/sftp/vendor/src/golang.org/x/crypto/ssh/agent}
     2015/04/29 13:44:42 INFO install compile {sftp github.com/pkg/sftp /home/dfc/devel/sftp/src/github.com/pkg/sftp}
     2015/04/29 13:44:42 INFO compile github.com/pkg/sftp/examples/buffered-write-benchmark [main.go]
     2015/04/29 13:44:42 INFO compile github.com/pkg/sftp/examples/gsftp [main.go]
     2015/04/29 13:44:42 INFO compile github.com/pkg/sftp/examples/streaming-read-benchmark [main.go]
     2015/04/29 13:44:42 INFO compile github.com/pkg/sftp/examples/streaming-write-benchmark [main.go]
     2015/04/29 13:44:42 INFO compile github.com/pkg/sftp/examples/buffered-read-benchmark [main.go]
     2015/04/29 13:44:42 INFO link /tmp/gb634560345/github.com/pkg/sftp/examples/main [/tmp/gb634560345/github.com/pkg/sftp/examples/buffered-read-benchmark.a]
     2015/04/29 13:44:42 INFO link /tmp/gb634560345/github.com/pkg/sftp/examples/main [/tmp/gb634560345/github.com/pkg/sftp/examples/gsftp.a]
     2015/04/29 13:44:42 INFO link /tmp/gb634560345/github.com/pkg/sftp/examples/main [/tmp/gb634560345/github.com/pkg/sftp/examples/buffered-write-benchmark.a]
     2015/04/29 13:44:42 INFO link /tmp/gb634560345/github.com/pkg/sftp/examples/main [/tmp/gb634560345/github.com/pkg/sftp/examples/streaming-write-benchmark.a]
     2015/04/29 13:44:42 INFO link /tmp/gb634560345/github.com/pkg/sftp/examples/main [/tmp/gb634560345/github.com/pkg/sftp/examples/streaming-read-benchmark.a]
     2015/04/29 13:44:44 INFO build duration: 4.25611787s map[compile:4.240654481s link:9.331042949s]

And now it builds. Some things to note

- The package name `all` matches all the packages inside your project's `src/` directory. It's a simple way to build everything, you can use other import paths and globs.
- There is no way to build your vendored source, it will be built if required to build your code in the `src/` directory.
- There is currently a bug where commands are linked, but not moved into the project's `bin/` directory. This will be fixed shortly.

## More complicated example

For the second example we'll take a project that uses `godep` vendoring and convert it to be a `gb` project. First we'll need to setup a project and get the source

     % mkdir -p ~/devel/confd
     % cd ~/devel/confd
     % mkdir -p src/github.com/kelseyhightower/confd
     % git clone https://github.com/kelseyhightower/confd src/github.com/kelseyhightower/confd  

Now, we know this project uses `godeps`, so already includes all its dependencies, so we just need to rearrange things a bit.
 
     % mkdir -p vendor/src/
     % mv src/github.com/kelseyhightower/confd/Godeps/_workspace/src/* vendor/src/

Let's see if it builds

     % gb build all
     2015/04/29 13:52:40 INFO project root "/home/dfc/devel/confd"
     ...
     2015/04/29 13:52:40 INFO compile github.com/kelseyhightower/confd [confd.go config.go node_var.go version.go]
     2015/04/29 13:52:40 INFO compile github.com/kelseyhightower/confd/integration/zookeeper [main.go]
     2015/04/29 13:52:40 INFO link /tmp/gb137712104/github.com/kelseyhightower/confd/integration/main [/tmp/gb137712104/github.com/kelseyhightower/confd/integration/zookeeper.a]
     2015/04/29 13:52:40 INFO link /tmp/gb137712104/github.com/kelseyhightower/main [/tmp/gb137712104/github.com/kelseyhightower/confd.a]
     2015/04/29 13:52:42 INFO build duration: 1.657282147s map[compile:387.488748ms link:2.166243738s]

And it does (modulo the linking bug).

# Wrapping up

Setting up, or converting code to a `gb` project is simple. Once you're done, just check the whole project into your source control.
