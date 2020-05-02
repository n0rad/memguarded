# memguarded

A simple demo client and server for `github.com/awnumar/memguard`

This can : 
- run `server` to start a unix socket server to store a secret in memguard
- run `set` to send the secret to the server
- run `get` to get the secret from the server

Security support with :
- unix socket file permission set to current user only
- check SO_PEERCRED matches current server user
- socket password
