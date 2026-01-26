package auth

import (
	"errors"
	"regexp"
	"slices"
)

// ValidatePassword 密码强度验证
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}

// IsValidSHA256 验证是否为有效的SHA256哈希字符串
func IsValidSHA256(hash string) bool {
	if len(hash) != 64 {
		return false
	}

	for _, ch := range hash {
		if !((ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'f') ||
			(ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	// RFC 5322 标准的正则（相对严格）
	pattern := `^[a-zA-Z0-9.!#$%&'*+/=?^_{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}

// ValidateNickname 验证昵称格式
func ValidateNickname(nickname string) bool {
	if len(nickname) < 3 || len(nickname) > 20 {
		return false
	}

	for i := 0; i < len(nickname); i++ {
		if nickname[i] > 127 {
			return false
		}
	}

	return true
}

// ValidateRole 验证角色是否合法
func ValidateRole(role string) bool {
	return role == "interviewee" || role == "interviewer"
}

// ValidateStatus 验证面试状态是否合法
func ValidateStatus(status string) bool {
	validStatuses := []string{"r1_pending", "r1_passed", "r2_pending", "r2_passed", "rejected", "offer"}
	return slices.Contains(validStatuses, status)
}

// ValidateEmailCodePurpose 验证邮箱验证码用途
func ValidateEmailCodePurpose(purpose string) bool {
	validPurposes := []string{"register", "reset", "profile"}
	return slices.Contains(validPurposes, purpose)
}

// ValidateDirection 验证方向是否合法
func ValidateDirection(direction string) bool {
	validDirections := []string{"Web", "Pwn", "Reverse", "Crypto", "Misc", "Dev", "Art"}
	return slices.Contains(validDirections, direction)
}

// ValidateDirections 验证方向列表是否合法
func ValidateDirections(directions []string) bool {
	if len(directions) == 0 {
		return false
	}
	for _, dir := range directions {
		if !ValidateDirection(dir) {
			return false
		}
	}
	return true
}
