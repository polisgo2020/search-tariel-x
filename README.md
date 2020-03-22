# search

github.com/polisgo2020/search-tariel-x implements inverted index to perform full-text search.

## Build

```bash
go build -o search
```

## Usage

### Build index

```bash
./search build --sources ~/path/to/text/files/ --index output
```

### Search over the index

```bash
./search search --index output
```
