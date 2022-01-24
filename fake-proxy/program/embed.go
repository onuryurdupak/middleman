package program

import (
	"fake-proxy/utils/stdout_utils"
	"fmt"
)

const (
	stamp_build_date  = "${build_date}"
	stamp_commit_hash = "${commit_hash}"
	stamp_source      = "${source}"

	ErrSuccess  = 0
	ErrInput    = 1
	ErrInternal = 2
	ErrUnknown  = 3

	helpPrompt = `Run 'fake-proxy -h' for help.`

	helpMessage = `
If your terminal does not render styles properly, run 'interpolator -hr' to view in style-free mode.

<b><u><yellow>PARAMETERS:</yellow></u></b>
<b><yellow>-v</yellow></b>: Show version info.
<b><yellow>-p</yellow></b>: Run on custom port.

<b><u><yellow>DESCRIPTION</yellow></u></b>
fake-proxy <b><yellow>(B)</yellow></b> acts as an intermediary which forwards requests from client <b><yellow>(A)</yellow></b> to server <b><yellow>(C)</yellow></b> and returns the server's response to client <b><yellow>(A)</yellow></b>.
When <b>fake-proxy</b> is run on your deployment environment <b><yellow>(B)</yellow></b> (and allowed to host on port 8080) you can make calls like:

<green>POST: http: //{address-of-B}:8080/https://{address-of-C}</green>

instead of:

<green>POST: https: //{address-of-C}</green>

from your local client application. It will redirect request headers and request body.

<b><u><yellow>USAGE</yellow></u></b>
<b>client <b><yellow>(A)</yellow></b> -> fake-proxy <b><yellow>(B)</yellow></b> -> server <b><yellow>(C)</yellow></b></b>

You are developing & debugging a project on your local machine <b><yellow>(A)</yellow></b> which needs to make API calls to an application on a remote machine <b><yellow>(C)</yellow></b>.You local machine <b><yellow>(A)</yellow></b> has no direct access to remote machine <b><yellow>(C)</yellow></b>, but your deployment environment <b><yellow>(B)</yellow></b> does.
There may be similar circumstances where it is more practical to go this way instead of configuring & using a real http proxy.

To run fake-proxy on a custom port use -p argument. Example:

<green>fake-proxy -p 5000</green>
`
)

func versionInfo() string {
	return fmt.Sprintf(`Build Date: %s | Commit: %s
Source: %s`, stamp_build_date, stamp_commit_hash, stamp_source)
}

func helpMessageStyled() string {
	msg, _ := stdout_utils.ProcessStyle(helpMessage)
	return msg
}

func helpMessageUnstyled() string {
	return stdout_utils.RemoveStyle(helpMessage)
}
