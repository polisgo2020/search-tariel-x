# Inverted index search

github.com/polisgo2020/search-tariel-x implements inverted index to perform full-text search.
It uses in-memory or PostgreSQL storage engine.

## Build

```bash
go build -o search
```

## Examples of usage

### Build file with index

```bash
./search build file --sources ~/path/to/text/files/ --index index.data
```

or use JSON encoder:

```bash
./search build file --sources ~/path/to/text/files/ --index index.data --json
```

### Search over the index file with CLI.

```bash
./search search file --index index.data
```

### Search over the index file with web interface.

```bash
./search search file --index index.data --listen 0.0.0.0:8080
```

or

```bash
LISTEN=0.0.0.0:8080 ./search search file --index index.data
```

### Use PostgreSQL

Create migrations with [migrations package](migrations/README.md).

Build:

```bash
./search build db --sources ~/path/to/text/files/ --pg postgres://login:pass@localhost:5432/idx?sslmode=disable
```

Search:

```bash
./search search db --pg postgres://login:pass@localhost:5432/idx?sslmode=disable
```

## List of environment variables:

- `PGSQL`, example `postgres://login:pass@localhost:5432/idx?sslmode=disable`
- `LOG_LEVEL`, default `debug`
- `LISTEN`, example `0.0.0.0:8080`

## Usage in external projects:

Download

```bash
go get github.com/polisgo2020/search-tariel-x/index
```

Use

```go
package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/polisgo2020/search-tariel-x/index"
)

func main() {
	engine := index.NewMemoryIndex()
	i := index.NewIndex(engine, nil)

	input := bytes.NewBuffer([]byte("input document with tokens to search"))
	i.AddSource("document1", input)

	// Sleep is needed to ensure that document is added to the index becase AddSource is async operation.
	time.Sleep(time.Second)

	results, _ := i.Search("tokens to search")
	for _, result := range results {
		fmt.Printf("%s\n", result.Document.Name)
	}
}
```