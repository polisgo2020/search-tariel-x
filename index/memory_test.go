package index

import (
	"reflect"
	"sync"
	"testing"
)

func TestMemoryIndex_Add(t *testing.T) {
	i := &MemoryIndex{
		index:   map[string]MemoryOccurrences{},
		sources: map[string]*Source{},
		m:       &sync.RWMutex{},
	}
	s1 := Source{Name: "file1"}
	s2 := Source{Name: "file2"}
	i.Add("appl", 0, s1)
	i.Add("appl", 0, s2)
	i.Add("banana", 1, s1)
	i.Add("banana", 1, s2)
	i.Add("orang", 2, s2)
	i.Add("raspberri", 2, s1)

	expected := map[string]MemoryOccurrences{
		"appl":      {"file1": []int{0}, "file2": []int{0}},
		"banana":    {"file1": []int{1}, "file2": []int{1}},
		"orang":     {"file2": []int{2}},
		"raspberri": {"file1": []int{2}},
	}

	if !reflect.DeepEqual(i.index, expected) {
		t.Errorf("%v is not equal to expected %v", i.index, expected)
	}
}

func TestMemoryIndex_Get(t *testing.T) {
	i := &MemoryIndex{
		index:   map[string]MemoryOccurrences{},
		sources: map[string]*Source{},
		m:       &sync.RWMutex{},
	}
	s1 := Source{Name: "file1"}
	s2 := Source{Name: "file2"}
	i.Add("appl", 0, s1)
	i.Add("appl", 0, s2)
	i.Add("banana", 1, s1)
	i.Add("banana", 1, s2)
	i.Add("orang", 2, s2)
	i.Add("raspberri", 2, s1)

	occurences, err := i.Get([]string{"appl", "banana"})
	if err != nil {
		t.Error(err)
	}

	expected := map[string]Occurrences{
		"appl": {
			i.sources["file1"]: []int{0},
			i.sources["file2"]: []int{0},
		},
		"banana": {
			i.sources["file1"]: []int{1},
			i.sources["file2"]: []int{1},
		},
	}

	if !reflect.DeepEqual(occurences, expected) {
		t.Errorf("%v is not equal to expected %v", occurences, expected)
	}
}
