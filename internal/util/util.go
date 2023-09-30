package util

import "crypto/sha256"

func compareByteSlices(slice1, slice2 []byte) int {
	minLength := len(slice1)
	if len(slice2) < minLength {
		minLength = len(slice2)
	}

	for i := 0; i < minLength; i++ {
		if slice1[i] < slice2[i] {
			return -1
		} else if slice1[i] > slice2[i] {
			return 1
		}
	}

	return 0
}

func SliceContains(S [][]byte, E []byte) (bool, int) {
	cmpResult := -1
	i := 0
	for i = 0; i < len(S); i++ {
		cmpResult = compareByteSlices(E, S[i])
		if cmpResult <= 0 {
			break
		}
	}
	return cmpResult == 0, i
}

func IsLeaf(S []int) bool {
	for i := 0; i < len(S); i++ {
		if S[i] != 0 {
			return false
		}
	}
	return true
}

func HashTerm(query string) []byte {
	bytes := sha256.Sum256([]byte(query))
	return bytes[0:4]
}
