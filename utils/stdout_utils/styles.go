package stdout_utils

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

const (
	seekModeOpener byte = 0
	seekModeCloser byte = 1
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
//
// <b>This will be printed in bold.</b>
//
// <u>This will be printed underlined./<u>
//
// <b><u>This will be printed both bold and underlined.</u></b>
//
// <yellow>This will be printed in yellow.</yellow>
func ProcessStyle(in string) (string, error) {
	var err error
	for _, sd := range styles {
		in, err = processStyle(in, sd)
		if err != nil {
			return "", err
		}
	}
	return in, nil
}

func PrintfStyled(format string, args ...interface{}) {
	rawString := fmt.Sprintf(format, args...)
	processedString, _ := ProcessStyle(rawString)
	fmt.Print(processedString)
}

// RemoveStyle removes style tags from input string and returns it.
func RemoveStyle(in string) string {
	for _, sd := range styles {
		in = removeStyle(in, sd)
	}
	return in
}

func processStyle(in string, styleData styleData) (string, error) {
	sb := strings.Builder{}
	var builtString string

	opener := styleData.openTag
	closer := styleData.closeTag

	openerSize := len(opener)
	closerSize := len(closer)

	cursor := 0
	lastSplit := 0
	seekMode := seekModeOpener
	for {
		var selection string
		var endRange int
		if seekMode == seekModeOpener {
			endRange = cursor + openerSize
		} else if seekMode == seekModeCloser {
			endRange = cursor + closerSize
		} else {
			return "", fmt.Errorf("unexpected seek mode: %d", seekMode)
		}

		if endRange > len(in) {
			if seekMode == seekModeOpener {
				sb.WriteString(in[lastSplit:])
			} else if seekMode == seekModeCloser {
				sb.WriteString(styleData.style.Sprint(in[lastSplit:]))
			} else {
				return "", fmt.Errorf("unexpected seek mode: %d", seekMode)
			}

			builtString = sb.String()
			builtString = strings.Replace(builtString, styleData.openTag, "", -1)
			builtString = strings.Replace(builtString, styleData.closeTag, "", -1)

			break
		}

		selection = in[cursor:endRange]

		if seekMode == seekModeOpener && selection == opener {
			appendText := in[lastSplit:endRange]
			sb.WriteString(appendText)
			lastSplit = endRange
			seekMode = seekModeCloser
			cursor += openerSize
		} else if seekMode == seekModeCloser && selection == closer {
			appendText := styleData.style.Sprintf("%s", in[lastSplit:endRange])
			sb.WriteString(appendText)
			lastSplit = endRange
			seekMode = seekModeOpener
			cursor += closerSize
		} else {
			cursor++
		}
	}
	return builtString, nil
}

func removeStyle(in string, styleData styleData) string {
	return strings.Replace(strings.Replace(in, styleData.openTag, "", -1), styleData.closeTag, "", -1)
}
