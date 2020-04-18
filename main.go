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

	"github.com/go-pg/pg/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

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
		Name:    "index",
		Aliases: []string{"i"},
		Usage:   "Index file",
	}

	pgFlag := &cli.StringFlag{
		Name:    "postgresql",
		Aliases: []string{"pg"},
		Usage:   "Postgresql connection strings",
		EnvVars: []string{"PG_SQL"},
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
				logLevelFlag,
				indexFileFlag,
				&cli.StringFlag{
					Name:     "sources",
					Aliases:  []string{"s"},
					Usage:    "Files to index",
					Required: true,
				},
				jsonFlag,
				pgFlag,
			},
			Action: build,
		},
		{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "Search over the index",
			Flags: []cli.Flag{
				logLevelFlag,
				indexFileFlag,
				jsonFlag,
				pgFlag,
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

	engine := index.NewMemoryIndex()
	i := index.NewIndex(engine, nil)

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

	if err := engine.Encode(encoder); err != nil {
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
	var err error
	if err = initLogger(c); err != nil {
		return err
	}
	var engine index.IndexEngine
	if c.String("index") != "" {
		engine, err = getMemoryEngine(c)
		if err != nil {
			return err
		}
	}
	if c.String("postgresql") != "" {
		engine, err = getPgEngine(c)
		if err != nil {
			return err
		}
	}
	defer engine.Close()

	index := index.NewIndex(engine, nil)

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

func getMemoryEngine(c *cli.Context) (index.IndexEngine, error) {
	indexFile := c.String("index")
	file, err := os.Open(indexFile)
	if err != nil {
		return nil, fmt.Errorf("can not open index file %s: %w", indexFile, err)
	}

	var decoder index.Decoder
	if c.Bool("json") {
		decoder = json.NewDecoder(file)
	} else {
		decoder = gob.NewDecoder(file)
	}
	return index.Decode(decoder)
}

func getPgEngine(c *cli.Context) (index.IndexEngine, error) {
	pgOpt, err := pg.ParseURL(c.String("postgresql"))
	if err != nil {
		return nil, err
	}
	pgdb := pg.Connect(pgOpt)
	return index.NewDbIndex(pgdb), nil
}
