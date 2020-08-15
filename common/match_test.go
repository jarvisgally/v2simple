package common

import (
	"bufio"
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	mather := NewMather("../whitelist")

	expectedMatches := `
127.0.0.1
39.156.69.79
alibabacloud.com.hk
www.jd.com
`
	expectedNotMatches := `
google.com
facebook.com
youtube.com
youtubei.googleapis.com
46.82.174.69
`
	t.Run("Check", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader(expectedMatches))
		for scanner.Scan() {
			l := strings.TrimSpace(scanner.Text())
			if len(l) > 0 && !mather.Check(l) {
				t.Errorf("%v should match", l)
			}
		}
		scanner = bufio.NewScanner(strings.NewReader(expectedNotMatches))
		for scanner.Scan() {
			l := strings.TrimSpace(scanner.Text())
			if len(l) > 0 && mather.Check(l) {
				t.Errorf("%v should not matched", l)
			}
		}
	})
}
