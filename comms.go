// forked from github.com/go-mangos/mangos/example/pair/pair.go
//
// pair implements a pair example.  node0 is a listening
// pair socket, and node1 is a dialing pair socket.
//
// To use:
//
//   $ go build .
//   $ url=tcp://127.0.0.1:40899
//   $ ./pair node0 $url & node0=$!
//   $ ./pair node1 $url & node1=$!
//   $ sleep 3
//   $ kill $node0 $node1
//
package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/pair"
	"github.com/go-mangos/mangos/transport/ipc"
	"github.com/go-mangos/mangos/transport/tcp"
)

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func sendName(sock mangos.Socket, name string) {
	fmt.Printf("%s: SENDING \"%s\"\n", name, name)
	if err := sock.Send([]byte(name)); err != nil {
		die("failed sending: %s", err)
	}
}

func recvName(sock mangos.Socket, name string) {
	var msg []byte
	var err error
	if msg, err = sock.Recv(); err == nil {
		fmt.Printf("%s: RECEIVED: \"%s\"\n", name, string(msg))
	}
}

func sendRecv(sock mangos.Socket, name string) {
	for {
		sock.SetOption(mangos.OptionRecvDeadline, 100*time.Millisecond)
		recvName(sock, name)
		time.Sleep(time.Second)
		sendName(sock, name)
	}
}

func core(lPort string) {
	var sock mangos.Socket
	var err error
	if sock, err = pair.NewSocket(); err != nil {
		die("can't get new pair socket: %s", err)
	}
	sock.AddTransport(ipc.NewTransport())
	sock.AddTransport(tcp.NewTransport())
	if err = sock.Listen("tcp://localhost:" + lPort); err != nil {
		die("can't listen on pair socket: %s", err.Error())
	}
	sendRecv(sock, "core")
}

func backup(lPort string) {
	var sock mangos.Socket
	var err error

	if sock, err = pair.NewSocket(); err != nil {
		die("can't get new pair socket: %s", err.Error())
	}
	sock.AddTransport(ipc.NewTransport())
	sock.AddTransport(tcp.NewTransport())
	if err = sock.Dial("tcp://localhost:" + lPort); err != nil {
		die("can't dial on pair socket: %s", err.Error())
	}
	sendRecv(sock, "backup")
}

func openSSH(lPort string, rPort string, rHost string) {
	cmd := exec.Command("ssh", "-L", lPort+":localhost:"+rPort, rHost)
	fmt.Println(cmd)
}

func main() {
	usage := `Ark backup node communication.
	
	Usage:
	  comms (core|backkup) <localPort> <remotePort> <remoteHost>`

	arguments, _ := docopt.ParseDoc(usage)

	lPort := arguments["<lPort>"].(string)
	rPort := arguments["<rPort>"].(string)
	rHost := arguments["<remoteHost>"].(string)
	openSSH(lPort, rPort, rHost)

	if arguments["core"] == true {
		core(lPort)
	} else if arguments["backup"] == true {
		backup(lPort)
	}
}
