// Copyright 2019 Tomas Machalek <tomas.machalek@gmail.com>
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

func TestIsElement(t *testing.T) {
	assert.True(t, isElement("<foo />"))
	assert.True(t, isElement("<foo>"))
	assert.True(t, isElement("</foo>"))
	assert.True(t, isElement("<>"))
	assert.False(t, isElement("<xxx"))
	assert.False(t, isElement("xxx>"))
}

func TestIsOpenElement(t *testing.T) {
	assert.False(t, isOpenElement("<foo />"))
	assert.True(t, isOpenElement("<foo>"))
	assert.False(t, isOpenElement("</foo>"))
	assert.True(t, isOpenElement("<>"))
	assert.False(t, isOpenElement("<xxx"))
	assert.False(t, isOpenElement("xxx>"))
}

func TestIsCloseElement(t *testing.T) {
	assert.False(t, isCloseElement("<foo />"))
	assert.False(t, isCloseElement("<foo>"))
	assert.True(t, isCloseElement("</foo>"))
	assert.False(t, isCloseElement("<>"))
	assert.False(t, isCloseElement("<xxx"))
	assert.False(t, isCloseElement("xxx>"))
}

func TestIsSelfCloseElement(t *testing.T) {
	assert.True(t, isSelfCloseElement("<foo />"))
	assert.False(t, isSelfCloseElement("<foo>"))
	assert.False(t, isSelfCloseElement("</foo>"))
	assert.False(t, isSelfCloseElement("<>"))
	assert.True(t, isSelfCloseElement("</>"))
	assert.False(t, isSelfCloseElement("<xxx"))
	assert.False(t, isSelfCloseElement("xxx>"))
}

func TestParseAttrVal(t *testing.T) {
	attrs := parseAttrVal(`x="200" foo_x="value foo"`)
	assert.Equal(t, "200", attrs["x"])
	assert.Equal(t, "value foo", attrs["foo_x"])
}

func TestParseAttrValInvalid(t *testing.T) {
	attrs := parseAttrVal(`x="200 y=400`)
	assert.Equal(t, 0, len(attrs))
	attrs = parseAttrVal(`x=200 y=400`)
	assert.Equal(t, 0, len(attrs))

	// we don't even accept xml-legal stuff:
	attrs = parseAttrVal(`x= "200" y ="400"`)
	assert.Equal(t, 0, len(attrs))
}
