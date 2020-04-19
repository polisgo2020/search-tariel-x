package index

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// DbIndex is postgresql-based engine for storing inverted index.
type DbIndex struct {
	pg             *pg.DB
	tokensCache    map[string]int
	tokensM        sync.RWMutex
	documentsCache map[string]int
	documentsM     sync.RWMutex
	insertC        chan Occurrence
}

// NewDbIndex creates new postgresql-based engine.
// Use the method instead of creating empty struct.
func NewDbIndex(pg *pg.DB) *DbIndex {
	pg.AddQueryHook(dbLogger{})
	i := &DbIndex{
		pg:             pg,
		tokensCache:    map[string]int{},
		tokensM:        sync.RWMutex{},
		documentsCache: map[string]int{},
		documentsM:     sync.RWMutex{},
		insertC:        make(chan Occurrence),
	}
	go i.flush()
	return i
}

// Token is the container for a token in PgSQL.
type Token struct {
	ID    int    `pg:"id,pk"`
	Token string `pg:"token"`
}

// Document is the container for a document in PgSQL.
type Document struct {
	ID   int    `pg:"id,pk"`
	Name string `pg:"name"`
}

// Occurrence is the container for an occurrence in PgSQL.
type Occurrence struct {
	ID         int `pg:"id,pk"`
	TokenID    int `pg:"token_id"`
	DocumentID int `pg:"document_id"`
	Position   int `pg:"position"`
}

func (i *DbIndex) flush() {
	var insertList []Occurrence

	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			if len(insertList) == 0 {
				continue
			}
			if _, err := i.pg.Model(&insertList).Insert(); err != nil {
				log.Err(err).Msg("error inserting rows")
				continue
			}
			log.Info().Msgf("inserted %d occurrences", len(insertList))
			insertList = []Occurrence{}
		case occurrence := <-i.insertC:
			insertList = append(insertList, occurrence)
		}
	}
}

// Add adds new token, document and position to the database.
// If the token or the document has been already inserted the function would take it from cache.
func (i *DbIndex) Add(token string, position int, source Source) error {
	tkn, err := i.getToken(token)
	if err != nil {
		return err
	}
	doc, err := i.getDocument(source.Name)
	if err != nil {
		return err
	}
	i.insertC <- Occurrence{
		TokenID:    tkn.ID,
		DocumentID: doc.ID,
		Position:   position,
	}
	return err
}

func (i *DbIndex) getToken(token string) (*Token, error) {
	i.tokensM.RLock()
	if id, ok := i.tokensCache[token]; ok {
		i.tokensM.RUnlock()
		return &Token{
			ID:    id,
			Token: token,
		}, nil
	}
	i.tokensM.RUnlock()

	tkn := &Token{}
	err := i.pg.Model(tkn).Where("token=?", token).Select()
	if err != nil && err != pg.ErrNoRows {
		return nil, fmt.Errorf("error selecting %s %w", token, err)
	}

	if err == nil {
		return tkn, nil
	}

	i.tokensM.Lock()
	defer i.tokensM.Unlock()
	tkn.Token = token
	if _, err := i.pg.Model(tkn).Returning("*").Insert(); err != nil {
		return nil, fmt.Errorf("error inserting %s %w", token, err)
	}
	log.Debug().Msgf("add token %s %d to cache", token, tkn.ID)
	i.tokensCache[token] = tkn.ID
	return tkn, nil
}

func (i *DbIndex) getDocument(name string) (*Document, error) {
	i.documentsM.RLock()
	if id, ok := i.documentsCache[name]; ok {
		i.documentsM.RUnlock()
		return &Document{
			ID:   id,
			Name: name,
		}, nil
	}
	i.documentsM.RUnlock()

	doc := &Document{}
	err := i.pg.Model(doc).Where("name=?", name).Select()
	if err != nil && err != pg.ErrNoRows {
		return nil, fmt.Errorf("error selecting %s %w", name, err)
	}

	if err == nil {
		return doc, err
	}

	i.documentsM.Lock()
	defer i.documentsM.Unlock()
	doc.Name = name
	if _, err := i.pg.Model(doc).Returning("*").Insert(); err != nil {
		return nil, fmt.Errorf("error inserting %s %w", name, err)
	}
	log.Debug().Msgf("add document %s %d to cache", name, doc.ID)
	i.documentsCache[name] = doc.ID
	return doc, err
}

// Get returns occurrences list for the list of tokens.
func (i *DbIndex) Get(tokens []string) (map[string]Occurrences, error) {
	type item struct {
		Position int    `pg:"position"`
		Token    string `pg:"token"`
		Name     string `pg:"name"`
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

// Close the engine.
func (i *DbIndex) Close() {
	i.pg.Close()
}
