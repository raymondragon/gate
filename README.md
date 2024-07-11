## Overview
GATE is a Golang-powered Authorization and/or Transmission Entry-point.
This project provides a Go-based server application that handles authorization and transmission functionalities.

## Features

- **Authorization Handling**: Manages HTTP(S) requests for IP display and recording.
- **Transmissions Handling**: Facilitates TCP connections between a local and a remote address with IP-based access control.

## Usage

Run the server with the appropriate flags for authorization and transmission:

```sh
gate -A "Authorization://local:port/secret_path#file" -T "Transmissions://local:port/remote:port#file"
```

- `-A`: Defines the authorization URL.
- `-T`: Defines the transmissions URL.

## Flags

- `-A "http(s)://local:port/secret_path#file"`
  - `local:port`: Local address and port for authorization handling.
  - `secret_path`: Path for handling authorization requests.
  - `file`: (Optional) File used for storing IP records. Defaults to "IPlist".

- `-T "tcp://local:port/remote:port#file"`
  - `local:port`: Local address and port for listening to TCP connections.
  - `remote:port`: Remote address and port to forward connections.
  - `file`: (Optional) File used for checking authorized IPs. Defaults to the file used in the authorization URL.

## Example

```sh
gate -A "https://:8080/auth" -T "tcp://:9000/127.0.0.1:9001"
```

In this example:
- The server handles authorization on `https://ip:8080/auth` and stores IPs in `IPlist`.
- The server listens on `:9000` for TCP connections and forwards them to `127.0.0.1:9001`, checking IPs against `IPlist`.

## Docker or Podman Usage

You can also run this project using Docker or Podman. Below is an example command:

```sh
podman run -d --name=gate-ssh --restart=always --net=host ghcr.io/raymondragon/gate -A=http://:80/secret_string -T=tcp://:22/127.0.0.1:2222
```

## License

This project is licensed under the MIT License. See the LICENSE file for details.


## Contact

For issues and discussion, please open an issue on GitHub.

## Sponsor

This project was developed using the testing environment provided by [Alice Networks](https://alicenetworks.net).
