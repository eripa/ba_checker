package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mkideal/cli"
)

const toolVersion = "0.0.1"

type configuration struct {
	Sites     []string
	Endpoints map[string]bool
}

type argT struct {
	cli.Helper
	Config  string `cli:"c,config" usage:"JSON config file {\"sites:\" [\"https://www.example.com\"], \"endpoints\": {\"index.php\": false, \"wp-admin/users.php\": true}}"`
	Version bool   `cli:"version" usage:"Check version"`
}

func main() {
	cli.Run(&argT{}, func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		if argv.Version {
			ctx.String("ba_checker v%s\n", toolVersion)
			return nil
		}
		if argv.Config == "" {
			return fmt.Errorf("--config <config.json> is required.\n")
		}
		if _, err := os.Stat(argv.Config); os.IsNotExist(err) {
			return fmt.Errorf("Error: %s does not exist", argv.Config)
		}
		file, _ := os.Open(argv.Config)
		decoder := json.NewDecoder(file)
		config := configuration{}
		err := decoder.Decode(&config)
		if err != nil {
			fmt.Println("error:", err)
		}
		fmt.Printf("%v\n", config.Endpoints)
		return nil
	})
}
