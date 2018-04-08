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
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const (
	channelChunkSize          = 250000 // changing the value affects performance (10k...300k ~ 15%)
	logProgressEachNthDefault = 1000000
	LineTypeToken             = "token"
	LineTypeStruct            = "struct"
	LineTypeIgnored           = "ignored"

	AccumulatorTypeStack = "stack"
	AccumulatorTypeComb  = "comb"
	AccumulatorTypeNil   = "nil"

	CharsetISO8859_1   = "iso-8859-1"
	CharsetISO8859_2   = "iso-8859-2"
	CharsetISO8859_3   = "iso-8859-3"
	CharsetISO8859_4   = "iso-8859-4"
	CharsetISO8859_5   = "iso-8859-5"
	CharsetISO8859_6   = "iso-8859-6"
	CharsetISO8859_7   = "iso-8859-7"
	CharsetISO8859_8   = "iso-8859-8"
	CharsetWindows1250 = "windows-1250"
	CharsetWindows1251 = "windows-1251"
	CharsetWindows1252 = "windows-1252"
	CharsetWindows1253 = "windows-1253"
	CharsetWindows1254 = "windows-1254"
	CharsetWindows1255 = "windows-1255"
	CharsetWindows1256 = "windows-1256"
	CharsetWindows1257 = "windows-1257"
	CharsetWindows1258 = "windows-1258"
	CharsetUTF_8       = "utf-8"
)

var (
	tagSrchRegexp  = regexp.MustCompile("^<([\\w\\d\\p{Po}]+)(\\s+.*?|)/?>$")
	attrValRegexp  = regexp.MustCompile("(\\w+)=\"([^\"]+)\"")
	closeTagRegexp = regexp.MustCompile("</([^>]+)\\s*>")
)

// --------------------------------------------------------

// ParserConf contains configuration parameters for
// vertical file parser
type ParserConf struct {

	// Source vertical file (either a plain text file or a gzip one)
	InputFilePath string `json:"inputFilePath"`

	Encoding string `json:"encoding"`

	FilterArgs [][][]string `json:"filterArgs"`

	StructAttrAccumulator string `json:"structAttrAccumulator"`

	LogProgressEachNth int `json:"logProgressEachNth"`
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

// TokenMatchesFilter tests whether a provided token matches
// a filter in Conjunctive normal form encoded as a 3-d list
// E.g.:
// div.author = 'John Doe' AND (div.title = 'Unknown' OR div.title = 'Superunknown')
// encodes as:
// { {{"div.author" "John Doe"}} {{"div.title" "Unknown"} {"div.title" "Superunknown"}} }
func TokenMatchesFilter(token *Token, filterCNF [][][]string) bool {
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

// GetCharmapByName returns a proper Charmap instance based
// on provided encoding name. The name detection is case
// insensitive (e.g. utf-8 is the same as UTF-8). The number
// of supported charsets is
func GetCharmapByName(name string) (*charmap.Charmap, error) {
	switch strings.ToLower(name) {
	case CharsetISO8859_1:
		return charmap.ISO8859_1, nil
	case CharsetISO8859_2:
		return charmap.ISO8859_2, nil
	case CharsetISO8859_3:
		return charmap.ISO8859_3, nil
	case CharsetISO8859_4:
		return charmap.ISO8859_4, nil
	case CharsetISO8859_5:
		return charmap.ISO8859_5, nil
	case CharsetISO8859_6:
		return charmap.ISO8859_6, nil
	case CharsetISO8859_7:
		return charmap.ISO8859_7, nil
	case CharsetISO8859_8:
		return charmap.ISO8859_8, nil
	case CharsetWindows1250:
		return charmap.Windows1250, nil
	case CharsetWindows1251:
		return charmap.Windows1251, nil
	case CharsetWindows1252:
		return charmap.Windows1252, nil
	case CharsetWindows1253:
		return charmap.Windows1253, nil
	case CharsetWindows1254:
		return charmap.Windows1254, nil
	case CharsetWindows1255:
		return charmap.Windows1255, nil
	case CharsetWindows1256:
		return charmap.Windows1256, nil
	case CharsetWindows1257:
		return charmap.Windows1257, nil
	case CharsetWindows1258:
		return charmap.Windows1258, nil
	case CharsetUTF_8:
		return nil, nil
	case "":
		log.Printf("No charset specified, assuming utf-8")
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

func openInputFile(path string) (io.Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	finfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !finfo.Mode().IsRegular() {
		return nil, fmt.Errorf("Path %s is not a regular file", path)
	}

	var rd io.Reader
	if strings.HasSuffix(path, ".gz") {
		rd, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}

	} else {
		rd = f
	}
	return rd, nil
}

// ParseVerticalFile processes a corpus vertical file
// line by line and applies a custom LineProcessor on
// them. The processing is parallelized in the sense
// that reading a file into lines and processing of
// the lines runs in different goroutines. To reduce
// overhead, the data are passed between goroutines
// in chunks.
func ParseVerticalFile(conf *ParserConf, lproc LineProcessor) error {
	logProgressEachNth := logProgressEachNthDefault
	if conf.LogProgressEachNth > 0 {
		logProgressEachNth = conf.LogProgressEachNth
	}
	rd, err := openInputFile(conf.InputFilePath)
	if err != nil {
		return err
	}
	brd := bufio.NewScanner(rd)

	stack, err := createStructAttrAccumulator(conf.StructAttrAccumulator)
	if err != nil {
		return err
	}

	chm, chErr := GetCharmapByName(conf.Encoding)
	if chErr != nil {
		return chErr
	}
	log.Printf("Configured conversion from charset %s", chm)
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
			if progress%logProgressEachNth == 0 {
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
				if TokenMatchesFilter(tk, conf.FilterArgs) {
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
	f, err := os.Open(conf.InputFilePath)
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
