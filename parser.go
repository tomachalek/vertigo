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

// parser

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

type ElementNode interface {
	Name() string
}

type Element interface {
	Name() string
	Attrs() map[string]string
}

// -------------------------

type StartElement struct {
	name  string
	attrs map[string]string
}

func (s *StartElement) Stringer() string {
	return fmt.Sprintf("StartElement (%s)", s.name)
}

func (s *StartElement) Name() string {
	return s.name
}

func (s *StartElement) Attrs() map[string]string {
	return s.attrs
}

// --------------------------

type EndElement struct {
	name string
}

func (s *EndElement) Stringer() string {
	return fmt.Sprintf("EndElement (%s)", s.name)
}

func (s *EndElement) Name() string {
	return s.name
}

// ----------------------------

type SelfCloseElement struct {
	name  string
	attrs map[string]string
}

func (s *SelfCloseElement) Stringer() string {
	return fmt.Sprintf("SelfCloseElement (%s)", s.name)
}

func (s *SelfCloseElement) Name() string {
	return s.name
}

func (s *SelfCloseElement) Attrs() map[string]string {
	return s.attrs
}

// -------------------------

type NodeProcessor interface {
	process(elm Element)
}

type Parser struct {
	stack        *Stack
	elmProcessor NodeProcessor
}

func NewParser(processor NodeProcessor) *Parser {
	p := &Parser{stack: NewStack(), elmProcessor: processor}
	return p
}

func parseAttrs(str string) map[string]string {
	return nil
}

func isOpenElement(tagSrc string) bool {
	return strings.HasPrefix(tagSrc, "<") && !strings.HasSuffix(tagSrc, "/>")
}

func isCloseElement(tagSrc string) bool {
	return strings.HasPrefix(tagSrc, "</")
}

func isSelfCloseElement(tagSrc string) bool {
	return strings.HasPrefix(tagSrc, "<") && strings.HasSuffix(tagSrc, "/>")
}

func parseAttrVal(src string) map[string]string {
	ans := make(map[string]string)
	rg := regexp.MustCompile("(\\w+)=\"([^\"]+)\"")
	srch := rg.FindAllStringSubmatch(src, -1)
	for i := 0; i < len(srch); i++ {
		ans[srch[i][1]] = srch[i][2]
	}
	return ans
}

func (p *Parser) parseLine(line string) {
	rg := regexp.MustCompile("<([\\w]+)(\\s*[^>]*)|>")
	srch := rg.FindStringSubmatch(line)
	if len(srch) > 0 {
		fmt.Println("LINE: ", srch[1], srch[2])
	}
	switch {
	case isOpenElement(line):
		elm := StartElement{name: srch[1], attrs: parseAttrVal(srch[2])}
		p.stack.Push(elm)
		p.elmProcessor.process(&elm)
		// intercept
	case isCloseElement(line):
		p.stack.Push(EndElement{name: srch[1]})
		// intercept
		p.stack.Pop()
	case isSelfCloseElement(line):
		p.stack.Pop() //Push(SelfCloseElement{name: srch[1]})
	}
}

func (p *Parser) Parse(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		p.parseLine(scanner.Text())
	}
}
