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

// -------------------------------------------------------

type structAttrs struct {
	elms map[string]*Structure
}

func (sa *structAttrs) Begin(v *Structure) {
	sa.elms[v.Name] = v
}

func (sa *structAttrs) End(name string) *Structure {
	tmp := sa.elms[name]
	delete(sa.elms, name)
	return tmp
}

func (sa *structAttrs) GetAttrs() map[string]string {
	ans := make(map[string]string)
	for k, v := range sa.elms {
		for k2, v2 := range v.Attrs {
			ans[k+"."+k2] = v2
		}
	}
	return ans
}

func (sa *structAttrs) Size() int {
	return len(sa.elms)
}

func newStructAttrs() *structAttrs {
	return &structAttrs{elms: make(map[string]*Structure)}
}

// -------------------------------------------------------

type nilStructAttrs struct{}

func (nsa *nilStructAttrs) Begin(v *Structure) {}

func (nsa *nilStructAttrs) End(name string) *Structure {
	return nil
}

func (nsa *nilStructAttrs) GetAttrs() map[string]string {
	return make(map[string]string)
}

func (nsa *nilStructAttrs) Size() int {
	return 0
}

func newNilStructAttrs() *nilStructAttrs {
	return &nilStructAttrs{}
}
