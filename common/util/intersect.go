package util

// IsIntersectMap returns true if subMap is contained in wholeMap
func IsIntersectMap(subMap map[string]string, wholeMap map[string]string) bool {
	result := true
	for k, v := range subMap {
		if wholeMap[k] != v {
			result = false
			break
		}
	}
	return result
}
