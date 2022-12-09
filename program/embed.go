package program

import (
	"fmt"

	"github.com/onuryurdupak/gomod/stdout"
)

const (
	stamp_build_date  = "${build_date}"
	stamp_commit_hash = "${commit_hash}"
	stamp_source      = "${source}"

	ErrSuccess  = 0
	ErrInput    = 1
	ErrInternal = 2
	ErrUnknown  = 3

	helpPrompt = `Run 'middleman -h' for help.`

	helpMessage = `
If your terminal does not render styles properly, run 'middleman -hr' to view help in raw mode.

<b><u><yellow>PARAMETERS:</yellow></u></b>
<b><yellow>-v</yellow></b>: Show version info.
<b><yellow>-p</yellow></b>: Run on custom port.
<b><yellow>--raw</b></yellow>: Print request & response bodies without styles.

Readme is availabile at: https://github.com/onuryurdupak/middleman#readme
`
)

func versionInfo() string {
	return fmt.Sprintf(`Build Date: %s | Commit: %s
Source: %s`, stamp_build_date, stamp_commit_hash, stamp_source)
}

func helpMessageStyled() string {
	msg, _ := stdout.ProcessStyle(helpMessage)
	return msg
}

func helpMessageUnstyled() string {
	return stdout.RemoveStyle(helpMessage)
}
