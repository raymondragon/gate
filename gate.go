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
    authURL = flag.String("A", "", "Authorization: http(s)://local:port/secret_path#ipfile")
    tranURL = flag.String("T", "", "Transmission: tcp(udp)://local:port/remote:port#ipfile")
)

func main() {
    flag.Parse()
    if *authURL == "" && *tranURL == "" {
        log.Fatalf("[ERRO] %v", "URL Flag Unprovided")
    }
    if *authURL != "" {
        aURL, err := golib.URLParse(*authURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        log.Printf("[INFO] %v://%v:%v%v <-> [FILE] %v", aURL.Scheme, aURL.Hostname, aURL.Port, aURL.Path, aURL.Fragment)
        go listenAndAuth(aURL)
    }
    if *tranURL != "" {
        tURL, err := golib.URLParse(*tranURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        log.Printf("[INFO] %v://%v:%v <-> %v", tURL.Scheme, tURL.Hostname, tURL.Port, strings.TrimPrefix(tURL.Path, "/"))
        authEnabled := tURL.Fragment != ""
        listenAndCopy(tURL, authEnabled)
    }
    select {}
}

func listenAndAuth(parsedURL ParsedURL) {
    http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
        golib.IPDisplayHandler(w, r)
        golib.IPRecordHandler(parsedURL.Fragment)(w, r)
    })
    switch parsedURL.Scheme {
    case "http":
        log.Printf("[INFO] %v", *authURL)
        if err := golib.ServeHTTP(parsedURL.Hostname, parsedURL.Port, nil); err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
    case "https":
        tlsConfig, err := golib.TLSConfigApplication(parsedURL.Hostname)
        if err != nil {
            log.Printf("[WARN] %v", err)
            tlsConfig, err = golib.TLSConfigGeneration(parsedURL.Hostname)
            if err != nil {
                log.Printf("[WARN] %v", err)
            }
        }
        log.Printf("[INFO] %v", *authURL)
        if err := golib.ServeHTTPS(parsedURL.Hostname, parsedURL.Port, nil, tlsConfig); err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
    default:
        log.Fatalf("[ERRO] %v", parsedURL.Scheme)
    }
}

func listenAndCopy(parsedURL ParsedURL, authEnabled bool) {
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
            go golib.HandleConn(localConn, authEnabled, parsedURL.Fragment, strings.TrimPrefix(parsedURL.Path, "/"))
        }
    default:
        log.Fatalf("[ERRO] %v", parsedURL.Scheme)
    }
}