package index

import (
	"context"

	"github.com/go-pg/pg/v9"
	"github.com/rs/zerolog/log"
)

type dbLogger struct{}

func (d dbLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	return c, nil
}

func (d dbLogger) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	uq, err := q.FormattedQuery()
	if err != nil {
		return err
	}
	log.Debug().Str("query", uq).Msg("query")
	return nil
}

type DbIndex struct {
	pg *pg.DB
}

func NewDbIndex(pg *pg.DB) *DbIndex {
	pg.AddQueryHook(dbLogger{})
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

func (i *DbIndex) Get(tokens []string) (map[string]Occurrences, error) {
	type item struct {
		Position int    `sql:"position"`
		Token    string `sql:"token"`
		Name     string `sql:"name"`
	}
	var items []item

	_, err := i.pg.Query(
		&items,
		`SELECT position, t.token, d.name FROM occurrences
			JOIN tokens t ON occurrences.token_id = t.id
			JOIN documents d on occurrences.document_id = d.id
			WHERE t.token IN (?);`,
		pg.In(tokens),
	)

	if err != nil {
		return nil, err
	}
	results := map[string]Occurrences{}
	documents := map[string]*Source{}
	for _, item := range items {
		if _, ok := documents[item.Name]; !ok {
			documents[item.Name] = &Source{
				Name: item.Name,
			}
		}
		if _, ok := results[item.Token]; !ok {
			results[item.Token] = Occurrences{}
		}
		doc := documents[item.Name]
		results[item.Token][doc] = append(results[item.Token][doc], item.Position)
	}
	return results, err
}

func (i *DbIndex) Close() {
	i.pg.Close()
}
