## Overview
GATE is a Golang-powered Authorization and/or Transmission Entry-point.
This project provides a Go-based server application that handles authorization and transmission functionalities. It leverages the `golib` library for URL parsing, HTTP handling, and IP management.

## Features

- **Authorization Handling**: Manages HTTP requests for IP display and recording.
- **Transmission Handling**: Facilitates TCP connections between a local and a remote address with IP-based access control.

## Requirements

- Go 1.13 or later
- `golib` library

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/yourusername/yourproject.git
    cd yourproject
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

## Usage

Run the server with the appropriate flags for authorization and transmission:

```sh
go run main.go -A "Authorization://local:port/secret_path#file" -T "Transmissions://local:port/remote:port#file"
```

- `-A`: Defines the authorization URL.
- `-T`: Defines the transmission URL.

## Flags

- `-A "Authorization://local:port/secret_path#file"`
  - `local:port`: Local address and port for authorization handling.
  - `secret_path`: Path for handling authorization requests.
  - `file`: (Optional) File used for storing IP records. Defaults to "IPlist".

- `-T "Transmissions://local:port/remote:port#file"`
  - `local:port`: Local address and port for listening to TCP connections.
  - `remote:port`: Remote address and port to forward connections.
  - `file`: (Optional) File used for checking authorized IPs. Defaults to the file used in the authorization URL.

## Example

```sh
go run main.go -A "Authorization://localhost:8080/auth#IPlist" -T "Transmissions://localhost:9000/remotehost:9001#IPlist"
```

In this example:
- The server handles authorization on `localhost:8080/auth` and stores IPs in `IPlist`.
- The server listens on `localhost:9000` for TCP connections and forwards them to `remotehost:9001`, checking IPs against `IPlist`.

## Code Structure

- **main.go**: Entry point of the application.
  - Parses flags.
  - Starts authorization and transmission handlers.
  - Handles incoming connections based on IP access control.

## Functions

### `main()`

- Parses command-line flags.
- Initializes authorization and transmission handlers.
- Keeps the application running.

### `handleAuthorization(parsedURL golib.ParsedURL)`

- Sets up HTTP handlers for displaying and recording IPs.
- Starts an HTTP server based on the provided URL.

### `handleTransmissions(parsedURL golib.ParsedURL)`

- Sets up a TCP listener.
- Forwards connections between local and remote addresses.
- Checks IPs against a specified file for access control.

## Dependencies

- `golib`: Custom library for URL parsing, HTTP handling, and IP management.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Contributing

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-foo`).
3. Commit your changes (`git commit -am 'Add feature foo'`).
4. Push to the branch (`git push origin feature-foo`).
5. Create a new Pull Request.

## Contact

For issues and discussion, please open an issue on GitHub.

## Sponsor

This project was developed using the testing environment provided by [Alice Networks](https://alicenetworks.net).
