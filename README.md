# fake-proxy


#### DESCRIPTION:

fake-proxy `(B)` acts as an intermediary which forwards requests from client `(A)` to server `(C)` and returns the server's response to client `(A)`. 

When **fake-proxy** is run on your deployment environment `(B)` (and allowed to host on port 8080) you can make calls like:

`POST: http://{address-of-B}:8080/https://{address-of-C}`

instead of:

`POST: https://{address-of-C}`

from your local client application.

It will redirect request headers and request body.

#### USAGE:

`client (A) -> fake-proxy (B) -> server (C)`  

You are developing & debugging a project on your local machine `(A)` which needs to make API calls to an application on a remote machine `(C)`. You local machine `(A)` has no direct access to remote machine `(C)`, but your deployment environment `(B)` does.
There may be similar circumstances where it is more practical to go this way instead of configuring & using a real http proxy.

To run fake-proxy on a custom port use -p argument. Example:

`fake-proxy -p 5000`
