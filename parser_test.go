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
	"fmt"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestingProcessor struct {
	paragraphs []*Structure
	newLines   []*Structure
	marks      []*Structure
	data       []*Token
}

func (tp *TestingProcessor) ProcToken(token *Token, line int, err error) error {
	fmt.Println("TOKEN: ", token)
	tp.data = append(tp.data, token)
	return nil
}

func (tp *TestingProcessor) ProcStruct(strc *Structure, line int, err error) error {
	fmt.Println("STRUCT: ", strc)
	switch strc.Name {
	case "doc":
	case "p":
		tp.paragraphs = append(tp.paragraphs, strc)
	case "nl":
		tp.newLines = append(tp.newLines, strc)
	case "m":
		tp.marks = append(tp.marks, strc)
	}
	return nil
}

func (tp *TestingProcessor) ProcStructClose(strc *StructureClose, line int, err error) error {
	fmt.Println("SCLOSE: ", strc)
	return nil
}

func TestParseVerticalFile(t *testing.T) {
	_, fileName, _, ok := runtime.Caller(0)
	if !ok {
		assert.Fail(t, "failed to determine vertical-generator script")
	}
	filePath := path.Join(path.Dir(fileName), "scripts/genvert.py")
	conf := ParserConf{
		InputFilePath:         "| /usr/bin/python3 " + filePath,
		StructAttrAccumulator: "nil",
	}
	tp := &TestingProcessor{
		paragraphs: make([]*Structure, 0, 20),
		newLines:   make([]*Structure, 0, 20),
		marks:      make([]*Structure, 0, 20),
		data:       make([]*Token, 0, 20),
	}
	err := ParseVerticalFile(&conf, tp)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(tp.paragraphs))
	for i, p := range tp.paragraphs {
		assert.Equal(t, fmt.Sprintf("par%d", i+1), p.Attrs["id"])
	}
	assert.Equal(t, 4, len(tp.newLines))
	for _, nl := range tp.newLines {
		assert.Equal(t, "nl", nl.Name)
		assert.True(t, nl.IsEmpty)
		assert.Equal(t, 0, len(nl.Attrs))
	}
	assert.Equal(t, 4, len(tp.marks))
	for _, m := range tp.marks {
		assert.Equal(t, "m", m.Name)
		assert.True(t, m.IsEmpty)
		assert.Equal(t, 0, len(m.Attrs))
	}
}
