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
	"regexp"
	"strings"
)

var (
	tagSrchRegexp   = regexp.MustCompile(`^<([\w\d\p{Po}]+)(\s+.*?|)>$`)
	tagSrchRegexpSC = regexp.MustCompile(`^<([\w\d\p{Po}]+)(\s+.*?|)/>$`)
	attrValRegexp   = regexp.MustCompile(`(\w+)="([^"]+)"`)
	closeTagRegexp  = regexp.MustCompile(`</([^>]+)\s*>`)
)

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

func parseLine(normLine string, elmStack structAttrAccumulator) (any, error) {
	normLine = strings.TrimRight(normLine, "\n\r ")
	switch {
	case isOpenElement(normLine):
		srch := tagSrchRegexp.FindStringSubmatch(normLine)
		if len(srch) < 3 {
			return nil, fmt.Errorf("cannot parse open element '%s'", normLine)
		}
		meta := &Structure{Name: srch[1], Attrs: parseAttrVal(srch[2])}
		err := elmStack.Begin(meta)
		return meta, err
	case isCloseElement(normLine):
		srch := closeTagRegexp.FindStringSubmatch(normLine)
		if len(srch) < 2 {
			return nil, fmt.Errorf("cannot parse close element '%s'", normLine)
		}
		elm, err := elmStack.End(srch[1])
		if err != nil {
			return nil, err
		}
		return &StructureClose{Name: elm.Name}, nil
	case isSelfCloseElement(normLine):
		srch := tagSrchRegexpSC.FindStringSubmatch(normLine)
		if len(srch) < 3 {
			return nil, fmt.Errorf("cannot parse self closing element '%s'", normLine)
		}
		return &Structure{Name: srch[1], Attrs: parseAttrVal(srch[2]), IsEmpty: true}, nil
	default:
		items := strings.Split(normLine, "\t")
		return &Token{
			Word:        items[0],
			Attrs:       items[1:],
			StructAttrs: elmStack.GetAttrs(),
		}, nil
	}
}
