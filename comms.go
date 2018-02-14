// base from github.com/go-mangos/mangos/examples/bus/bus.go

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	docopt "github.com/docopt/docopt-go"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/rep"
	"github.com/go-mangos/mangos/protocol/req"
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
	var sock mangos.Socket             // the socket
	var err error                      // catch errors
	var recv []byte                    // the received bytes
	cmds := make(map[string]*exec.Cmd) // the ssh commands that were started

	msg := []byte("hello from core")      // the test message to send from the core node to the backups
	prefixURL := "tcp://localhost:"       // the URL prefix
	bPortBase, err := strconv.Atoi(lPort) // first backup port
	if err != nil {
		die("strconv.Atoi: %s", err.Error())
	}

	if sock, err = req.NewSocket(); err != nil {
		die("req.NewSocket: %s", err.Error())
	}
	sock.AddTransport(ipc.NewTransport())
	sock.AddTransport(tcp.NewTransport())

	for index, element := range backups {
		currBPort := strconv.Itoa(bPortBase + index)                            // current backup port
		cmds[currBPort] = openSSH(element.port, element.host, currBPort, lPort) // open the SSH tunnel
	}
	for {
		for currBPort := range cmds {
			if err = sock.Dial(prefixURL + currBPort); err != nil {
				die("socket.Dial: %s", err.Error())
			}
			time.Sleep(3 * time.Second) // wait for the dial to complete

			for { // for now, just do it indefinitely
				// eventually, if the wrong reply is received (or no reply) move onto next backup
				fmt.Printf("SENDING REQUEST %s\n", string(msg))
				if err = sock.Send(msg); err != nil { // send message to the backup
					die("sock.Send: %s", err.Error())
				}
				if recv, err = sock.Recv(); err != nil {
					die("sock.Recv: %s", err.Error())
				}
				fmt.Printf("RECEIVED REPLY %s\n", string(recv))
				time.Sleep(3 * time.Second)
			}
		}
	}
}

func backup(lPort string) {
	var sock mangos.Socket // the socket
	var err error          // catch errors
	var recv []byte

	msg := []byte("hello from backup") // the test message to send from the backup to the core
	lURL := "tcp://localhost:" + lPort // the local URL to talk on

	if sock, err = rep.NewSocket(); err != nil {
		die("rep.NewSocket: %s", err)
	}
	sock.AddTransport(ipc.NewTransport())
	sock.AddTransport(tcp.NewTransport())
	if err = sock.Listen(lURL); err != nil {
		die("sock.Listen: %s", err.Error())
	}
	for {
		recv, err = sock.Recv()
		if err != nil {
			die("sock.Recv: %s", err.Error())
		}
		fmt.Printf("RECEIVED REQUEST %s\n", string(recv))
		fmt.Printf("SENDING REPLY %s\n", string(msg))
		err = sock.Send(msg)
		if err != nil {
			die("can't send reply: %s", err.Error())
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
