/*
Package index implements inverted index with thread-safe functions to index new documents, to search over the built
index, to encode and to decode the index. Inverted index uses in-memory or database engine.

Usage

To create new empty index instance use NewIndex function with created engine. E.g. new index with in-memory engine:

	engine := index.NewMemoryIndex()
	i := index.NewIndex(engine, nil)

which would create instance in thread-safe way starting internal channel listener to add new tokens.

New document can be added with AddSource function that parse word-by-word text from io.Reader, extract and clean tokens
and add them to the index. AddSource can be called in thread-safe way, e.g.:

	input := bytes.NewBuffer([]byte("input document"))
	err := i.AddSource("document name", input)

To encode in-memory index to file system, network, etc. use Encode function with the object which implements Encoder interface.

	encoder := json.NewEncoder(file)
	err := engine.Encode(encoder)

To create in-memory index from encoded data use Decode function.

	decoder := gob.NewDecoder(file)
	engine, err := index.Decode(decoder)
	i := index.NewIndex(engine, nil)

Do not encode and decode in-memory engine directly with for example json.Marshal because it may lead to data races.
*/
package index

import (
	"bufio"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/reiver/go-porterstemmer"
	"github.com/rs/zerolog/log"
	"github.com/zoomio/stopwords"
)

// Source contains the name of the file.
type Source struct {
	Name string
}

// Occurrences contain map of document to positions
type Occurrences map[*Source][]int

type newToken struct {
	source   Source
	token    string
	position int
}

// IndexEngine is the interface for the data storage object.
type IndexEngine interface {
	// Add new token to the storage.
	Add(token string, position int, source Source) error
	// Get list of occurences for the list of tokens.
	Get(tokens []string) (map[string]Occurrences, error)
	// Close the storage.
	Close()
}

// Index uses engine to store the list of indexed documents, the inverted index and search over the index.
type Index struct {
	engine         IndexEngine
	rangeAlgorithm RangeAlgorithm
	chanIn         chan newToken
}

func (i *Index) listen() {
	for t := range i.chanIn {
		if err := i.engine.Add(t.token, t.position, t.source); err != nil {
			log.Error().Err(err).Msgf("error inserting %s %s %d", t.token, t.source, t.position)
		}
	}
}

// NewIndex return empty index.
// Use NewIndex function instead of creating empty instance of index.
func NewIndex(engine IndexEngine, rangeAlgorithm RangeAlgorithm) *Index {
	i := &Index{
		engine:         engine,
		chanIn:         make(chan newToken),
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
		tokens = append(tokens, token)
	}

	occurrencesList, err := i.engine.Get(tokens)
	if err != nil || len(occurrencesList) == 0 {
		return nil, err
	}

	for token, occurrences := range occurrencesList {
		for source, positions := range occurrences {
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
	}

	if i.rangeAlgorithm == nil {
		return ScoreByCount(items, tokens)
	}

	return i.rangeAlgorithm(items, tokens)
}
