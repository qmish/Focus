package search

import (
	"strings"
	"unicode/utf8"
)

// HighlightSnippet возвращает короткий отрывок текста (<=maxLen),
// центрированный вокруг первого вхождения query, с подсветкой совпадения
// тегами <mark>...</mark>. Поиск регистронезависимый, но исходный
// регистр текста сохраняется.
//
// Если query пустой или не найден, возвращает trim'нутый prefix
// исходного текста.
func HighlightSnippet(content, query string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 160
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if query == "" {
		return truncate(content, maxLen)
	}

	lc := strings.ToLower(content)
	lq := strings.ToLower(query)
	idx := strings.Index(lc, lq)
	if idx < 0 {
		return truncate(content, maxLen)
	}

	// Окно вокруг совпадения: ~ треть до, остальное после.
	half := maxLen / 3
	startByte := idx - half
	if startByte < 0 {
		startByte = 0
	}
	startByte = alignToRune(content, startByte, false)
	endByte := startByte + maxLen
	if endByte > len(content) {
		endByte = len(content)
	}
	endByte = alignToRune(content, endByte, true)

	prefix := ""
	suffix := ""
	if startByte > 0 {
		prefix = "..."
	}
	if endByte < len(content) {
		suffix = "..."
	}

	// Соберём snippet с <mark> на исходном (case-preserving) тексте.
	snippet := content[startByte:endByte]
	relIdx := idx - startByte
	relEnd := relIdx + len(query)
	if relIdx < 0 || relEnd > len(snippet) {
		return prefix + escapeHTML(snippet) + suffix
	}
	return prefix +
		escapeHTML(snippet[:relIdx]) +
		"<mark>" + escapeHTML(snippet[relIdx:relEnd]) + "</mark>" +
		escapeHTML(snippet[relEnd:]) +
		suffix
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if max > len(r) {
		return s
	}
	return string(r[:max]) + "..."
}

func alignToRune(s string, byteIdx int, forward bool) int {
	if byteIdx <= 0 {
		return 0
	}
	if byteIdx >= len(s) {
		return len(s)
	}
	if forward {
		for byteIdx < len(s) && (s[byteIdx]&0xC0) == 0x80 {
			byteIdx++
		}
	} else {
		for byteIdx > 0 && (s[byteIdx]&0xC0) == 0x80 {
			byteIdx--
		}
	}
	return byteIdx
}

func escapeHTML(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			sb.WriteString("&amp;")
		case '<':
			sb.WriteString("&lt;")
		case '>':
			sb.WriteString("&gt;")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
