# gonnect

An implementation of kde-connect api using go and as few dependecies as possible.

## Goal

The goal of this project is to provide a simple lightweight, DE agnostic implementation of kde-connect.
The project right now has many missing features and may never be found.

## Usage


Start the server using

´´´sh
go run ./cmd/server
´´´

and interact with it using the client

´´´
go run ./cmd/cli <command>
´´´

where the cli should tell you about the available commands.

## Features

- [x] Discover
- [x] Discoverable
- [x] Pairing
- [x] Ensuring certs are correct
- [x] Ping pong
- [x] Clipboard sync (using wl-clipboard)
- [ ] File sharing
- [ ] Even fewer dependecies
- [ ] Notifications?
- [ ] Battery?
- [ ] Commands?
- [ ] Remote input?
- [ ] Rest of the kde connect spec?

## License

This project is covered by the license BSD 3-clause, see LICENSE file.
