package index

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
)

func TestIndex_AddSource(t *testing.T) {
	i := &Index{
		Index:   map[string]Occurrences{},
		Sources: map[string]*Source{},
		chanIn:  make(chan newToken, 10000),
		m:       &sync.RWMutex{},
	}
	i.AddSource("file1", bytes.NewBufferString("an apple banana raspberry"))
	i.AddSource("file2", bytes.NewBufferString("apple the banana orange"))
	close(i.chanIn)

	for t := range i.chanIn {
		i.add(t.token, t.position, t.source)
	}

	expected := map[string]Occurrences{
		"appl":      Occurrences{"file1": []int{0}, "file2": []int{0}},
		"banana":    Occurrences{"file1": []int{1}, "file2": []int{1}},
		"orang":     Occurrences{"file2": []int{2}},
		"raspberri": Occurrences{"file1": []int{2}},
	}

	if !reflect.DeepEqual(i.Index, expected) {
		t.Errorf("%v is not equal to expected %v", i.Index, expected)
	}
}

func TestIndex_Add(t *testing.T) {
	i := &Index{
		Index:   map[string]Occurrences{},
		Sources: map[string]*Source{},
		m:       &sync.RWMutex{},
	}
	s1 := Source{Name: "file1"}
	s2 := Source{Name: "file2"}
	i.add("appl", 0, s1)
	i.add("appl", 0, s2)
	i.add("banana", 1, s1)
	i.add("banana", 1, s2)
	i.add("orang", 2, s2)
	i.add("raspberri", 2, s1)

	expected := map[string]Occurrences{
		"appl":      Occurrences{"file1": []int{0}, "file2": []int{0}},
		"banana":    Occurrences{"file1": []int{1}, "file2": []int{1}},
		"orang":     Occurrences{"file2": []int{2}},
		"raspberri": Occurrences{"file1": []int{2}},
	}

	if !reflect.DeepEqual(i.Index, expected) {
		t.Errorf("%v is not equal to expected %v", i.Index, expected)
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

func TestIndex_Search(t *testing.T) {
	i := &Index{
		Index:   map[string]Occurrences{},
		Sources: map[string]*Source{},
		m:       &sync.RWMutex{},
		chanIn:  make(chan newToken, 10000),
	}
	i.AddSource("file1", bytes.NewBufferString("an apple banana raspberry"))
	i.AddSource("file2", bytes.NewBufferString("apple apple the banana orange"))
	close(i.chanIn)

	for t := range i.chanIn {
		i.add(t.token, t.position, t.source)
	}

	expected := map[*Source]*TmpResultItem{
		i.Sources["file1"]: {
			count: 2,
			occurrences: map[string][]int{
				"appl":   {0},
				"banana": {1},
			},
		},
		i.Sources["file2"]: {
			count: 2,
			occurrences: map[string][]int{
				"appl":   {0, 1},
				"banana": {2},
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

func TestIndex_Search2(t *testing.T) {
	i := &Index{
		Index:   map[string]Occurrences{},
		Sources: map[string]*Source{},
		m:       &sync.RWMutex{},
		chanIn:  make(chan newToken, 10000),
	}
	i.AddSource("file1", bytes.NewBufferString("an apple banana raspberry"))
	i.AddSource("file2", bytes.NewBufferString("apple apple the banana orange"))
	close(i.chanIn)

	for t := range i.chanIn {
		i.add(t.token, t.position, t.source)
	}

	expected := map[*Source]*TmpResultItem{}

	i.rangeAlgorithm = func(actual map[*Source]*TmpResultItem, tokens []string) (results []Result, err error) {
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("%v is not equal to expected %v", actual, expected)
		}
		return nil, nil
	}

	i.Search("the window apple")
}

func TestNewIndex(t *testing.T) {
	i := NewIndex(nil)
	i.AddSource("file1", bytes.NewBufferString("an apple banana raspberry"))
	i.AddSource("file2", bytes.NewBufferString("apple apple the banana orange"))
	close(i.chanIn)

	if len(i.Sources) != 2 {
		t.Errorf("Count of documents %d != 2", len(i.Sources))
	}
}
