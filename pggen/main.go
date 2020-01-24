package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"

	"github.com/opendoor-labs/pggen/gen"
)

func usage(ok bool) {
	usage := `
Usage: pggen [<options>] <config-file>

Args:
 <config-file> A configuration toml file containing a list of database objects
               that pggen should generate code for.

Options:
-h, --help                                   Print this message.

-c, --connection-string <connection-string>  The connection string to use to attach
                                             to the postgres instance we will
                                             generate shims for. Defaults to $DB_URL.

-o, --output-file <file-name>                The name of the file to write the shims to.
                                             If the file name ends with .go it will be
                                             re-written to end with .gen.go.
                                             Defaults to "./pg_generated.gen.go".
`
	if ok {
		fmt.Print(usage)
		os.Exit(0)
	} else {
		log.Fatal(usage)
	}
}

func main() {
	var config gen.Config
	config.OutputFileName = "./pg_generated.go"

	func() {
		// While parsing args we will might panic on out-of-bounds array
		// access. This means the arguments are malformed.
		defer func() {
			if x := recover(); x != nil {
				usage(false)
			}
		}()

		args := os.Args[1:]
		for len(args) > 0 {
			if args[0] == "-c" || args[0] == "--connection-string" {
				config.ConnectionString = args[1]
				args = args[2:]
			} else if args[0] == "-f" || args[0] == "--config-file" {
				config.ConfigFilePath = args[1]
				args = args[2:]
			} else if args[0] == "-o" || args[0] == "--output-file" {
				config.OutputFileName = args[1]
				args = args[2:]
			} else if args[0] == "-h" || args[0] == "--help" {
				usage(true)
			} else if args[0] == "--allow-test-mode" {
				// An undocumented argument that allows `pggen` to operate in test
				// mode. We gaurd test mode with a special flag because running in
				// test mode involves Execing `db.sql` which blows away and recreates
				// the `public` database schema.
				args = args[1:]
			} else if len(args) == 1 {
				config.ConfigFilePath = args[0]
				break
			} else {
				usage(false)
			}
		}
	}()

	if config.ConnectionString == "" {
		config.ConnectionString = os.Getenv("DB_URL")
		if len(config.ConnectionString) == 0 {
			log.Fatal("No connection string. Either pass '-c' or set DB_URL in the environment.")
		}
	}

	if strings.HasSuffix(config.OutputFileName, ".go") &&
		!strings.HasSuffix(config.OutputFileName, ".gen.go") {
		config.OutputFileName = config.OutputFileName[:len(config.OutputFileName)-3] + ".gen.go"
	}

	//
	// Create the codegenerator and invoke it
	//

	g, err := gen.FromConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	err = g.Gen()
	if err != nil {
		log.Fatal(err)
	}
}
