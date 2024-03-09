FROM docker.io/golang:alpine as builder
WORKDIR /root
ADD . .
RUN go mod init gate && go mod tidy
RUN env CGO_ENABLED=0 go build -v -ldflags '-w -s'
FROM scratch
WORKDIR /
COPY --from=builder /root/gate .
ENTRYPOINT ["/gate"]