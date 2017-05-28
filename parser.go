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

package vertigo

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

const (
	channelChunkSize = 250000 // changing the value affects performance (10k...300k ~ 15%)
	logProgressEach  = 1000000
	LineTypeToken    = "token"
	LineTypeStruct   = "struct"
	LineTypeIgnored  = "ignored"

	AccumulatorTypeStack = "stack"
	AccumulatorTypeComb  = "comb"
	AccumulatorTypeNil   = "nil"

	CharsetISO8859_2   = "ISO-8859-2"
	CharsetWindows1250 = "windows-1250"
	CharsetUTF_8       = "UTF-8"
)

var (
	tagSrchRegexp  = regexp.MustCompile("^<([\\w]+)(\\s*[^>]*?|)/?>$")
	attrValRegexp  = regexp.MustCompile("(\\w+)=\"([^\"]+)\"")
	closeTagRegexp = regexp.MustCompile("</([^>]+)\\s*>")
)

// --------------------------------------------------------

// ParserConf contains configuration parameters for
// vertical file parser
type ParserConf struct {

	// Source vertical file (either a plain text file or a gzip one)
	VerticalFilePath string `json:"verticalFilePath"`

	Encoding string `json:"encoding"`

	FilterArgs [][][]string `json:"filterArgs"`

	StructAttrAccumulator string `json:"structAttrAccumulator"`
}

// LoadConfig loads the configuration from a JSON file.
// In case of an error the program exits with panic.
func LoadConfig(path string) *ParserConf {
	rawData, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var conf ParserConf
	err = json.Unmarshal(rawData, &conf)
	if err != nil {
		panic(err)
	}
	return &conf
}

// --------------------------------------------------------

// Token is a representation of
// a parsed line. It connects both, positional attributes
// and currently accumulated structural attributes.
type Token struct {
	Word        string
	Attrs       []string
	StructAttrs map[string]string
}

func (v *Token) WordLC() string {
	return strings.ToLower(v.Word)
}

// --------------------------------------------------------

type Structure struct {
	Name  string
	Attrs map[string]string
}

// --------------------------------------------------------

type StructureClose struct {
	Name string
}

// --------------------------------------------------------

type LineProcessor interface {
	ProcToken(token *Token)
	ProcStruct(strc *Structure)
	ProcStructClose(strc *StructureClose)
}

// --------------------------------------------------------

// this is quite simplified but it should work for our purposes
func isElement(tagSrc string) bool {
	return strings.HasPrefix(tagSrc, "<") && strings.HasSuffix(tagSrc, ">")
}

func isOpenElement(tagSrc string) bool {
	return isElement(tagSrc) && !strings.HasPrefix(tagSrc, "</") &&
		!strings.HasSuffix(tagSrc, "/>")
}

func isCloseElement(tagSrc string) bool {
	return isElement(tagSrc) && strings.HasPrefix(tagSrc, "</")
}

func isSelfCloseElement(tagSrc string) bool {
	return isElement(tagSrc) && strings.HasSuffix(tagSrc, "/>")
}

func parseAttrVal(src string) map[string]string {
	ans := make(map[string]string)
	srch := attrValRegexp.FindAllStringSubmatch(src, -1)
	for i := 0; i < len(srch); i++ {
		ans[srch[i][1]] = srch[i][2]
	}
	return ans
}

func parseLine(line string, elmStack structAttrAccumulator) interface{} {
	switch {
	case isOpenElement(line):
		srch := tagSrchRegexp.FindStringSubmatch(line)
		meta := &Structure{Name: srch[1], Attrs: parseAttrVal(srch[2])}
		elmStack.Begin(meta)
		return meta
	case isCloseElement(line):
		srch := closeTagRegexp.FindStringSubmatch(line)
		elmStack.End(srch[1])
		return &StructureClose{Name: srch[1]}
	case isSelfCloseElement(line):
		srch := tagSrchRegexp.FindStringSubmatch(line)
		return &Structure{Name: srch[1], Attrs: parseAttrVal(srch[2])}
	default:
		items := strings.Split(line, "\t")
		return &Token{
			Word:        items[0],
			Attrs:       items[1:],
			StructAttrs: elmStack.GetAttrs(),
		}
	}
}

func tokenMatchesFilter(token *Token, filterCNF [][][]string) bool {
	var sub bool
	for _, item := range filterCNF {
		sub = false
		for _, v := range item {
			if v[1] == token.StructAttrs[v[0]] {
				sub = true
				break
			}
		}
		if sub == false {
			return false
		}
	}
	return true
}

func createStructAttrAccumulator(ident string) (structAttrAccumulator, error) {
	switch ident {
	case AccumulatorTypeStack:
		return newStack(), nil
	case AccumulatorTypeComb:
		return newStructAttrs(), nil
	case AccumulatorTypeNil:
		return newNilStructAttrs(), nil
	default:
		return nil, fmt.Errorf("Unknown accumulator type \"%s\"", ident)
	}
}

// SupportedCharsets returns a list of names of
// character sets.
func SupportedCharsets() []string {
	return []string{CharsetISO8859_2, CharsetUTF_8, CharsetWindows1250}
}

func getCharmapByName(name string) (*charmap.Charmap, error) {
	switch strings.ToLower(name) {
	case CharsetISO8859_2:
		return charmap.ISO8859_2, nil
	case CharsetWindows1250:
		return charmap.Windows1250, nil
	case CharsetUTF_8:
		return nil, nil
	default:
		return nil, fmt.Errorf("Unsupported charset '%s'", name)
	}
}

func importString(s string, ch *charmap.Charmap) string {
	if ch == nil {
		return s
	}
	ans, _, _ := transform.String(ch.NewDecoder(), s)
	// TODO handle error
	return ans
}

// ParseVerticalFile processes a corpus vertical file
// line by line and applies a custom LineProcessor on
// them. The processing is parallelized in the sense
// that reading a file into lines and processing of
// the lines runs in different goroutines. To reduce
// overhead, the data are passed between goroutines
// in chunks.
func ParseVerticalFile(conf *ParserConf, lproc LineProcessor) error {
	f, err := os.Open(conf.VerticalFilePath)
	if err != nil {
		return err
	}

	var rd io.Reader
	if strings.HasSuffix(conf.VerticalFilePath, ".gz") {
		rd, err = gzip.NewReader(f)
		if err != nil {
			return err
		}

	} else {
		rd = f
	}
	brd := bufio.NewScanner(rd)

	stack, err := createStructAttrAccumulator(conf.StructAttrAccumulator)
	if err != nil {
		return err
	}

	chm, chErr := getCharmapByName(conf.Encoding)
	if chErr != nil {
		return err

	} else if chm != nil {
		log.Printf("Configured conversion from charset %s", chm)

	} else {
		log.Printf("Assume encoding is utf-8")
	}
	ch := make(chan []interface{})
	chunk := make([]interface{}, channelChunkSize)
	go func() {
		i := 0
		progress := 0
		for brd.Scan() {
			line := parseLine(importString(brd.Text(), chm), stack)
			chunk[i] = line
			i++
			if i == channelChunkSize {
				i = 0
				ch <- chunk
				chunk = make([]interface{}, channelChunkSize)
			}
			progress++
			if progress%logProgressEach == 0 {
				log.Printf("...processed %d lines.\n", progress)
			}
		}
		if i > 0 {
			ch <- chunk[:i]
		}
		close(ch)
	}()

	for tokens := range ch {
		for _, token := range tokens {
			switch token.(type) {
			case *Token:
				tk := token.(*Token)
				if tokenMatchesFilter(tk, conf.FilterArgs) {
					lproc.ProcToken(tk)
				}
			case *Structure:
				lproc.ProcStruct(token.(*Structure))
			case *StructureClose:
				lproc.ProcStructClose(token.(*StructureClose))
			}
		}
	}
	log.Println("Parsing done. Metadata stack size: ", stack.Size())
	return nil
}

//ParseVerticalFileNoGoRo is just for benchmarking purposes
func ParseVerticalFileNoGoRo(conf *ParserConf, lproc LineProcessor) {
	f, err := os.Open(conf.VerticalFilePath)
	if err != nil {
		panic(err)
	}
	rd := bufio.NewScanner(f)
	stack := newStack()

	for rd.Scan() {
		token := parseLine(rd.Text(), stack)
		switch token.(type) {
		case *Token:
			lproc.ProcToken(token.(*Token))
		}
	}

	log.Println("Parsing done. Metadata stack size: ", stack.Size())
}
