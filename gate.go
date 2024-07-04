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
    rawAURL = flag.String("A", "", "Authorization://host:port/secret_path#file")
    rawTURL = flag.String("T", "", "Transmissions://host:port/target:port#file")
    semTEMP = make(chan struct{}, 1024)
)

func main() {
    flag.Parse()
    if *rawAURL == "" && *rawTURL == "" {
        flag.Usage()
        log.Fatalf("[ERRO] Invalid Flags")
    }
    defaultFile := ""
    if *rawAURL != "" {
        parsedAURL, err := golib.URLParse(*rawAURL)
        if err != nil {
            log.Printf("[WARN] URL Parsing: %v", err)
        }
        if parsedAURL.Fragment == "" {
            parsedAURL.Fragment, defaultFile = "IPlist", "IPlist"
        } else {
            defaultFile = parsedAURL.Fragment
        }
        if parsedAURL.Scheme != "auto" {
            log.Printf("[INFO] %v <-> [FILE] %v", strings.Split(*rawAURL, "#")[0], parsedAURL.Fragment)
        } else {
            log.Printf("[INFO] %v", strings.Split(*rawAURL, "#")[0])
        }
        go handleAuthorization(parsedAURL)
    }
    if *rawTURL != "" {
        parsedTURL, err := golib.URLParse(*rawTURL)
        if err != nil {
            log.Printf("[WARN] URL Parsing: %v", err)
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
    switch parsedURL.Scheme {
    case "http":
        http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
            golib.IPDisplayHandler(w, r)
            golib.IPRecordHandler(parsedURL.Fragment)(w, r)
        })
        if err := golib.ServeHTTP(parsedURL.Hostname, parsedURL.Port, nil); err != nil {
            log.Fatalf("[ERRO] HTTP Service: %v", err)
        }
    case "https":
        http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
            golib.IPDisplayHandler(w, r)
            golib.IPRecordHandler(parsedURL.Fragment)(w, r)
        })
        tlsConfig, err := golib.TLSConfigApplication(parsedURL.Username, parsedURL.Hostname)
        if err != nil {
            log.Printf("[WARN] Cert Application: %v", err)
            tlsConfig, err = golib.TLSConfigGeneration(parsedURL.Hostname)
            if err != nil {
                log.Printf("[WARN] Cert Generation: %v", err)
            }
        }
        if err := golib.ServeHTTPS(parsedURL.Hostname, parsedURL.Port, nil, tlsConfig); err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
    case "auto":
        autoHandler := golib.ProxyHandler(parsedURL.Hostname, parsedURL.Username, parsedURL.Password, nil)
        tlsConfig, err := golib.TLSConfigApplication(parsedURL.Username, parsedURL.Hostname)
        if err != nil {
            log.Printf("[WARN] Cert Application: %v", err)
            tlsConfig, err = golib.TLSConfigGeneration(parsedURL.Hostname)
            if err != nil {
                log.Printf("[WARN] Cert Generation: %v", err)
            }
        }
        if err := golib.ServeHTTPS(parsedURL.Hostname, parsedURL.Port, autoHandler, tlsConfig); err != nil {
            log.Fatalf("[ERRO] HTTPS Service: %v", err)
        }
    default:
        log.Fatalf("[ERRO] Invalid Scheme: %v", parsedURL.Scheme)
    }
}

func handleTransmissions(parsedURL golib.ParsedURL) {
    localAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(parsedURL.Hostname, parsedURL.Port))
    if err != nil {
        log.Printf("[WARN] Addr Resolving: %v", err)
    }
    listener, err := net.ListenTCP("tcp", localAddr)
    if err != nil {
        log.Fatalf("[ERRO] TCP Listening: %v", err)
    }
    defer listener.Close()
    for {
        localConn, err := listener.Accept()
        if err != nil {
            log.Printf("[WARN] Listener Denied: %v", err)
            continue
        }
        semTEMP <- struct{}{}
        go func(localConn net.Conn) {
            defer localConn.Close()
            defer func() { <-semTEMP }()
            clientIP := localConn.RemoteAddr().(*net.TCPAddr).IP.String()
            if parsedURL.Fragment != "" && !golib.IsInFile(clientIP, parsedURL.Fragment) {
                log.Printf("[WARN] Access Denied: %v", clientIP)
                return
            }
            remoteConn, err := net.Dial("tcp", strings.TrimPrefix(parsedURL.Path, "/"))
            if err != nil {
                log.Printf("[WARN] TCP Dialing: %v", err)
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
