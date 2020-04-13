package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	stdLog "log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/polisgo2020/search-tariel-x/index"
	ifaceCli "github.com/polisgo2020/search-tariel-x/interface/cli"
	"github.com/polisgo2020/search-tariel-x/interface/ws"
)

func main() {
	app := cli.NewApp()
	app.Name = "Search index"
	app.Usage = "generate index from text files and search over them"
	app.Before = initLogger

	indexFileFlag := &cli.StringFlag{
		Name:     "index",
		Aliases:  []string{"i"},
		Usage:    "Index file",
		Required: true,
	}

	jsonFlag := &cli.BoolFlag{
		Name:  "json",
		Usage: "Use json-encoded index",
	}

	logLevelFlag := &cli.StringFlag{
		Name:    "logLevel",
		Usage:   "Log level",
		Value:   "debug",
		EnvVars: []string{"LOG_LEVEL"},
	}

	app.Commands = []*cli.Command{
		{
			Name:    "build",
			Aliases: []string{"b"},
			Usage:   "Build search index",
			Flags: []cli.Flag{
				indexFileFlag,
				&cli.StringFlag{
					Name:     "sources",
					Aliases:  []string{"s"},
					Usage:    "Files to index",
					Required: true,
				},
				jsonFlag,
				logLevelFlag,
			},
			Action: build,
		},
		{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "Search over the index",
			Flags: []cli.Flag{
				indexFileFlag,
				jsonFlag,
				logLevelFlag,
				&cli.StringFlag{
					Name:    "listen",
					Aliases: []string{"l"},
					Usage:   "Interface to listen",
					EnvVars: []string{"LISTEN"},
				},
			},
			Action: search,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
}

func initLogger(c *cli.Context) error {
	logLevel, err := zerolog.ParseLevel(c.String("logLevel"))
	if err != nil {
		stdLog.Print(err)
		return err
	}
	zerolog.SetGlobalLevel(logLevel)
	return nil
}

func build(c *cli.Context) error {
	if err := initLogger(c); err != nil {
		return err
	}
	sourcesDir := c.String("sources")
	files, err := ioutil.ReadDir(sourcesDir)
	if err != nil {
		return err
	}

	i := index.NewIndex(nil)

	wg := &sync.WaitGroup{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		wg.Add(1)
		go func(fileName string) {
			defer wg.Done()
			if err := readFile(fileName, i); err != nil {
				log.Printf("cannot read file %s: %w", fileName, err)
			}
		}(filepath.Join(sourcesDir, file.Name()))
	}
	wg.Wait()

	indexFile := c.String("index")
	output, err := os.Create(indexFile)
	if err != nil {
		return fmt.Errorf("can not create output file %s: %w", indexFile, err)
	}
	defer output.Close()

	var encoder index.Encoder
	if c.Bool("json") {
		encoder = json.NewEncoder(output)
	} else {
		encoder = gob.NewEncoder(output)
	}

	if err := i.Encode(encoder); err != nil {
		return fmt.Errorf("can not write index: %w", err)
	}

	return nil
}

func readFile(name string, i *index.Index) error {
	input, err := os.Open(name)
	if err != nil {
		return err
	}
	defer input.Close()

	return i.AddSource(name, input)
}

func search(c *cli.Context) error {
	if err := initLogger(c); err != nil {
		return err
	}
	indexFile := c.String("index")
	file, err := os.Open(indexFile)
	if err != nil {
		return fmt.Errorf("can not open index file %s: %w", indexFile, err)
	}

	var decoder index.Decoder
	if c.Bool("json") {
		decoder = json.NewDecoder(file)
	} else {
		decoder = gob.NewDecoder(file)
	}

	index, err := index.Decode(decoder)
	if err != nil {
		return fmt.Errorf("can not read index file %s: %w", indexFile, err)
	}

	if c.String("listen") == "" {
		iface, err := ifaceCli.New(os.Stdin, os.Stdout, index)
		if err != nil {
			return err
		}
		return iface.Run()
	}

	iface, err := ws.New(c.String("listen"), 10*time.Second, index)
	if err != nil {
		return err
	}
	return iface.Run()
}
