// Package builtin 是随二进制发布的内置资源。
package builtin

import _ "embed"

//go:embed avatar.jpg
var avatar []byte

// Avatar 是内置的默认头像：AVATAR_FILE 没指定时用它，省得部署时先放一张图才能起来。
func Avatar() []byte {
	return avatar
}
