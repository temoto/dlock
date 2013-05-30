package main

func stringListFind(a []string, s string) int {
	for i := 0; i < len(a); i++ {
		if a[i] == s {
			return i
		}
	}
	return -1
}

// https://code.google.com/p/go-wiki/wiki/SliceTricks
func stringListRemove(a []string, s string) []string {
	for i := 0; i < len(a); i++ {
		if a[i] == s {
			copy(a[i:], a[i+1:])
			a[len(a)-1] = ""
			a = a[:len(a)-1]
			break
		}
	}
	return a
}
