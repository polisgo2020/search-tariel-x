package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/urfave/cli/v2"

	"github.com/polisgo2020/search-tariel-x/index"
)

func main() {
	app := cli.NewApp()
	app.Name = "Search index"
	app.Usage = "generate index from text files and search over them"

	indexFileFlag := &cli.StringFlag{
		Name:     "index, i",
		Usage:    "Index file",
		Required: true,
	}

	jsonFlag := &cli.BoolFlag{
		Name:  "json",
		Usage: "Use json-encoded index",
	}

	app.Commands = []*cli.Command{
		{
			Name:    "build",
			Aliases: []string{"b"},
			Usage:   "Build search index",
			Flags: []cli.Flag{
				indexFileFlag,
				&cli.StringFlag{
					Name:     "sources, s",
					Usage:    "Files to index",
					Required: true,
				},
				jsonFlag,
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
			},
			Action: search,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func build(c *cli.Context) error {
	sourcesDir := c.String("sources")
	files, err := ioutil.ReadDir(sourcesDir)
	if err != nil {
		return err
	}

	i := index.NewIndex()

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

	for {
		reader := bufio.NewReader(os.Stdin)
		query, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("can not read query: %w", err)
		}

		results, err := index.Search(query, nil)
		if err != nil {
			return err
		}
		for i, result := range results {
			fmt.Printf("%d. %s\n", i+1, result.Document.Name)
		}
	}

	return nil
}
