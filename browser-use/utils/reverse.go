package utils

func Reverse[T any](list []T) []T {
	for i, j := 0, len(list)-1; i < j; {
		list[i], list[j] = list[j], list[i]
		i++
		j--
	}
	return list
}
