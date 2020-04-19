package index

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
)

func TestIndex_AddSource(t *testing.T) {
	i := &Index{
		chanIn: make(chan newToken, 10000),
	}
	i.AddSource("file1", bytes.NewBufferString("an apple banana raspberry"))
	i.AddSource("file2", bytes.NewBufferString("apple the banana orange"))
	close(i.chanIn)

	e := MemoryIndex{
		index:   map[string]MemoryOccurrences{},
		sources: map[string]*Source{},
		m:       &sync.RWMutex{},
	}

	for t := range i.chanIn {
		e.Add(t.token, t.position, t.source)
	}

	expected := map[string]MemoryOccurrences{
		"appl":      {"file1": []int{0}, "file2": []int{0}},
		"banana":    {"file1": []int{1}, "file2": []int{1}},
		"orang":     {"file2": []int{2}},
		"raspberri": {"file1": []int{2}},
	}

	if !reflect.DeepEqual(e.index, expected) {
		t.Errorf("%v is not equal to expected %v", e.index, expected)
	}
}

func TestScoreByCount(t *testing.T) {
	s1 := &Source{Name: "file1"}
	s2 := &Source{Name: "file2"}
	input := map[*Source]*TmpResultItem{
		s1: {
			count: 2,
			occurrences: map[string][]int{
				"appl":   {0},
				"banana": {1},
			},
		},
		s2: {
			count: 2,
			occurrences: map[string][]int{
				"appl":   {0, 2},
				"banana": {1},
			},
		},
	}
	actual, _ := ScoreByCount(input, []string{"appl", "banana"})
	expected := []Result{
		{
			Document: s2,
			Score:    3,
		},
		{
			Document: s1,
			Score:    2,
		},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%v is not equal to expected %v", actual, expected)
	}
}

func TestScoreByCount2(t *testing.T) {
	s1 := &Source{Name: "file1"}
	s2 := &Source{Name: "file2"}
	input := map[*Source]*TmpResultItem{
		s1: {
			count: 1,
			occurrences: map[string][]int{
				"appl": {0},
			},
		},
		s2: {
			count: 2,
			occurrences: map[string][]int{
				"appl":   {0, 2},
				"banana": {1},
			},
		},
	}
	actual, _ := ScoreByCount(input, []string{"appl", "banana"})
	expected := []Result{
		{
			Document: s2,
			Score:    3,
		},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%v is not equal to expected %v", actual, expected)
	}
}

type emptyEngine struct {
	results      map[string]Occurrences
	sourcesCount int
}

func (ee *emptyEngine) Add(token string, position int, source Source) error {
	ee.sourcesCount++
	return nil
}

func (ee *emptyEngine) Get(tokens []string) (map[string]Occurrences, error) {
	return ee.results, nil
}

func (ee *emptyEngine) Close() {}

func TestIndex_Search(t *testing.T) {
	ee := &emptyEngine{}

	i := &Index{
		engine: ee,
		chanIn: make(chan newToken, 10000),
	}
	i.AddSource("file1", bytes.NewBufferString("an apple banana raspberry"))
	i.AddSource("file2", bytes.NewBufferString("apple apple the banana orange"))
	close(i.chanIn)

	s1 := Source{Name: "file1"}
	s2 := Source{Name: "file2"}

	ee.results = map[string]Occurrences{
		"appl": {
			&s1: []int{0},
			&s2: []int{0, 1},
		},
		"banana": {
			&s1: []int{1},
			&s2: []int{2},
		},
	}

	expected := map[*Source]*TmpResultItem{
		&s1: {
			count: 2,
			occurrences: map[string][]int{
				"banana": {1},
				"appl":   {0},
			},
		},
		&s2: {
			count: 2,
			occurrences: map[string][]int{
				"banana": {2},
				"appl":   {0, 1},
			},
		},
	}

	i.rangeAlgorithm = func(actual map[*Source]*TmpResultItem, tokens []string) (results []Result, err error) {
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("%v is not equal to expected %v", actual, expected)
		}
		return nil, nil
	}

	i.Search("the apple banana")
}

func TestNewIndex(t *testing.T) {
	ee := &emptyEngine{}
	i := NewIndex(ee, nil)
	i.AddSource("file1", bytes.NewBufferString("an apple banana raspberry"))
	i.AddSource("file2", bytes.NewBufferString("apple apple the banana orange"))
	close(i.chanIn)

	if ee.sourcesCount != 7 {
		t.Errorf("Count of documents %d != 2", ee.sourcesCount)
	}
}
