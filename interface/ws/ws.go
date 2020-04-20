package ws

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/polisgo2020/search-tariel-x/index"
	"github.com/rs/zerolog/log"
)

type Ws struct {
	listen    string
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
		listen:    listen,
		i:         i,
		indexTpl:  indexTpl,
		searchTpl: searchTpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.indexHandler)
	mux.HandleFunc("/search", ws.searchHandler)

	logMw := logMiddleware(mux)

	ws.server = http.Server{
		Addr:         listen,
		Handler:      logMw,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	return ws, nil
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		log.Debug().
			Str("method", r.Method).
			Str("remote", r.RemoteAddr).
			Str("path", r.URL.Path).
			Int("duration", int(time.Since(start))).
			Msgf("Called url %s", r.URL.Path)
	})
}

func (ws *Ws) indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := ws.indexTpl.Execute(w, nil); err != nil {
		log.Error().Err(err).Msg("error rendering template")
	}
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
	if err := ws.searchTpl.Execute(w, struct {
		Results []index.Result
		Query   string
	}{
		Results: results,
		Query:   query,
	}); err != nil {
		log.Error().Err(err).Msg("error rendering template")
	}
}

func (ws *Ws) Run() error {
	log.Info().Str("interface", ws.listen).Msg("started to listen")
	return ws.server.ListenAndServe()
}
