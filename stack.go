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
)

type stackItem struct {
	value *Structure
	prev  *stackItem
}

// ---------------------------------------------------------

// Stack represents a data structure used to keep
// vertical file (xml-like) metadata. It is implemented
// as a simple linked list
type stack struct {
	last        *stackItem
	cachedAttrs map[string]string
	dirty       bool
}

// newStack creates a new Stack instance
func newStack() *stack {
	return &stack{
		cachedAttrs: make(map[string]string),
		dirty:       false,
	}
}

// Push adds an item at the beginning
func (s *stack) Begin(value *Structure) error {
	item := &stackItem{value: value, prev: s.last}
	s.last = item
	s.dirty = true
	return nil
}

// Pop takes the first element
func (s *stack) End(name string) (*Structure, error) {
	if name != s.last.value.Name {
		return nil, fmt.Errorf("tag nesting problem: expected %s, found %s", s.last.value.Name, name)
	}
	item := s.last
	s.last = item.prev
	s.dirty = true
	return item.value, nil
}

// Size returns a size of the stack
func (s *stack) Size() int {
	size := 0
	item := s.last
	for {
		if item != nil {
			size++
			item = item.prev

		} else {
			break
		}
	}
	return size
}

// GetAttrs returns all the actual structural attributes
// and their values found on stack.
// Elements are encoded as follows:
// [struct_name].[attr_name]=[value]
// (e.g. doc.author="Isaac Asimov")
func (s *stack) GetAttrs() map[string]string {
	if !s.dirty {
		return s.cachedAttrs
	}
	newAttrs := make(map[string]string, len(s.cachedAttrs))
	curr := s.last
	for curr != nil {
		for k, v := range curr.value.Attrs {
			newAttrs[curr.value.Name+"."+k] = v
		}
		curr = curr.prev
	}
	s.cachedAttrs = newAttrs
	s.dirty = false
	return s.cachedAttrs
}
