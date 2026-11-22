package program

import (
	"fmt"

	"github.com/onuryurdupak/gomod/v2/stdout"
)

const (
	stamp_build_date    = "${build_date}"
	stamp_build_version = "${build_version}"
	stamp_source        = "${source}"

	CodeSuccess     = 0
	CodeInputErr    = 1
	CodeInternalErr = 2
	CodeUnknownErr  = 3

	helpPrompt = `Run 'middleman -h' for help.`

	helpMessage = `
If your terminal does not render styles properly, run 'middleman -hr' to view help in raw mode.

<b><u><yellow>PARAMETERS:</yellow></u></b>
<b><yellow>-v</yellow></b>: Show version info.
<b><yellow>-h</yellow></b>: Show help.
<b><yellow>-hr</yellow></b>: Show help without styles.
<b><yellow>-p <port></yellow></b>: Run on custom port.
<b><yellow>--raw</yellow></b>: Print request & response bodies as they are, without pretty-printing.

Readme is available at: https://github.com/onuryurdupak/middleman#readme
`
)

func versionInfo() string {
	return fmt.Sprintf(`Build Date: %s | Version: %s
Source: %s`, stamp_build_date, stamp_build_version, stamp_source)
}

func helpMessageStyled() string {
	msg, _ := stdout.ProcessStyle(helpMessage)
	return msg
}

func helpMessageUnstyled() string {
	return stdout.RemoveStyle(helpMessage)
}
