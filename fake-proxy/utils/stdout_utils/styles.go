package stdout_utils

import (
	"github.com/fatih/color"
	"regexp"
	"strings"
)

type styleData struct {
	openTag  string
	closeTag string
	style    *color.Color
}

var styles = []styleData{
	{"<b>", "</b>", color.New(color.Bold)},
	{"<u>", "</u>", color.New(color.Underline)},
	{"<red>", "</red>", color.New(color.FgRed)},
	{"<green>", "</green>", color.New(color.FgGreen)},
	{"<blue>", "</blue>", color.New(color.FgBlue)},
	{"<yellow>", "</yellow>", color.New(color.FgYellow)},
	{"<cyan>", "</cyan>", color.New(color.FgCyan)},
	{"<magenta>", "</magenta>", color.New(color.FgMagenta)},
}

// ProcessStyle formats input string according to html-like tag placements.
// Output string can be sent to stdout which will be displayed as a style applied text.
//
// Examples:
// <b>This will be printed in bold.</b>
// <u>This will be printed underlined./<u>
// <b><u>This will be printed both bold and underlined.</u></b>
// <yellow>This will be printed in yellow.</yellow>
func ProcessStyle(in string) string {
	for _, sd := range styles {
		in = processStyle(in, sd)
	}
	return in
}

// RemoveStyle removes style tags from input string and returns it.
func RemoveStyle(in string) string {
	for _, sd := range styles {
		in = removeStyle(in, sd)
	}
	return in
}

func processStyle(in string, styleData styleData) string {
	regStyle, err := regexp.Compile(styleData.openTag + `.+` + styleData.closeTag)
	if err != nil {
		return in
	}

	replaces := make(map[string]string)
	matches := regStyle.FindAllString(in, -1)
	for _, m := range matches {
		textContent := strings.Replace(strings.Replace(m, styleData.openTag, "", 1), styleData.closeTag, "", 1)
		replaceText := styleData.style.Sprintf("%s", textContent)
		replaces[m] = replaceText
	}

	for k, v := range replaces {
		in = strings.Replace(in, k, v, -1)
	}
	return in
}

func removeStyle(in string, styleData styleData) string {
	return strings.Replace(strings.Replace(in, styleData.openTag, "", -1), styleData.closeTag, "", -1)
}
