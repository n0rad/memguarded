# memguarded

A client and server for `github.com/awnumar/memguard`

The **memguarded** binary can : 
- run `server` to start a unix socket server to store a secret in memguard
- run `set` to send the secret to the server
- run `get` to get the secret from the server


The client and server code can be used **as libraries** to be integrated in other programs

Security support with :
- unix socket file permission set to current user only
- Check SO_PEERCRED matches current server user
- Socket password

_The code is designed to be sure the password (and the socket password) do not live in memory elsewhere than in memguard_
