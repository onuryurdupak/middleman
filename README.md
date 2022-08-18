# middleman


#### DESCRIPTION:

middleman `(B)` acts as an intermediary which forwards requests from client `(A)` to server `(C)` and returns the response back to client `(A)`. 

When **middleman** is run on server `(B)` (and say it's allowed to listen on port 8080) you can make calls like:

`POST: http://{address-of-B}:8080/https://{address-of-C}`

instead of:

`POST: https://{address-of-C}`

from your local client application `(A)`.

Headers and body will be redirected to server `(C)` and their response will be redirected back to original client `(A)`.

Middleman also pretty prints request/response data to stdout and brands request/response pairs with unique IDs.


![Pretty printed output of request/response on default Ubuntu termianl.](/Readme/Pretty_Print_Example.png)


#### USAGE:

`client (A) -> middleman (B) -> server (C)`  

You are developing & debugging a project on your local machine `(A)` which needs to make API calls to an application on a remote machine `(C)`. You local machine `(A)` has no direct access to remote machine `(C)`, but your deployment environment `(B)` does.
There may be similar circumstances where it is more practical to go this way instead of configuring & using a http proxy.

To run middleman on a custom port use -p argument. Example:

`middleman -p 5000`
