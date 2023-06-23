# middleman


#### DESCRIPTION:

middleman `(B)` works like a proxy server, redirecting requests from client `(A)` to server `(C)` and returning the response back to client `(A)`. Unlike a conventional http proxy server, it detects address to redirect http content via request's URL.

When **middleman** is run on server `(B)` (with default listen port of 8080) calls can be made like:

`POST: http://{address-of-B}:8080/https://{address-of-C}`

instead of:

`POST: https://{address-of-C}`

from your local client application `(A)`.

Headers and body will be redirected to server `(C)` and their response will be returned back to caller `(A)`.

Middleman also pretty prints request/response data to stdout and brands request/response pairs with unique IDs.


![Pretty printed output of request/response on default Ubuntu terminal.](/Readme/Pretty_Print_Example.png)


#### USAGE:

`client (A) -> middleman (B) -> server (C)`  

You are developing & debugging a project on your local machine `(A)` which needs to make API calls to an application on a remote machine `(C)`. You local machine `(A)` has no direct access to remote machine `(C)`, but your deployment environment `(B)` does.
There may be similar circumstances where it is more practical to go this way instead of configuring & using a conventional http proxy.

To run middleman on a custom port use -p argument. Example:

`middleman -p 5000`
