/*
Package index implements inverted index with thread-safe functions to index new documents, to search over the built
index, to encode and to decode the index.

Usage

To create new empty index instance use NewIndex function

	i := index.NewIndex()

which would create instance in thread-safe way starting internal channel listener to add new tokens.

New document can be added with AddSource function that parse word-by-word text from io.Reader, extract and clean tokens
and add them to the index. AddSource can be called in thread-safe way, e.g.:

	input := bytes.NewBuffer([]byte("input document"))
	err := i.AddSource("document name", input)

To encode index to file system, network, etc. use Encode function with the object which implements Encoder interface.

	encoder := json.NewEncoder(file)
	err := i.Encode(encoder)

To create index from encoded data use Decode function.

	decoder := gob.NewDecoder(file)
	i, err := index.Decode(decoder)

Do not encode and decode index directly with for example json.Marshal because it may lead to data races.
*/
package index

import (
	"bufio"
	"io"
	"sort"
	"strings"
	"sync"
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
	source   Source
	token    string
	position int
}

// Index store list of indexed documents and inverted index.
type Index struct {
	Index          map[string]Occurrences
	Sources        map[string]*Source
	rangeAlgorithm RangeAlgorithm
	chanIn         chan newToken
	m              *sync.RWMutex
}

func (i *Index) listen() {
	for t := range i.chanIn {
		i.add(t.token, t.position, t.source)
	}
}

// NewIndex return empty index.
// Use NewIndex function instead of creating empty instance of index.
func NewIndex(rangeAlgorithm RangeAlgorithm) *Index {
	i := &Index{
		Index:          map[string]Occurrences{},
		Sources:        map[string]*Source{},
		chanIn:         make(chan newToken),
		m:              &sync.RWMutex{},
		rangeAlgorithm: rangeAlgorithm,
	}
	go i.listen()
	return i
}

// AddSource scan new document and add extracted tokens to the index in thread-safe way.
func (i *Index) AddSource(name string, text io.Reader) error {
	source := Source{Name: name}

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

func (i *Index) add(token string, position int, source Source) error {
	i.m.Lock()
	defer i.m.Unlock()
	if _, ok := i.Sources[source.Name]; !ok {
		i.Sources[source.Name] = &source
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

// TmpResultItem is the container for temporary search results produced by the search function.
// Use this container to filter and sort results with custom RangeAlgorithm function.
type TmpResultItem struct {
	count       int
	occurrences map[string][]int
	score       int
}

type RangeAlgorithm func(items map[*Source]*TmpResultItem, tokens []string) ([]Result, error)

// ScoreByCount is the default scoring algorithm which ranges search results by count of found tokens.
func ScoreByCount(items map[*Source]*TmpResultItem, tokens []string) ([]Result, error) {
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
func (i *Index) Search(query string) ([]Result, error) {
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

		i.m.RLock()
		occurrences, ok := i.Index[token]
		i.m.RUnlock()
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

	if i.rangeAlgorithm == nil {
		return ScoreByCount(items, tokens)
	}

	return i.rangeAlgorithm(items, tokens)
}

// Encoder is the interface implemented by the object that can encode data from the Index.
type Encoder interface {
	// Encode must be able to encode data generated by Decode function.
	Encode(e interface{}) error
}

// Encode is the thread-safe function to encode Index.
func (i *Index) Encode(encoder Encoder) error {
	i.m.RLock()
	defer i.m.RUnlock()

	return encoder.Encode(i)
}

// Decoder is the interface implemented by the object that can decode data into the Index.
type Decoder interface {
	// Decode must be able to decode data generated by Encode function.
	Decode(e interface{}) error
}

// Decode is the thread-safe function to extract index from the encoded data.
func Decode(decoder Decoder) (*Index, error) {
	i := NewIndex(nil)
	i.m.Lock()
	defer i.m.Unlock()
	return i, decoder.Decode(i)
}
