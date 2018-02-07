// base from github.com/go-mangos/mangos/examples/bus/bus.go

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"

	docopt "github.com/docopt/docopt-go"
)

type node struct {
	port string
	host string
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func core(backups []node) {
	// TODO UPDATE FROM BUS TEMPLATE CODE
	/*
		var sock mangos.Socket
		var err error
		var msg []byte
		var x int

		if sock, err = bus.NewSocket(); err != nil {
			die("bus.NewSocket: %s", err)
		}
		sock.AddTransport(ipc.NewTransport())
		sock.AddTransport(tcp.NewTransport())
		if err = sock.Listen(args[2]); err != nil {
			die("sock.Listen: %s", err.Error())
		}

		// wait for everyone to start listening
		time.Sleep(time.Second)
		for x = 3; x < len(args); x++ {
			if err = sock.Dial(args[x]); err != nil {
				die("socket.Dial: %s", err.Error())
			}
		}

		// wait for everyone to join
		time.Sleep(time.Second)

		fmt.Printf("%s: SENDING '%s' ONTO BUS\n", args[1], args[1])
		if err = sock.Send([]byte(args[1])); err != nil {
			die("sock.Send: %s", err.Error())
		}
		for {
			if msg, err = sock.Recv(); err != nil {
				die("sock.Recv: %s", err.Error())
			}
			fmt.Printf("%s: RECEIVED \"%s\" FROM BUS\n", args[1],
				string(msg))

		}
	*/
}

func backup() {}

func openSSH(rPort string, rHost string) *exec.Cmd {
	cmd := exec.Command("ssh", "-N", "-L", "5124:localhost:5124", "-i ~/.ssh/id_rsa", "-p"+rPort, rHost)
	err := cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("cmd.Start: %s", err))
		return nil
	}
	return cmd
}

func main() {
	usage := `Ark backup node communication.
	
	Usage:
	  comms [<config.json>]`

	arguments, _ := docopt.ParseDoc(usage)
	if arguments["<config.json>"] == nil {
		backup()
	}
	file, err := os.Open(arguments["<config.json>"].(string))
	if err != nil {
		die("os.Open: %s", err)
	}
	defer file.Close()

	var backups []node

	reader := csv.NewReader(file)
	for {
		tokens, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			die("reader.Read: %s", err)
		}
		if len(tokens) != 2 {
			die("reader.Read: result not of length 2")
		}
		n := node{port: tokens[0], host: tokens[1]}
		backups = append(backups, n)
	}
	core(backups)
}
