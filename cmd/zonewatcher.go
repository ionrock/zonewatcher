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

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		cli.Command{
			Name:   "server",
			Usage:  "Run the zonewatcher web server",
			Action: ServerAction,
		},
	}
	app.Run(os.Args)
}
