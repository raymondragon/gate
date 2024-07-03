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
    rawAURL = flag.String("A", "", "Authorization://user:pass@host:port/path#file")
    rawTURL = flag.String("T", "", "Transmission://relay:port/target:port#file")
    semTEMP = make(chan struct{}, 1024)
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
            log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*rawTURL, "#")[0], parsedTURL.Fragment)
        } else {
            log.Printf("[INFO] %v", strings.Split(*rawTURL, "#")[0])
        }
        handleTransmissions(parsedTURL)
    }
    select {}
}

func handleAuthorization(parsedURL golib.ParsedURL) {
    http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
        golib.IPDisplayHandler(w, r)
        golib.IPRecordHandler(parsedURL.Fragment)(w, r)
    })
    authHandler := golib.ProxyHandler(parsedURL.Hostname, parsedURL.Username, parsedURL.Password, nil)
    switch parsedURL.Scheme {
    case "http":
        if err := golib.ServeHTTP(parsedURL.Hostname, parsedURL.Port, authHandler); err != nil {
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
        if err := golib.ServeHTTPS(parsedURL.Hostname, parsedURL.Port, authHandler, tlsConfig); err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
    default:
        log.Fatalf("[ERRO] Invalid Scheme: %v", parsedURL.Scheme)
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
        semTEMP <- struct{}{}
        go func(localConn net.Conn) {
            defer localConn.Close()
            defer func() { <-semTEMP }()
            clientIP := localConn.RemoteAddr().(*net.TCPAddr).IP.String()
            if parsedURL.Fragment != "" && !golib.IsInFile(clientIP, parsedURL.Fragment) {
                log.Printf("[WARN] %v not in allowed IP list", clientIP)
                return
            }
            remoteConn, err := net.Dial("tcp", strings.TrimPrefix(parsedURL.Path, "/"))
            if err != nil {
                log.Printf("[WARN] %v", err)
                return
            }
            defer remoteConn.Close()
            done := make(chan struct{})
            go func() {
                io.Copy(remoteConn, localConn)
                done <- struct{}{}
            }()
            go func() {
                io.Copy(localConn, remoteConn)
                done <- struct{}{}
            }()
            <-done
        }(localConn)
    }
}
