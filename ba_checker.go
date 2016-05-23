package main

import (
	"fmt"
	"os"

	"github.com/mkideal/cli"
)

type argT struct {
	cli.Helper
	Config string `cli:"*c,config" usage:"Config file"`
}

func (argv *argT) Validate(ctx *cli.Context) error {
	if _, err := os.Stat(argv.Config); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist", argv.Config)
	}
	return nil
}

func main() {
	cli.Run(&argT{}, func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		ctx.String("Hello, %s!\n", argv.Config)
		return nil
	})
}
