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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenMatchesFilter(t *testing.T) {
	f := [][][]string{
		{
			{"doc.type", "foo"},
			{"doc.type", "bar"},
		},
		{
			{"doc.language", "en"},
			{"doc.language", "cs"},
		},
	}

	tokenMeta := make(map[string]string)
	tokenMeta["doc.type"] = "bar"
	tokenMeta["doc.language"] = "en"
	token := &Token{StructAttrs: tokenMeta}
	ans := token.MatchesFilter(f)
	assert.Equal(t, true, ans)
}

func TestTokenMatchesFilterNoMatch(t *testing.T) {
	f := [][][]string{
		{
			{"doc.type", "foo"},
			{"doc.type", "bar"},
		},
		{
			{"doc.language", "en"},
			{"doc.language", "cs"},
		},
	}

	tokenMeta := make(map[string]string)
	tokenMeta["doc.type"] = "xxx"
	tokenMeta["doc.language"] = "en"
	token := &Token{StructAttrs: tokenMeta}
	ans := token.MatchesFilter(f)
	assert.Equal(t, false, ans)
}

func TestTokenPosAttrByIndex(t *testing.T) {
	tk := Token{
		Idx:   0,
		Word:  "word",
		Attrs: []string{"lemma", "tag", "pos", "func"},
	}
	assert.Equal(t, "word", tk.PosAttrByIndex(0))
	assert.Equal(t, "lemma", tk.PosAttrByIndex(1))
	assert.Equal(t, "tag", tk.PosAttrByIndex(2))
	assert.Equal(t, "pos", tk.PosAttrByIndex(3))
	assert.Equal(t, "func", tk.PosAttrByIndex(4))
	assert.Equal(t, "", tk.PosAttrByIndex(-10))
	assert.Equal(t, "", tk.PosAttrByIndex(80))
}
