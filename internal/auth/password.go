// Package auth 提供密码加密与 JWT 签发/校验能力
// 作者: wym
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword 用 bcrypt 对明文密码做单向哈希，成本因子使用默认值（10）
// 存库的是这个哈希值，绝不存明文密码
func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// CheckPassword 校验明文密码是否与库中哈希匹配
// 返回 nil 表示密码正确
func CheckPassword(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
