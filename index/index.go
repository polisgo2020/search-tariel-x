package index

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

type Source struct {
	Name string
}

type Occurences map[string][]int

type Index struct {
	Index   map[*Source]Occurences
	Sources map[string]*Source
}

func NewIndex() *Index {
	return &Index{
		Index:   map[*Source]Occurences{},
		Sources: map[string]*Source{},
	}
}

func (i *Index) AddSource(name string, text io.Reader) error {
	source := &Source{Name: name}

	scanner := bufio.NewScanner(text)
	scanner.Split(bufio.ScanWords)
	var position int
	for scanner.Scan() {
		token := i.prepare(scanner.Text())
		if err := i.add(token, position, source); err != nil {
			return err
		}
		position++
	}
	return nil
}

func (i *Index) prepare(rawToken string) string {
	token := strings.TrimFunc(rawToken, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	return strings.ToLower(token)
}

func (i *Index) add(token string, position int, source *Source) error {
	if _, ok := i.Sources[source.Name]; !ok {
		i.Sources[source.Name] = source
	}
	if _, ok := i.Index[source]; !ok {
		i.Index[source] = map[string][]int{}
	}
	if _, ok := i.Index[source][source.Name]; !ok {
		i.Index[source][source.Name] = []int{}
	}
	i.Index[source][source.Name] = append(i.Index[source][source.Name], position)
	return nil
}
