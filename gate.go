package main

import (
    "flag"
    "io"
    "log"
    "net"
    "net/http"
    "strings"

    "github.com/raymondragon/golib"
)

var (
    rawAURL = flag.String("A", "", "Authorization://local:port/secret_path#file")
    rawTURL = flag.String("T", "", "Transmissions://local:port/remote:port#file")
)

func main() {
    flag.Parse()
    if *rawAURL == "" && *rawTURL == "" {
        flag.Usage()
        log.Fatalf("[ERRO] %v", "Invalid Flag(s)")
    }
    defaultFile := ""
    if *rawAURL != "" {
        parsedAURL, err := golib.URLParse(*rawAURL)
        if err != nil {
            log.Printf("[WARN] %v", err)
        }
        if parsedAURL.Fragment == "" {
            parsedAURL.Fragment, defaultFile = "IPlist", "IPlist"
        } else {
            defaultFile = parsedAURL.Fragment
        }
        log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*rawAURL, "#")[0], parsedAURL.Fragment)
        go handleAuthorization(parsedAURL)
    }
    if *rawTURL != "" {
        parsedTURL, err := golib.URLParse(*rawTURL)
        if err != nil {
            log.Printf("[WARN] %v", err)
        }
        if defaultFile != "" {
            parsedTURL.Fragment = defaultFile
        }
        log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*rawTURL, "#")[0], parsedTURL.Fragment)
        handleTransmissions(parsedTURL)
    }
    select {}
}

func handleAuthorization(parsedURL golib.ParsedURL) {
    http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
        golib.IPDisplayHandler(w, r)
        golib.IPRecordHandler(parsedURL.Fragment)(w, r)
    })
    if err := golib.ServeHTTP(parsedURL.Hostname, parsedURL.Port, nil); err != nil {
        log.Fatalf("[ERRO] %v", err)
    }
}

func handleTransmissions(parsedURL golib.ParsedURL) {
    localAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(parsedURL.Hostname, parsedURL.Port))
    if err != nil {
        log.Printf("[WARN] %v", err)
    }
    listener, err := net.ListenTCP("tcp", localAddr)
    if err != nil {
        log.Fatalf("[ERRO] %v", err)
    }
    defer listener.Close()
    for {
        localConn, err := listener.Accept()
        if err != nil {
            log.Printf("[WARN] %v", err)
            continue
        }
        go func(localConn net.Conn) {
            defer localConn.Close()
            clientIP := localConn.RemoteAddr().(*net.TCPAddr).IP.String()
            if parsedURL.Fragment != "" && !golib.IsInFile(clientIP, parsedURL.Fragment) {
                log.Printf("[WARN] %v", clientIP)
                return
            }
            remoteConn, err := net.Dial("tcp", strings.TrimPrefix(parsedURL.Path, "/"))
            if err != nil {
                log.Printf("[WARN] %v", err)
                return
            }
            defer remoteConn.Close()
            go io.Copy(remoteConn, localConn)
            io.Copy(localConn, remoteConn)
        }(localConn)
    }
}