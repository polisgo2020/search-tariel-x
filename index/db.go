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

type Token struct {
	ID    int    `sql:"id,pk"`
	Token string `sql:"token"`
}

type Document struct {
	ID   int    `sql:"id,pk"`
	Name string `sql:"name"`
}

type Occurrence struct {
	ID         int `sql:"id,pk"`
	TokenID    int `sql:"token_id"`
	DocumentID int `sql:"document_id"`
	Position   int `sql:"position"`
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
	occurrence := Occurrence{
		TokenID:    tkn.ID,
		DocumentID: doc.ID,
		Position:   position,
	}
	_, err = i.pg.Model(&occurrence).Returning("*").Insert()
	return err
}

func (i *DbIndex) getToken(token string) (*Token, error) {
	tkn := &Token{Token: token}
	err := i.pg.Select(tkn)
	return tkn, err
}

func (i *DbIndex) getDocument(name string) (*Document, error) {
	doc := &Document{Name: name}
	err := i.pg.Select(doc)
	return doc, err
}

func (i *DbIndex) Get(token string) (Occurrences, error) {
	_, err := i.pg.Query(
		pg.Scan(&id),
		`SELECT occurences.file_id, SUM(occurences.count) as sum, 
					array_agg(occurences.word_id) as words 
				FROM occurences
				JOIN words on occurences.word_id = words.id
				WHERE words.word IN ('hello', 'cat')
				GROUP BY occurences.file_id
				ORDER BY sum DESC
			)`,
		comment.Author, comment.Content, comment.PostID,
	)
	return nil, err
}

func (i *DbIndex) Close() {
	i.pg.Close()
}
