package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/polisgo2020/search-tariel-x/index"
)

type Cli struct {
	in  *os.File
	out *os.File
	i   *index.Index
}

func New(in *os.File, out *os.File, i *index.Index) (*Cli, error) {
	if in == nil || out == nil || i == nil {
		return nil, errors.New("incorrect in, out interface or index obj")
	}
	return &Cli{
		in:  in,
		out: out,
		i:   i,
	}, nil
}

func (c *Cli) Run() error {
	for {
		reader := bufio.NewReader(c.in)
		query, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("can not read query: %w", err)
		}

		results, err := c.i.Search(query)
		if err != nil {
			return err
		}
		for i, result := range results {
			fmt.Fprintf(c.out, "%d. %s\n", i+1, result.Document.Name)
		}
	}
	return nil
}
