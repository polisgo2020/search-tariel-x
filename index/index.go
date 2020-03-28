// Package index implements inverted index with functions to add document and to search over the build index.
package index

import (
	"bufio"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/reiver/go-porterstemmer"
	"github.com/zoomio/stopwords"
)

// Source contains the name of the file.
type Source struct {
	Name string
}

// Occurrences contain map of document to positions
type Occurrences map[string][]int

type newToken struct {
	source   *Source
	token    string
	position int
}

// Index store list of indexed documents and inverted index.
type Index struct {
	Index   map[string]Occurrences
	Sources map[string]*Source
	chanIn  chan newToken
}

func (i *Index) listen() {
	for t := range i.chanIn {
		i.add(t.token, t.position, t.source)
	}
}

// NewIndex return empty index.
func NewIndex() *Index {
	i := &Index{
		Index:   map[string]Occurrences{},
		Sources: map[string]*Source{},
		chanIn:  make(chan newToken, 10000),
	}
	go i.listen()
	return i
}

// AddSource scan new document and add extracted tokens to the index.
func (i *Index) AddSource(name string, text io.Reader) error {
	source := &Source{Name: name}

	scanner := bufio.NewScanner(text)
	scanner.Split(bufio.ScanWords)
	var position int
	for scanner.Scan() {
		token := i.prepare(scanner.Text())
		if stopwords.IsStopWord(token) {
			continue
		}
		i.chanIn <- newToken{
			source:   source,
			token:    token,
			position: position,
		}
		position++
	}
	return nil
}

func (i *Index) prepare(rawToken string) string {
	token := strings.TrimFunc(rawToken, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	return porterstemmer.StemString(token)
}

func (i *Index) add(token string, position int, source *Source) error {
	if _, ok := i.Sources[source.Name]; !ok {
		i.Sources[source.Name] = source
	}
	if _, ok := i.Index[token]; !ok {
		i.Index[token] = map[string][]int{}
	}
	if _, ok := i.Index[token][source.Name]; !ok {
		i.Index[token][source.Name] = []int{}
	}
	i.Index[token][source.Name] = append(i.Index[token][source.Name], position)
	return nil
}

// Result contains the document description and the score.
type Result struct {
	Document *Source
	Score    int
}

type TmpResultItem struct {
	count       int
	occurrences map[string][]int
	score       int
}

type RangeAlgorithm func(items map[*Source]*TmpResultItem, tokens []string) ([]Result, error)

// ScoreByCount is the default scoring algorithm which ranges search results by count of found tokens.
var ScoreByCount = func(items map[*Source]*TmpResultItem, tokens []string) ([]Result, error) {
	results := make([]Result, 0, len(items))

	for source, item := range items {
		if item.count < len(tokens) {
			continue
		}
		score := 0
		for _, positions := range item.occurrences {
			score += len(positions)
		}
		results = append(results, Result{
			Document: source,
			Score:    score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// Search query over the index.
// The default range algorithm is `ScoreByCount` which ranges search results by count of found tokens.
func (i *Index) Search(query string, rangeAlgorithm RangeAlgorithm) ([]Result, error) {
	rawTokens := strings.FieldsFunc(query, func(r rune) bool {
		return !unicode.IsLetter(r)
	})

	items := map[*Source]*TmpResultItem{}
	tokens := make([]string, 0, len(rawTokens))

	for _, rawToken := range rawTokens {
		token := porterstemmer.StemString(rawToken)
		if stopwords.IsStopWord(token) {
			continue
		}

		occurrences, ok := i.Index[token]
		if !ok {
			return nil, nil
		}
		for document, positions := range occurrences {
			source := i.Sources[document]
			if _, ok := items[source]; !ok {
				items[source] = &TmpResultItem{
					count:       0,
					occurrences: map[string][]int{},
				}
			}

			item := items[source]
			item.count++
			item.occurrences[token] = positions
		}
		tokens = append(tokens, token)
	}

	if rangeAlgorithm == nil {
		return ScoreByCount(items, tokens)
	}

	return rangeAlgorithm(items, tokens)
}
