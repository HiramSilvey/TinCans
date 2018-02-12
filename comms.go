// base from github.com/go-mangos/mangos/examples/bus/bus.go

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	docopt "github.com/docopt/docopt-go"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/bus"
	"github.com/go-mangos/mangos/transport/ipc"
	"github.com/go-mangos/mangos/transport/tcp"
)

type node struct {
	port string
	host string
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func core(backups []node, lPort string) {
	var sock mangos.Socket // the socket
	var cmds []*exec.Cmd   // the ssh commands that were started
	var err error          // catch errors
	var recv []byte

	msg := []byte("hello from core")      // the test message to send from the core node to the backups
	prefixURL := "tcp://localhost:"       // the URL prefix
	lURL := prefixURL + lPort             // the local URL to listen on
	bPortBase, err := strconv.Atoi(lPort) // first backup port
	if err != nil {
		die("strconv.Atoi: %s", err.Error())
	}
	bPortBase++ // start 1 higher than lPort

	if sock, err = bus.NewSocket(); err != nil {
		die("bus.NewSocket: %s", err.Error())
	}
	sock.AddTransport(ipc.NewTransport()) // *not sure if needed*
	sock.AddTransport(tcp.NewTransport()) // transport for TCP messages
	if err = sock.Listen(lURL); err != nil {
		die("sock.Listen: %s", err.Error())
	}

	for index, element := range backups {
		currBPort := strconv.Itoa(bPortBase + index)                               // current backup port
		cmds = append(cmds, openSSH(element.port, element.host, currBPort, lPort)) // open the SSH tunnel
		if err = sock.Dial(prefixURL + currBPort); err != nil {
			die("socket.Dial: %s", err.Error())
		}
	}

	for {
		fmt.Printf("%s: SENDING '%s' ONTO BUS\n", lURL, msg)
		if err = sock.Send(msg); err != nil { // send the message to the bus
			die("sock.Send: %s", err.Error())
		}
		for {
			if recv, err = sock.Recv(); err != nil { // receive all messages from the bus
				die("sock.Recv: %s", err.Error())
			}
			fmt.Printf("%s: RECEIVED \"%s\" FROM BUS\n", lURL,
				string(recv))
		}
	}
}

func backup(lPort string) {
	var sock mangos.Socket // the socket
	var err error          // catch errors
	var recv []byte

	msg := []byte("hello from backup") // the test message to send from the backup to the core
	lURL := "tcp://localhost:" + lPort // the local URL to talk on

	if sock, err = bus.NewSocket(); err != nil {
		die("bus.NewSocket: %s", err.Error())
	}

	sock.AddTransport(ipc.NewTransport()) // *not sure if needed*
	sock.AddTransport(tcp.NewTransport()) // transport for TCP messages
	if err = sock.Listen(lURL); err != nil {
		die("sock.Listen: %s", err.Error())
	}
	if err = sock.Dial(lURL); err != nil {
		die("socket.Dial: %s", err.Error())
	}

	for {
		if recv, err = sock.Recv(); err != nil { // receive a message from the bus
			die("sock.Recv: %s", err.Error())
		}
		fmt.Printf("%s: RECEIVED \"%s\" FROM BUS\n", lURL, string(recv))
		fmt.Printf("%s: SENDING '%s' ONTO BUS\n", lURL, msg)
		if err = sock.Send(msg); err != nil { // send the message to the bus
			die("sock.Send: %s", err.Error())
		}
	}
}

func openSSH(rPort string, rHost string, bPort string, lPort string) *exec.Cmd {
	cmd := exec.Command("ssh", "-N", "-L", bPort+":localhost:"+lPort, "-i ~/.ssh/id_rsa", "-p"+rPort, rHost) // port forward without opening an SSH session
	err := cmd.Start()
	if err != nil {
		die("cmd.Start: %s", err.Error())
	}
	return cmd
}

func main() {
	usage := `Ark backup node communication.
	
	Usage:
	  comms [<config.csv>]`

	arguments, _ := docopt.ParseDoc(usage)
	lPort := "5124" // local port
	if arguments["<config.csv>"] == nil {
		backup(lPort) // backup node
	}
	file, err := os.Open(arguments["<config.csv>"].(string)) // the config file for the core node
	if err != nil {
		die("os.Open: %s", err.Error())
	}
	defer file.Close()

	var backups []node

	reader := csv.NewReader(file)
	for {
		tokens, err := reader.Read() // read the line and break it into tokens using the comma delim
		if err == io.EOF {
			break
		} else if err != nil {
			die("reader.Read: %s", err.Error())
		}
		if len(tokens) != 2 {
			die("reader.Read: result not of length 2")
		}
		n := node{port: tokens[0], host: tokens[1]}
		backups = append(backups, n)
	}
	core(backups, lPort) // core node
}
