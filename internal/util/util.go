package util

import "bytes"

func SliceContains(S [][]byte, E []byte) (bool, int) {
	for i := 0; i < len(S); i++ {
		if t := bytes.Compare(S[i], E); t >= 0 {
			if t == 0 {
				return true, i
			}
			return false, i
		}
	}
	return false, 0
}

func IsLeaf(S []int) bool {
	for i := 0; i < len(S); i++ {
		if S[i] != 0 {
			return false
		}
	}
	return true
}
