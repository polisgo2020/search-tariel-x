package main

import (
	"encoding/gob"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/polisgo2020/search-tariel-x/index"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalln("Invalid number of arguments. Example of call: search /path/to/files /file/with/index")
	}

	files, err := ioutil.ReadDir(os.Args[1])
	if err != nil {
		log.Fatalln("Can not read input directory", err)
	}

	i := index.NewIndex()

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if err := readFile(filepath.Join(os.Args[1], file.Name()), i); err != nil {
			log.Println("Cannot read file", file.Name(), err)
			continue
		}
	}

	output, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatalln("Can not create output file", os.Args[2], err)
	}
	defer output.Close()
	if err := writeIndexBinary(output, i); err != nil {
		log.Fatalln("Can not write index", err)
	}

	outputJsonName := os.Args[2] + ".json"
	outputJson, err := os.Create(outputJsonName)
	if err != nil {
		log.Fatalln("Can not create output file", outputJsonName, err)
	}
	defer outputJson.Close()
	if err := writeIndexJson(outputJson, i); err != nil {
		log.Fatalln("Can not write index", err)
	}
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
	return gob.NewEncoder(file).Encode(i)
}

func writeIndexJson(file io.Writer, i *index.Index) error {
	return json.NewEncoder(file).Encode(i)
}
