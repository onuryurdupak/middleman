# fake-proxy

An intermediary which forwards requests from client to server and returns the server's response to client. 

#### What does it do?

`client (A) -> redirector (B) -> server (C)`  

You are developing & debugging a project on your local machine `(A)` which needs to make API calls to an application on a remote machine `(C)`. You local machine `(A)` has no direct access to remote machine `(C)`, but your deployment environment `(B)` does.



When redirector is placed on your deployment environment `(B)` and allowed to serve port 8080 you can make calls like:

`http://{uri-B}:8080/https://target-api`

instead of:

`https://target-api`

There may be circumstances where it is more practical to go this way instead of configuring & using a real http proxy.

