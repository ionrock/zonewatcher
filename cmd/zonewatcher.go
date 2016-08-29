package main

import (
	"log"
	"os"

	"github.com/ionrock/zonewatcher"
	"github.com/urfave/cli"
)

func ServerAction(c *cli.Context) error {
	s := zonewatcher.Server{}
	s.Start()

	return nil
}

func TestingDNSServerAction(c *cli.Context) error {
	handler := zonewatcher.DnsHandler{}
	host := c.Args()[0]
	port := c.Args()[1]

	go handler.Serve(handler.NewServer("tcp", host, port))
	go handler.Serve(handler.NewServer("udp", host, port))

	zonewatcher.Listen()
	return nil
}

func WatchZoneAction(c *cli.Context) error {
	args := c.Args()
	zone := args[0]
	ns := "127.0.0.1:53"

	if len(args) > 1 {
		ns = args[1]
	}

	dig := zonewatcher.DNSClient{Ns: ns}

	o := zonewatcher.Observer{Zone: zone, Ns: ns}
	o.Watch(dig, nil)

	log.Printf("%#v", o)

	return nil
}

func QueryZoneWatchDBAction(c *cli.Context) error {
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
			Name:   "dnsserver",
			Usage:  "Run a testing dns server",
			Action: TestingDNSServerAction,
		},
		cli.Command{
			Name:   "watch",
			Usage:  "Watch a zone",
			Action: WatchZoneAction,
		},
		cli.Command{
			Name:   "query",
			Usage:  "Query a zonewatcher db",
			Action: QueryZoneWatchDBAction,
			Flags:  []cli.Flag{},
		},
	}
	app.Run(os.Args)
}
