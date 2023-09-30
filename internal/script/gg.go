package script

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var matchCases = regexp.MustCompile(`case (\d+):`)
var matchBase = regexp.MustCompile(`b: '(\d+/)'`)
var matchSubdomain = regexp.MustCompile(`(..)(.)$`)

func ParseScript(script string) *Script {
	s := new(Script)
	matches := matchCases.FindAllStringSubmatch(script, -1)
	if matches == nil {
		return nil
	}
	result := make([]int, len(matches))
	for i, match := range matches {
		result[i], _ = strconv.Atoi(match[1])
	}
	s.m = result
	s.BasePath = matchBase.FindStringSubmatch(script)[1]
	if strings.Contains(script, "o = 1; break;") {
		s.base = 1
	}
	return s
}

type Script struct {
	BasePath string
	m        []int
	base     int
}

func (s *Script) M(g int) int {
	if slices.Contains(s.m, g) {
		return s.base
	}
	if s.base == 1 {
		return 0
	}
	return 1
}

func (s *Script) S(h string) string {
	v := matchSubdomain.FindStringSubmatch(h)
	if len(v) >= 3 {
		k, _ := strconv.ParseInt(v[2]+v[1], 16, 64)
		return strconv.FormatInt(k, 10)
	}
	return ""
}

func (s *Script) SubdomainFromURL(url, base string) string {
	val := "b"
	if base != "" {
		val = base
	}

	r := regexp.MustCompile(`/[0-9a-f]{61}([0-9a-f]{2})([0-9a-f])`)
	m := r.FindStringSubmatch(url)
	if m == nil {
		return "a"
	}

	g, err := strconv.ParseInt(m[2]+m[1], 16, 64)
	if err != nil {
		return "a"
	}

	return string(rune(97+s.M(int(g)))) + val
}

func (s *Script) FullPathFromHash(hash string) string {
	return fmt.Sprintf("%s%s/%s", s.BasePath, s.S(hash), hash)
}
