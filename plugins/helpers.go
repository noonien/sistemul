package plugins

import (
	"sort"
)

func sortedUniqueStrings(s []string) []string {
	sort.Strings(s)

	var i int
	ns := make([]string, 0, len(s))
	for j := range s {
		if i > 0 && ns[i-1] == s[j] {
			continue
		}

		ns = append(ns, s[j])
		i++
	}

	return ns
}
