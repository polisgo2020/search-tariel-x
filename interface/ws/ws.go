package ws

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/polisgo2020/search-tariel-x/index"
)

var tpl = `
<html>
	<body>
	<form action="/" method="get">
		<input type="text" name="query">
		<input type="submit" value="Search">
	</form>
	%s
	</body>
</html>
`

type Ws struct {
	i      *index.Index
	server http.Server
}

func New(listen string, timeout time.Duration, i *index.Index) (*Ws, error) {
	if i == nil {
		return nil, errors.New("incorrect index obj")
	}

	if listen == "" {
		return nil, errors.New("incorrect listen interface")
	}

	ws := &Ws{
		i: i,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.handler)

	ws.server = http.Server{
		Addr:         listen,
		Handler:      mux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	return ws, nil
}

func (ws *Ws) handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	var result string
	if query != "" {
		results, err := ws.i.Search(query)
		if err != nil {
			log.Printf("Error search %q over index: %q", query, err)
			fmt.Fprintf(w, "Error search %q over index.", query)
		}
		resultsList := make([]string, 0, len(results))
		for _, result := range results {
			resultsList = append(resultsList, fmt.Sprintf("<li>%s, score %d</li>", result.Document.Name, result.Score))
		}
		result = fmt.Sprintf("<p><ul>%s</ul></p>", strings.Join(resultsList, "\n"))
	}

	fmt.Fprintf(w, tpl, result)
}

func (ws *Ws) Run() error {
	return ws.server.ListenAndServe()
}
