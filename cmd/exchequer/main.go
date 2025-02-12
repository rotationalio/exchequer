package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/oklog/ulid/v2"
	"github.com/urfave/cli/v2"

	"github.com/rotationalio/exchequer/pkg"
	"github.com/rotationalio/exchequer/pkg/config"
	"github.com/rotationalio/exchequer/pkg/exchequer"

	confire "github.com/rotationalio/confire/usage"
)

func main() {
	godotenv.Load()

	app := cli.NewApp()
	app.Name = "exchequer"
	app.Usage = "serve and manage the exchequer billing service"
	app.Version = pkg.Version()
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{
		{
			Name:     "serve",
			Usage:    "run the exchequer service configured from the environment",
			Action:   serve,
			Category: "server",
		},
		{
			Name:     "config",
			Usage:    "print the exchequer configuration guide",
			Category: "server",
			Action:   usage,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "print in list mode instead of table mode",
				},
			},
		},
		{
			Name:     "tokenkey",
			Usage:    "generate an RSA token key pair and ulid for JWT token signing",
			Category: "admin",
			Action:   generateTokenKey,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "out",
					Aliases: []string{"o"},
					Usage:   "path to write keys out to (optional, will be saved as ulid.pem by default)",
				},
				&cli.IntFlag{
					Name:    "size",
					Aliases: []string{"s"},
					Usage:   "number of bits for the generated keys",
					Value:   4096,
				},
			},
		},
	}

	app.Run(os.Args)
}

//===========================================================================
// Server Commands
//===========================================================================

func serve(c *cli.Context) (err error) {
	var conf config.Config
	if conf, err = config.New(); err != nil {
		return cli.Exit(err, 1)
	}

	var svc *exchequer.Server
	if svc, err = exchequer.New(conf); err != nil {
		return cli.Exit(err, 1)
	}

	if err = svc.Serve(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func usage(c *cli.Context) error {
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	format := confire.DefaultTableFormat
	if c.Bool("list") {
		format = confire.DefaultListFormat
	}

	var conf config.Config
	if err := confire.Usagef(config.Prefix, &conf, tabs, format); err != nil {
		return cli.Exit(err, 1)
	}

	tabs.Flush()
	return nil
}

//===========================================================================
// Administrative Commands
//===========================================================================

func generateTokenKey(c *cli.Context) (err error) {
	// Create ULID and determine outpath
	keyid := ulid.Make()

	var out string
	if out = c.String("out"); out == "" {
		out = fmt.Sprintf("%s.pem", keyid)
	}

	// Generate RSA keys using crypto random
	var key *rsa.PrivateKey
	if key, err = rsa.GenerateKey(rand.Reader, c.Int("size")); err != nil {
		return cli.Exit(err, 1)
	}

	// Open file to PEM encode keys to
	var f *os.File
	if f, err = os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600); err != nil {
		return cli.Exit(err, 1)
	}

	if err = pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("RSA key id: %s -- saved with PEM encoding to %s\n", keyid, out)
	return nil
}

//===========================================================================
// Helper Functions
//===========================================================================

// func openDB(c *cli.Context) (err error) {
// 	if conf, err = config.New(); err != nil {
// 		return cli.Exit(err, 1)
// 	}

// 	if db, err = store.Open(conf.DatabaseURL); err != nil {
// 		return cli.Exit(err, 1)
// 	}

// 	return nil
// }

// func closeDB(c *cli.Context) error {
// 	if db != nil {
// 		if err := db.Close(); err != nil {
// 			return cli.Exit(err, 1)
// 		}
// 	}
// 	return nil
// }
