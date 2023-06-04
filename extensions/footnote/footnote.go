package footnote

import (
	"fmt"
	"regexp"

	"github.com/emad-elsaid/xlog"
)

func init() {
	xlog.RegisterPreprocessor(extractFootnotes)
}

var refReg = regexp.MustCompile(`(?iU)\s*\[\?\]\((?P<url>.+)\)`)

func extractFootnotes(in xlog.Markdown) xlog.Markdown {
	linkSet := map[string]int{}
	counter := 0
	urlIndex := refReg.SubexpIndex("url")

	out := refReg.ReplaceAllStringFunc(string(in), func(found string) string {
		matches := refReg.FindStringSubmatch(found)
		url := matches[urlIndex]

		index, ok := linkSet[url]
		if !ok {
			counter++
			index = counter
			linkSet[url] = counter
		}

		return fmt.Sprintf("<sup>[[%d]](#ref:%d)</sup>", index, index)
	})

	if len(linkSet) == 0 {
		return xlog.Markdown(out)
	}

	out += "---"
	out += "\n<ol>"
	for url, index := range linkSet {
		out += fmt.Sprintf("\n<li><a id=\"ref:%d\" href=\"%s\">%s</a></li>", index, url, url)
	}
	out += "</ol>"

	return xlog.Markdown(out)
}
