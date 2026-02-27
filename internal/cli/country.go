package cli

import (
	"regexp"
	"strings"
)

var slovakiaRe = regexp.MustCompile(`(?i)^(sk|slovensko|slovakia|slovensk[aá]\s+republika)$`)

func isSlovakia(country string) bool {
	return slovakiaRe.MatchString(strings.TrimSpace(country))
}
