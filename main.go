package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/polisgo2020/search-tariel-x/index"
)

func main() {
	app := cli.NewApp()
	app.Name = "Search index"
	app.Usage = "generate index from text files and search over them"

	indexFileFlag := &cli.StringFlag{
		Name:  "index, i",
		Usage: "Index file",
	}

	sourcesFlag := &cli.StringFlag{
		Name:  "sources, s",
		Usage: "Files to index",
	}

	app.Commands = []*cli.Command{
		{
			Name:    "build",
			Aliases: []string{"b"},
			Usage:   "Build search index",
			Flags: []cli.Flag{
				indexFileFlag,
				sourcesFlag,
			},
			Action: build,
		},
		{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "Search over the index",
			Flags: []cli.Flag{
				indexFileFlag,
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

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if err := readFile(filepath.Join(sourcesDir, file.Name()), i); err != nil {
			return fmt.Errorf("cannot read file %s: %w", file.Name(), err)
		}
	}

	indexFile := c.String("index")
	output, err := os.Create(indexFile)
	if err != nil {
		return fmt.Errorf("can not create output file %s: %w", indexFile, err)
	}
	defer output.Close()
	if err := writeIndexBinary(output, i); err != nil {
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

func writeIndexBinary(file io.Writer, i *index.Index) error {
	return gob.NewEncoder(file).Encode(*i)
}

func search(c *cli.Context) error {
	indexFile := c.String("index")
	file, err := os.Open(indexFile)
	if err != nil {
		return fmt.Errorf("can not open index file %s: %w", indexFile, err)
	}
	index, err := readIndexBinary(file)
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
			log.Printf("%d. %s", i, result.Document.Name)
		}
	}

	return nil
}

func readIndexBinary(file io.Reader) (*index.Index, error) {
	i := &index.Index{}
	return i, gob.NewDecoder(file).Decode(i)
}
