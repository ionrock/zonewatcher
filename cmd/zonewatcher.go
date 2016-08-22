package main

import (
	// "log"
	"os"

	"github.com/ionrock/zonewatcher"
	"github.com/urfave/cli"
)

func ServerAction(c *cli.Context) error {
	s := zonewatcher.Server{}
	s.Start()

	return nil
}

func PingPongDNSServerAction(c *cli.Context) error {
	s := zonewatcher.DnsHandler{}
	host := c.Args()[0]
	port := c.Args()[1]
	go zonewatcher.Serve("tcp", host, port, &s)
	go zonewatcher.Serve("udp", host, port, &s)

	zonewatcher.Listen()
	return nil
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		cli.Command{
			Name:   "server",
			Usage:  "Run the zonewatcher web server",
			Action: ServerAction,
		},
		cli.Command{
			Name:   "pingpong",
			Usage:  "Run a ping/pong dns server",
			Action: PingPongDNSServerAction,
		},
	}
	app.Run(os.Args)
}
