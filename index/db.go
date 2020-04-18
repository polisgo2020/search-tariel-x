package index

import (
	"github.com/go-pg/pg/v9"
)

type DbIndex struct {
	pg *pg.DB
}

func NewDbIndex(pg *pg.DB) *DbIndex {
	i := &DbIndex{
		pg: pg,
	}
	return i
}

type ModelToken struct {
	ID    int    `sql:"id,pk"`
	Token string `sql:"token"`
}

type ModelDocument struct {
	ID   int    `sql:"id,pk"`
	Name string `sql:"name"`
}

type ModelOccurrence struct {
	ID       int `sql:"id,pk"`
	WordID   int `sql:"word_id"`
	FileID   int `sql:"file_id"`
	Position int `sql:"position"`
}

func (i *DbIndex) Add(token string, position int, source Source) error {
	tkn, err := i.getToken(token)
	if err != nil {
		return err
	}
	doc, err := i.getDocument(source.Name)
	if err != nil {
		return err
	}
	occurrence := ModelOccurrence{
		WordID:   tkn.ID,
		FileID:   doc.ID,
		Position: position,
	}
	_, err = i.pg.Model(&occurrence).Returning("*").Insert()
	return err
}

func (i *DbIndex) getToken(token string) (*ModelToken, error) {
	tkn := &ModelToken{Token: token}
	err := i.pg.Select(tkn)
	return tkn, err
}

func (i *DbIndex) getDocument(name string) (*ModelDocument, error) {
	doc := &ModelDocument{Name: name}
	err := i.pg.Select(doc)
	return doc, err
}

func (i *DbIndex) Get(token string) (Occurrences, error) {
	return nil, nil
}

func (i *DbIndex) Close() {
	i.pg.Close()
}
