// Copyright 2021 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mendersoftware/go-lib-micro/config"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	dconfig "github.com/mendersoftware/azure-iot-manager/config"
	"github.com/mendersoftware/azure-iot-manager/server"
	store "github.com/mendersoftware/azure-iot-manager/store/mongo"
)

func main() {
	doMain(os.Args)
}

func doMain(args []string) {
	var configPath string

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "config",
				Usage: "Configuration `FILE`. " +
					"Supports JSON, TOML, YAML and HCL " +
					"formatted configs.",
				Value:       "/etc/azure-iot-manager/config.yaml",
				Destination: &configPath,
			},
		},
		Commands: []cli.Command{
			{
				Name:   "server",
				Usage:  "Run the HTTP API server",
				Action: cmdServer,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "automigrate",
						Usage: "Run database migrations before starting.",
					},
				},
			},
			{
				Name:   "migrate",
				Usage:  "Run the migrations",
				Action: cmdMigrate,
			},
		},
	}
	app.Usage = "Azure IoT Manager"
	app.Action = cmdServer

	app.Before = func(args *cli.Context) error {
		err := config.FromConfigFile(configPath, dconfig.Defaults)
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("error loading configuration: %s", err),
				1)
		}

		// Enable setting config values by environment variables
		config.Config.SetEnvPrefix("AZURE_IOT_MANAGER")
		config.Config.AutomaticEnv()
		config.Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

		return nil
	}

	err := app.Run(args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdServer(args *cli.Context) error {
	mgoConfig := store.NewConfig().SetAutomigrate(args.Bool("automigrate"))
	dataStore, err := store.SetupDataStore(mgoConfig)
	if err != nil {
		return err
	}
	defer dataStore.Close()
	return server.InitAndRun(config.Config, dataStore)
}

func cmdMigrate(args *cli.Context) error {
	return nil
}
