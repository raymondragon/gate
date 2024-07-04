FROM golang:alpine as builder
WORKDIR /root
ADD . .
RUN go mod init gate && go mod tidy
RUN env CGO_ENABLED=0 go build -v -ldflags '-w -s'
FROM alpine
WORKDIR /
COPY --from=builder /root/gate .
ENTRYPOINT ["/gate"]
