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

# Converting an existing project

This section shows how to construct a `gb` project using existing code bases.

## Simple example

In this example we'll create a `gb` project from the `github.com/pkg/sftp` codebase. 

First, create a project,

     % mkdir -p ~/devel/sftp
     % cd ~/devel/sftp

Now checkout `github.com/pkg/sftp` to the path it expects

     % mkdir -p src/github.com/pkg/sftp
     % git clone https://github.com/pkg/sftp src/github.com/pkg/sftp

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
     2015/04/29 19:50:55 INFO compile github.com/pkg/sftp/examples/buffered-read-benchmark [main.go]
     2015/04/29 19:50:55 INFO compile github.com/pkg/sftp/examples/buffered-write-benchmark [main.go]
     2015/04/29 19:50:55 INFO compile github.com/pkg/sftp/examples/gsftp [main.go]
     2015/04/29 19:50:55 INFO compile github.com/pkg/sftp/examples/streaming-read-benchmark [main.go]
     2015/04/29 19:50:55 INFO compile github.com/pkg/sftp/examples/streaming-write-benchmark [main.go]
     2015/04/29 19:50:56 INFO link /home/dfc/devel/sftp/bin/buffered-read-benchmark [/tmp/gb786934546/github.com/pkg/sftp/examples/buffered-read-benchmark/main.a]
     2015/04/29 19:50:56 INFO link /home/dfc/devel/sftp/bin/gsftp [/tmp/gb786934546/github.com/pkg/sftp/examples/gsftp/main.a]
     2015/04/29 19:50:56 INFO link /home/dfc/devel/sftp/bin/streaming-read-benchmark [/tmp/gb786934546/github.com/pkg/sftp/examples/streaming-read-benchmark/main.a]
     2015/04/29 19:50:56 INFO link /home/dfc/devel/sftp/bin/streaming-write-benchmark [/tmp/gb786934546/github.com/pkg/sftp/examples/streaming-write-benchmark/main.a]
     2015/04/29 19:50:56 INFO link /home/dfc/devel/sftp/bin/buffered-write-benchmark [/tmp/gb786934546/github.com/pkg/sftp/examples/buffered-write-benchmark/main.a]
     2015/04/29 19:50:58 INFO build duration: 2.535541868s map[compile:1.895628229s link:9.827128875s]

And now it builds. Some things to note

- The package name `all` matches all the packages inside your project's `src/` directory. It's a simple way to build everything, you can use other import paths and globs.
- There is no way to build your vendored source, it will be built if required to build your code in the `src/` directory.

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
     2015/04/29 19:52:16 INFO project root "/home/dfc/devel/confd"
     2015/04/29 19:52:16 INFO compile github.com/kelseyhightower/confd [confd.go config.go node_var.go version.go]
     2015/04/29 19:52:16 INFO compile github.com/kelseyhightower/confd/integration/zookeeper [main.go]
     2015/04/29 19:52:16 INFO link /home/dfc/devel/confd/bin/zookeeper [/tmp/gb934182157/github.com/kelseyhightower/confd/integration/zookeeper/main.a]
     2015/04/29 19:52:16 INFO link /home/dfc/devel/confd/bin/confd [/tmp/gb934182157/github.com/kelseyhightower/confd/main.a]
     2015/04/29 19:52:17 INFO build duration: 1.7575955s map[compile:405.681764ms link:2.275663206s]

And it does.

# Wrapping up

Setting up, or converting code to a `gb` project is simple. Once you're done, just check the whole project into your source control.
