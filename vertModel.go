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
	"strings"
)

// Token is a representation of
// a parsed line. It connects both, positional attributes
// and currently accumulated structural attributes.
type Token struct {
	Idx         int
	Word        string
	Attrs       []string
	StructAttrs map[string]string
}

// WordLC returns the 'word' positional attribute converted
// to lowercase
func (t *Token) WordLC() string {
	return strings.ToLower(t.Word)
}

// PosAttrByIndex returns a positional attribute based
// on its original index in vertical file
func (t *Token) PosAttrByIndex(idx int) string {
	if idx == 0 {
		return t.Word

	} else if idx > 0 && idx < len(t.Attrs)+1 {
		return t.Attrs[idx-1]
	}
	return ""
}

// MatchesFilter tests whether a provided token matches
// a filter in Conjunctive normal form encoded as a 3-d list
// E.g.:
// div.author = 'John Doe' AND (div.title = 'Unknown' OR div.title = 'Superunknown')
// encodes as:
// { {{"div.author" "John Doe"}} {{"div.title" "Unknown"} {"div.title" "Superunknown"}} }
func (t *Token) MatchesFilter(filterCNF [][][]string) bool {
	var sub bool
	for _, item := range filterCNF {
		sub = false
		for _, v := range item {
			if v[1] == t.StructAttrs[v[0]] {
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

// --------------------------------------------------------

// Structure represent a structure opening tag
type Structure struct {

	// Name defines a name of a structure tag (e.g. 'doc' for <doc> element)
	Name string

	// Attrs store structural attributes of the tag
	// (e.g. <doc id="foo"> produces map with a single key 'id' and value 'foo')
	Attrs map[string]string

	// IsEmpty defines a possible self-closing tag
	// if true then the structure is self-closing
	// (i.e. there is no 'close element' event following)
	IsEmpty bool
}

// --------------------------------------------------------

// StructureClose represent a structure closing tag
type StructureClose struct {
	Name string
}
