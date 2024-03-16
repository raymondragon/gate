package main

import (
    "flag"
    "log"
    "net"
    "net/http"
    "strings"

    "github.com/raymondragon/golib"
)

var (
    rawAURL = flag.String("A", "", "Authorization: http://local:port/secret_path#file")
    rawTURL = flag.String("T", "", "Transmission:   tcp://local:port/remote:port#file")
)

func main() {
    flag.Parse()
    if *rawAURL == "" && *tranURL == "" {
        flag.Usage()
        log.Fatalf("[ERRO] %v", "Invalid Flag(s)")
    }
    defaultFile := "IPlist"
    if *rawAURL != "" {
        parsedAURL, err := golib.URLParse(*rawAURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        if parsedAURL.Fragment == "" {
            parsedAURL.Fragment = defaultFile
        } else {
            defaultFile = parsedAURL.Fragment
        }
        log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*rawAURL, "#")[0], parsedAURL.Fragment)
        go listenAndAuth(parsedAURL)
    }
    if *rawTURL != "" {
        parsedTURL, err := golib.URLParse(*rawTURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        if *rawAURL != "" {
            parsedTURL.Fragment = defaultFile
        }
        log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*rawTURL, "#")[0], parsedTURL.Fragment)
        listenAndCopy(parsedTURL)
    }
    select {}
}

func listenAndAuth(parsedURL golib.ParsedURL) {
    http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
        golib.IPDisplayHandler(w, r)
        golib.IPRecordHandler(parsedURL.Fragment)(w, r)
    })
    switch parsedURL.Scheme {
    case "http":
        if err := golib.ServeHTTP(parsedURL.Hostname, parsedURL.Port, nil); err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
    default:
        log.Fatalf("[ERRO] Invalid Scheme: %v", parsedURL.Scheme)
    }
}

func listenAndConn(parsedURL golib.ParsedURL) {
    localAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(parsedURL.Hostname, parsedURL.Port))
    if err != nil {
        log.Fatalf("[ERRO] %v", err)
    }
    switch parsedURL.Scheme {
    case "tcp":
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
            go golib.HandleConn(localConn, parsedURL.Fragment, strings.TrimPrefix(parsedURL.Path, "/"))
        }
    default:
        log.Fatalf("[ERRO] Invalid Scheme: %v", parsedURL.Scheme)
    }
}