package sliceutil

func StringInSlice(value string, slc []string) bool {
	for _, s := range slc {
		if s == value {
			return true
		}
	}

	return false
}
