package core

import "crypto/subtle"

// SecureEqual 比较 token 一类的秘密：长度不等会直接判否，长度相同则走恒定时间比较。
func SecureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
