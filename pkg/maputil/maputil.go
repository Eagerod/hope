package maputil

func MapStringBoolKeys(m *map[string]bool) *[]string {
	rv := make([]string, len(*m))
	i := 0
	for key := range *m {
		rv[i] = key
		i += 1
	}

	return &rv
}
