# memguarded

A client/server application and library to keep a secret ... secret as much as possible

Security support with:
- Storing the password on `github.com/awnumar/memguard`
- Unix socket file permission set to current user only
- Check SO_PEERCRED matches current server user (even "root" cannot connect to the socket)
- Client/Server cert check
- Socket password


The **memguarded** binary can : 
- run `server` to start a unix socket server to store a secret in memguard
- run `set` to send the secret to the server
- run `get` to get the secret from the server


_The code is designed to be sure the password (and the socket password) do not live in memory elsewhere than in memguard_
