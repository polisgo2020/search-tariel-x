package main

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		if _, err := db.Exec(`CREATE TABLE public.tokens
			(
			  id serial NOT NULL,
			  token text,
			  CONSTRAINT tokens_pk PRIMARY KEY (id)
			);`); err != nil {
			return err
		}
		if _, err := db.Exec(`CREATE TABLE public.documents
			(
			  id serial NOT NULL,
			  name text,
			  CONSTRAINT documents_pk PRIMARY KEY (id)
			);`); err != nil {
			return err
		}
		_, err := db.Exec(`CREATE TABLE public.occurrences
			(
			  id serial NOT NULL,
			  token_id integer,
			  document_id integer,
			  position integer,
			  CONSTRAINT occurrences_pk PRIMARY KEY (id),
			  CONSTRAINT occurrence_document_id FOREIGN KEY (document_id)
				  REFERENCES public.documents (id) MATCH SIMPLE
				  ON UPDATE NO ACTION ON DELETE CASCADE,
			  CONSTRAINT occurrence_token_fk FOREIGN KEY (token_id)
				  REFERENCES public.tokens (id) MATCH SIMPLE
				  ON UPDATE NO ACTION ON DELETE CASCADE
			);`)
		return err
	}, func(db migrations.DB) error {
		if _, err := db.Exec(`DROP TABLE public.words;`); err != nil {
			return err
		}
		if _, err := db.Exec(`DROP TABLE public.documents;`); err != nil {
			return err
		}
		if _, err := db.Exec(`DROP TABLE public.occurrences;`); err != nil {
			return err
		}
		return nil
	})
}
