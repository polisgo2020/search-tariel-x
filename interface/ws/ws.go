package ws

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/polisgo2020/search-tariel-x/index"
)

type Ws struct {
	i         *index.Index
	server    http.Server
	indexTpl  *template.Template
	searchTpl *template.Template
}

func New(listen string, timeout time.Duration, i *index.Index) (*Ws, error) {
	if i == nil {
		return nil, errors.New("incorrect index obj")
	}

	if listen == "" {
		return nil, errors.New("incorrect listen interface")
	}

	indexTpl, err := template.ParseFiles("interface/ws/templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("can not read index template %w", err)
	}
	searchTpl, err := template.ParseFiles("interface/ws/templates/search.html")
	if err != nil {
		return nil, fmt.Errorf("can not read search template %w", err)
	}

	ws := &Ws{
		i:         i,
		indexTpl:  indexTpl,
		searchTpl: searchTpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.indexHandler)
	mux.HandleFunc("/search", ws.searchHandler)

	ws.server = http.Server{
		Addr:         listen,
		Handler:      mux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	return ws, nil
}

func (ws *Ws) indexHandler(w http.ResponseWriter, r *http.Request) {
	ws.indexTpl.Execute(w, nil)
}

func (ws *Ws) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var results []index.Result
	var err error
	if query != "" {
		results, err = ws.i.Search(query)
		if err != nil {
			log.Printf("Error search %q over index: %q", query, err)
			fmt.Fprintf(w, "Error search %q over index.", query)
		}
	}
	ws.searchTpl.Execute(w, struct {
		Results []index.Result
		Query   string
	}{
		Results: results,
		Query:   query,
	})
}

func (ws *Ws) Run() error {
	return ws.server.ListenAndServe()
}
