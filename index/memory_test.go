package index

import (
	"reflect"
	"sync"
	"testing"
)

func TestMemoryIndex_Add(t *testing.T) {
	i := &MemoryIndex{
		Index:   map[string]MemoryOccurrences{},
		Sources: map[string]*Source{},
		m:       &sync.RWMutex{},
	}
	s1 := Source{Name: "file1"}
	s2 := Source{Name: "file2"}
	if err := i.Add("appl", 0, s1); err != nil {
		t.Error(err)
	}
	if err := i.Add("appl", 0, s2); err != nil {
		t.Error(err)
	}
	if err := i.Add("banana", 1, s1); err != nil {
		t.Error(err)
	}
	if err := i.Add("banana", 1, s2); err != nil {
		t.Error(err)
	}
	if err := i.Add("orang", 2, s2); err != nil {
		t.Error(err)
	}
	if err := i.Add("raspberri", 2, s1); err != nil {
		t.Error(err)
	}

	expected := map[string]MemoryOccurrences{
		"appl":      {"file1": []int{0}, "file2": []int{0}},
		"banana":    {"file1": []int{1}, "file2": []int{1}},
		"orang":     {"file2": []int{2}},
		"raspberri": {"file1": []int{2}},
	}

	if !reflect.DeepEqual(i.Index, expected) {
		t.Errorf("%v is not equal to expected %v", i.Index, expected)
	}
}

func TestMemoryIndex_Get(t *testing.T) {
	i := &MemoryIndex{
		Index:   map[string]MemoryOccurrences{},
		Sources: map[string]*Source{},
		m:       &sync.RWMutex{},
	}
	s1 := Source{Name: "file1"}
	s2 := Source{Name: "file2"}
	if err := i.Add("appl", 0, s1); err != nil {
		t.Error(err)
	}
	if err := i.Add("appl", 0, s2); err != nil {
		t.Error(err)
	}
	if err := i.Add("banana", 1, s1); err != nil {
		t.Error(err)
	}
	if err := i.Add("banana", 1, s2); err != nil {
		t.Error(err)
	}
	if err := i.Add("orang", 2, s2); err != nil {
		t.Error(err)
	}
	if err := i.Add("raspberri", 2, s1); err != nil {
		t.Error(err)
	}

	occurences, err := i.Get([]string{"appl", "banana"})
	if err != nil {
		t.Error(err)
	}

	expected := map[string]Occurrences{
		"appl": {
			i.Sources["file1"]: []int{0},
			i.Sources["file2"]: []int{0},
		},
		"banana": {
			i.Sources["file1"]: []int{1},
			i.Sources["file2"]: []int{1},
		},
	}

	if !reflect.DeepEqual(occurences, expected) {
		t.Errorf("%v is not equal to expected %v", occurences, expected)
	}
}
