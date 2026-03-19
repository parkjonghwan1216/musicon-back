package search

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var chosungs = []rune{
	'ㄱ', 'ㄲ', 'ㄴ', 'ㄷ', 'ㄸ', 'ㄹ', 'ㅁ', 'ㅂ', 'ㅃ',
	'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅉ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ',
}

// ExtractChosung 은 문자열에서 한글 초성을 추출합니다.
// 한글 음절은 초성으로 변환하고, 나머지 문자는 그대로 유지합니다.
func ExtractChosung(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			idx := (r - 0xAC00) / 28 / 21
			result.WriteRune(chosungs[idx])
		} else if utf8.ValidRune(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ExtractEnglishInitials 는 영문 문자열에서 각 단어의 첫 글자를 소문자로 추출합니다.
// 예: "I Want You" → "iwy"
func ExtractEnglishInitials(s string) string {
	var result strings.Builder
	words := strings.Fields(s)
	for _, w := range words {
		for _, r := range w {
			if unicode.IsLetter(r) {
				result.WriteRune(unicode.ToLower(r))
				break
			}
		}
	}
	return result.String()
}

// IsChosung 은 문자가 한글 초성(자음)인지 확인합니다.
func IsChosung(r rune) bool {
	return r >= 0x3131 && r <= 0x314E
}

// ContainsKorean 은 문자열에 한글이 포함되어 있는지 확인합니다.
func ContainsKorean(s string) bool {
	for _, r := range s {
		if (r >= 0xAC00 && r <= 0xD7A3) || IsChosung(r) {
			return true
		}
	}
	return false
}

// IsChosungOnly 는 문자열이 순수 한글 초성(+ 공백)으로만 이루어져 있는지 확인합니다.
func IsChosungOnly(s string) bool {
	hasChosung := false
	for _, r := range s {
		if IsChosung(r) {
			hasChosung = true
		} else if !unicode.IsSpace(r) {
			return false
		}
	}
	return hasChosung
}

// ExtractInitials 는 입력 문자열의 종류에 따라 적절한 이니셜을 추출합니다.
// 한글이 포함된 경우 초성을, 영문만 있는 경우 각 단어의 첫 글자를 반환합니다.
func ExtractInitials(s string) string {
	if ContainsKorean(s) {
		return ExtractChosung(s)
	}
	return ExtractEnglishInitials(s)
}
