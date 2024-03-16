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
    authURL = flag.String("A", "", "Authorization: http://local:port/secret_path#file")
    tranURL = flag.String("T", "", "Transmission:   tcp://local:port/remote:port#file")
)

func main() {
    flag.Parse()
    if *authURL == "" && *tranURL == "" {
        flag.Usage()
        log.Fatal("[ERRO] Invalid Flag(s)")
    }
    defaultFile := "IPlist"
    if *authURL != "" {
        parsedAURL, err := golib.URLParse(*authURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        if parsedAURL.Fragment == "" {
            parsedAURL.Fragment = defaultFile
        } else {
            defaultFile = parsedAURL.Fragment
        }
        log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*authURL, "#")[0], parsedAURL.Fragment)
        go listenAndAuth(parsedAURL)
    }
    if *tranURL != "" {
        parsedTURL, err := golib.URLParse(*tranURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        if *authURL != "" {
            parsedTURL.Fragment = defaultFile
        }
        log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*tranURL, "#")[0], parsedTURL.Fragment)
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

func listenAndCopy(parsedURL golib.ParsedURL) {
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
            go golib.HandleConn(localConn, parsedURL.Fragment != "", parsedURL.Fragment, strings.TrimPrefix(parsedURL.Path, "/"))
        }
    default:
        log.Fatalf("[ERRO] Invalid Scheme: %v", parsedURL.Scheme)
    }
}