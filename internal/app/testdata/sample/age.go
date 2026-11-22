package sample

func IsEligible(age int, enabled bool) bool {
	return age >= 18 && enabled
}

func Same(a, b int) bool {
	return a == b
}
