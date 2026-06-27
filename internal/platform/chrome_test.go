package platform

import (
	"reflect"
	"strings"
	"testing"
)

func TestChromeCommandArgsContainOnlyOfficialURLs(t *testing.T) {
	urls := []string{
		"https://chatgpt.com/",
		"https://gemini.google.com/app",
		"https://grok.com/",
	}

	got := chromeCommandArgs(urls)
	if !reflect.DeepEqual(got, urls) {
		t.Fatalf("chromeCommandArgs() = %v, want %v", got, urls)
	}

	forbidden := []string{
		"--headless",
		"--remote-debugging-port",
		"--user-data-dir",
		"--enable-automation",
	}
	joined := strings.Join(got, " ")
	for _, flag := range forbidden {
		if strings.Contains(joined, flag) {
			t.Fatalf("chrome args contain forbidden automation flag %q: %v", flag, got)
		}
	}
}

func TestChromeCommandArgsRejectsNonHTTPSURL(t *testing.T) {
	if got := chromeCommandArgs([]string{"javascript:alert(1)"}); got != nil {
		t.Fatalf("chromeCommandArgs() = %v, want nil", got)
	}
}

func TestChromeInternalPageArgsOnlyAllowsExtensionsPage(t *testing.T) {
	if got := chromeInternalPageArgs("chrome://extensions"); !reflect.DeepEqual(got, []string{"chrome://extensions"}) {
		t.Fatalf("chromeInternalPageArgs() = %v", got)
	}
	if got := chromeInternalPageArgs("chrome://settings/passwords"); got != nil {
		t.Fatalf("chromeInternalPageArgs() allowed unexpected page: %v", got)
	}
}
