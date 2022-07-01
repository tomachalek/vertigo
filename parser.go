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
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var (
	vertCmdSplit = regexp.MustCompile(`\s+`)
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

// ------

type structAttrAccumulator interface {
	Begin(value *Structure) error
	End(name string) (*Structure, error)
	GetAttrs() map[string]string
	Size() int
}

// --------------------------------------------------------

// LineProcessor describes an object able to handle
// Vertigo's parsing events.
type LineProcessor interface {

	// ProcToken is called each time the parser encounters a positional
	// attribute. In case parsing produces an error, it is passed to the
	// function without stopping the whole process.
	// In case the function returns an error, the parser stops
	// (in the simplest case it can be even the error it recieves)
	ProcToken(token *Token, line int, err error) error

	// ProcStruct is called each time parser encounters a structure opening
	// element (e.g. <doc>). In case parsing produces an error, it is passed
	// to the function without stopping the whole process.
	// In case the function returns an error, the parser stops.
	ProcStruct(strc *Structure, line int, err error) error

	// ProcStructClose is called each time parser encouters a structure
	// closing element (e.g. </doc>). In case parsing produces an error,
	// it is passed to the function without stopping the whole process.
	// In case the function returns an error, the parser stops.
	ProcStructClose(strc *StructureClose, line int, err error) error
}

// ----

type procItem struct {
	idx   int
	value interface{}
	err   error
}

// --------------------------------------------------------

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
// the lines runs in different goroutines. But the
// function as a whole behaves synchronously - i.e.
// once it returns a value, the processing is finished.
func ParseVerticalFile(conf *ParserConf, lproc LineProcessor) error {

	chm, chErr := GetCharmapByName(conf.Encoding)
	if chErr != nil {
		return chErr
	}
	log.Printf("Configured conversion from charset %s", chm)

	if strings.HasPrefix(conf.InputFilePath, "|") {
		script := vertCmdSplit.Split(conf.InputFilePath, -1)
		if len(script) < 2 {
			return fmt.Errorf("invalid dynamically generated vertical file specification")
		}
		cmd := exec.Command(script[1], script[2:]...)
		cmd.Env = os.Environ()
		rd, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		brd := bufio.NewScanner(rd)

		if err = cmd.Start(); err != nil {
			return err
		}
		if err = parseVerticalFromScanner(brd, chm, conf, lproc); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}

	} else {
		rd, err := openInputFile(conf.InputFilePath)
		if err != nil {
			return err
		}
		brd := bufio.NewScanner(rd)
		if err = parseVerticalFromScanner(brd, chm, conf, lproc); err != nil {
			return err
		}
	}
	return nil
}

func parseVerticalFromScanner(
	brd *bufio.Scanner, chm *charmap.Charmap, conf *ParserConf, lproc LineProcessor) error {
	ch := make(chan []procItem)
	chunk := make([]procItem, channelChunkSize)
	stop := make(chan struct{})
	defer close(stop)

	stack, err := createStructAttrAccumulator(conf.StructAttrAccumulator)
	if err != nil {
		return err
	}
	logProgressEachNth := logProgressEachNthDefault
	if conf.LogProgressEachNth > 0 {
		logProgressEachNth = conf.LogProgressEachNth
	}
	go func() {
		defer close(ch)
		i := 0
		lineNum := 0
		tokenNum := 0

		for brd.Scan() {
			line, parseErr := parseLine(importString(brd.Text(), chm), stack)
			tok, isTok := line.(*Token)
			if isTok {
				tok.Idx = tokenNum
				tokenNum++
			}
			chunk[i] = procItem{idx: lineNum, value: line, err: parseErr}
			i++
			if i == channelChunkSize {
				i = 0
				ch <- chunk
				chunk = make([]procItem, channelChunkSize)
			}
			if lineNum%logProgressEachNth == 0 {
				log.Printf("...processed %d lines.\n", lineNum)
			}
			lineNum++
			select {
			case <-stop:
				fmt.Println("STOPPING PARSING")
				return
			default:
			}
		}
		if i > 0 {
			ch <- chunk[:i]
		}
	}()

	var procErr error
	for items := range ch {
		for _, item := range items {
			switch item.value.(type) {
			case *Token:
				tk := item.value.(*Token)
				if tk.MatchesFilter(conf.FilterArgs) {
					procErr = lproc.ProcToken(tk, item.idx, err)
				}
			case *Structure:
				procErr = lproc.ProcStruct(item.value.(*Structure), item.idx, item.err)
			case *StructureClose:
				procErr = lproc.ProcStructClose(item.value.(*StructureClose), item.idx, item.err)
			}
			if procErr != nil {
				return procErr
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
	i := 0
	for rd.Scan() {
		token, err := parseLine(rd.Text(), stack)
		switch token.(type) {
		case *Token:
			lproc.ProcToken(token.(*Token), i, err)
		}
		i++
	}

	log.Println("Parsing done. Metadata stack size: ", stack.Size())
}
