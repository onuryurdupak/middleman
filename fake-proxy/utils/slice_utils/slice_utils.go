package slice_utils

func ContainsString(slice []string, contains string) bool {
	for _, e := range slice {
		if e == contains {
			return true
		}
	}
	return false
}

func RemoveString(slice *[]string, contains string) bool {
	foundIndex := -1
	for i, e := range *slice {
		if e == contains {
			foundIndex = i
			break
		}
	}
	if foundIndex == -1 {
		return false
	}

	tempA := (*slice)[0:foundIndex]
	tempB := (*slice)[foundIndex+1:]

	*slice = append(tempA, tempB...)
	return true
}
