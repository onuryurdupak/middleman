package embed

const (
	Stamp_build_date  = "${build_date}"
	Stamp_commit_hash = "${commit_hash}"

	ErrSuccess  = 0
	ErrInput    = 1
	ErrInternal = 2
	ErrUnknown  = 3

	HelpPrompt = `Run 'fake-proxy -h' for help.`

	HelpMessage = `
If your terminal does not render styles properly, run 'interpolator -hr' to view in style-free mode.

<b><u><yellow>DESCRIPTION</yellow></u></b>
fake-proxy (B) acts as an intermediary which forwards requests from client (A) to server (C) and returns the server's response to client (A).
When <b>fake-proxy</b> is run on your deployment environment (B) (and allowed to host on port 8080) you can make calls like:

<green>POST: http: //{address-of-B}:8080/https://{address-of-C}</green>

instead of:

<green>POST: https: //{address-of-C}</green>

from your local client application. It will redirect request headers and request body.

<b><u><yellow>USAGE</yellow></u></b>
<b>client (A) -> fake-proxy (B) -> server (C)</b>

You are developing & debugging a project on your local machine (A) which needs to make API calls to an application on a remote machine (C).You local machine (A) has no direct access to remote machine (C), but your deployment environment (B) does.
There may be similar circumstances where it is more practical to go this way instead of configuring & using a real http proxy.

To run fake-proxy on a custom port use -p argument. Example:

<green>fake-proxy -p 5000</green>
`
)
