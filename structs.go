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

	"github.com/rs/zerolog/log"
)

// -------------------------------------------------------

type structAttrs struct {
	elms        map[string]*Structure
	cachedAttrs map[string]string
	dirty       bool
}

func (sa *structAttrs) Begin(v *Structure) error {
	_, ok := sa.elms[v.Name]
	if ok {
		return fmt.Errorf("recursive structures not supported (element %s)", v.Name)
	}
	sa.elms[v.Name] = v
	sa.dirty = true
	return nil
}

func (sa *structAttrs) End(name string) (*Structure, error) {
	tmp, ok := sa.elms[name]
	if !ok {
		return nil, fmt.Errorf("cannot close unopened structure %s", name)
	}
	delete(sa.elms, name)
	sa.dirty = true
	return tmp, nil
}

func (sa *structAttrs) GetAttrs() map[string]string {
	if !sa.dirty {
		return sa.cachedAttrs
	}
	newAttrs := make(map[string]string, len(sa.cachedAttrs))
	for k, v := range sa.elms {
		for k2, v2 := range v.Attrs {
			newAttrs[k+"."+k2] = v2
		}
	}
	sa.cachedAttrs = newAttrs
	sa.dirty = false
	return sa.cachedAttrs
}

func (sa *structAttrs) Size() int {
	return len(sa.elms)
}

func newStructAttrs() *structAttrs {
	return &structAttrs{
		elms:        make(map[string]*Structure),
		cachedAttrs: make(map[string]string),
		dirty:       false,
	}
}

// -------------------------------------------------------

// nilStructAttrs can be used e.g. in case user is not
// interested in attaching complete structural attr. information
// to each token and wants to use a custom struct. attr processing
// instead. In such case a significant amount of memory can be
// saved.
type nilStructAttrs struct {
	attrs map[string]string
}

func (nsa *nilStructAttrs) Begin(v *Structure) error {
	return nil
}

func (nsa *nilStructAttrs) End(name string) (*Structure, error) {
	return &Structure{Name: name}, nil
}

func (nsa *nilStructAttrs) GetAttrs() map[string]string {
	return nsa.attrs
}

func (nsa *nilStructAttrs) Size() int {
	return 0
}

func newNilStructAttrs() *nilStructAttrs {
	log.Warn().Msg("using nil structattr accumulator")
	return &nilStructAttrs{
		attrs: make(map[string]string),
	}
}
