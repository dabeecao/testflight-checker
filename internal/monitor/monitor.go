package monitor

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

const (
	TestFlightURL = "https://testflight.apple.com/join/%s"
	TitleRegex    = `Join the (.+) beta - TestFlight - Apple`
	XPathTitle    = "/html/head/title/text()"
	XPathStatus   = "//*[@class=\"beta-status\"]/span/text()"
)

var FullTexts = []string{
	"This beta is full.",
	"This beta isn't accepting any new testers right now.",
}

var (
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
	}
	UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
)

type AppInfo struct {
	Title     string
	FreeSlots bool
}

func GetAppInfo(tfID string) (AppInfo, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(TestFlightURL, tfID), nil)
	if err != nil {
		return AppInfo{Title: "Ứng dụng chưa rõ", FreeSlots: false}, err
	}
	req.Header.Set("Accept-Language", "en-us")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return AppInfo{Title: "Ứng dụng chưa rõ", FreeSlots: false}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AppInfo{Title: "Ứng dụng chưa rõ", FreeSlots: false}, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		return AppInfo{Title: "Ứng dụng chưa rõ", FreeSlots: false}, err
	}

	titleNode := htmlquery.FindOne(doc, XPathTitle)
	title := "Ứng dụng chưa rõ"
	if titleNode != nil {
		rawTitle := htmlquery.InnerText(titleNode)
		re := regexp.MustCompile(TitleRegex)
		match := re.FindStringSubmatch(rawTitle)
		if len(match) > 1 {
			title = match[1]
		}
	}

	statusNode := htmlquery.FindOne(doc, XPathStatus)
	freeSlots := false
	if statusNode != nil {
		statusText := strings.TrimSpace(htmlquery.InnerText(statusNode))
		isFull := false
		for _, fullText := range FullTexts {
			if statusText == fullText {
				isFull = true
				break
			}
		}
		freeSlots = !isFull
	}

	return AppInfo{Title: title, FreeSlots: freeSlots}, nil
}
