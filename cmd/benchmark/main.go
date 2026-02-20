// Copyright 2017 Tomas Machalek <tomas.machalek@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	vertigo "github.com/tomachalek/vertigo/v6"
)

const top = 10

// tagCounter implements vertigo.LineProcessor and counts occurrences
// of values found at a given positional attribute column index.
type tagCounter struct {
	colIdx int
	counts map[string]int
	tokens int
}

func (tc *tagCounter) ProcToken(token *vertigo.Token, line int, err error) error {
	if err != nil {
		return err
	}
	tc.tokens++
	val := token.PosAttrByIndex(tc.colIdx)
	tc.counts[val]++
	return nil
}

func (tc *tagCounter) ProcStruct(strc *vertigo.Structure, line int, err error) error {
	return err
}

func (tc *tagCounter) ProcStructClose(strc *vertigo.StructureClose, line int, err error) error {
	return err
}

type tagCount struct {
	tag   string
	count int
}

func (tc *tagCounter) topN(n int) []tagCount {
	all := make([]tagCount, 0, len(tc.counts))
	for tag, cnt := range tc.counts {
		all = append(all, tagCount{tag, cnt})
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].count > all[j].count
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: benchmark <vertical-file> <column-index>\n")
		os.Exit(1)
	}

	colIdx, err := strconv.Atoi(os.Args[2])
	if err != nil || colIdx < 0 {
		fmt.Fprintf(os.Stderr, "column-index must be a non-negative integer\n")
		os.Exit(1)
	}

	conf := &vertigo.ParserConf{
		InputFilePath:         os.Args[1],
		StructAttrAccumulator: vertigo.AccumulatorTypeComb,
	}

	proc := &tagCounter{
		colIdx: colIdx,
		counts: make(map[string]int),
	}

	fmt.Printf("Parsing %s (column %d)...\n", conf.InputFilePath, colIdx)
	start := time.Now()

	if err := vertigo.ParseVerticalFile(context.Background(), conf, proc); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)

	fmt.Printf("\nDone in %s\n", elapsed)
	fmt.Printf("Total tokens processed: %d\n", proc.tokens)
	fmt.Printf("\nTop %d values at column %d:\n", top, colIdx)
	fmt.Printf("%-6s  %s\n", "count", "value")
	fmt.Printf("%-6s  %s\n", "------", "-----")
	for _, item := range proc.topN(top) {
		fmt.Printf("%-6d  %s\n", item.count, item.tag)
	}
}
