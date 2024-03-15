package main

import (
    "flag"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "strings"

    "github.com/raymondragon/golib"
)

var (
    authURL = flag.String("A", "", "Authorization: http(s)://local:port/secret_path#ipfile")
    tranURL = flag.String("T", "", "Transmission: tcp(udp)://local:port/remote:port#ipfile")
)

func main() {
    flag.Parse()
    switch {
    case *authURL != "" && *tranURL != "":
        aURL, err := urlParse(*authURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        tURL, err := urlParse(*tranURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        if aURL.Fragment == "" {
            aURL.Fragment = "IPlist"
        }
        tURL.Fragment = aURL.Fragment
        log.Printf("[INFO] %v://%v:%v%v <-> [FILE] %v", aURL.Scheme, aURL.Hostname, aURL.Port, aURL.Path, aURL.Fragment)
        go listenAndAuth(aURL)
        log.Printf("[INFO] %v://%v:%v <-> %v", tURL.Scheme, tURL.Hostname, tURL.Port, strings.TrimPrefix(tURL.Path, "/"))
        listenAndCopy(tURL, true)
        select {}
    case *authURL != "" && *tranURL == "":
        aURL, err := urlParse(*authURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        log.Printf("[INFO] %v://%v:%v%v <-> [FILE] %v", aURL.Scheme, aURL.Hostname, aURL.Port, aURL.Path, aURL.Fragment)
        listenAndAuth(aURL)
    case *authURL == "" && *tranURL != "":
        tURL, err := urlParse(*tranURL)
        if err != nil {
            log.Fatalf("[ERRO] %v", err)
        }
        if tURL.Fragment == "" {
            log.Printf("[INFO] %v://%v:%v <-> %v", tURL.Scheme, tURL.Hostname, tURL.Port, strings.TrimPrefix(tURL.Path, "/"))
            listenAndCopy(tURL, false)
        } else {
            log.Printf("[INFO] %v://%v:%v <-> %v [FILE] %v", tURL.Scheme, tURL.Hostname, tURL.Port, strings.TrimPrefix(tURL.Path, "/"), tURL.Fragment)
            listenAndCopy(tURL, true)
        }
    default:
        log.Fatalf("[ERRO] %v", "URL Flag Unprovided")
    }
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
            go func(localConn net.Conn) {
                defer localConn.Close()
                clientIP := localConn.RemoteAddr().(*net.TCPAddr).IP.String()
                if authEnabled && !golib.IsInFile(clientIP, parsedURL.Fragment) {
                    log.Printf("[WARN] %v", clientIP)
                    return
                }
                remoteConn, err := net.Dial("tcp", strings.TrimPrefix(parsedURL.Path, "/"))
                if err != nil {
                    log.Fatalf("[ERRO] %v", err)
                }
                defer remoteConn.Close()
                go io.Copy(remoteConn, localConn)
                io.Copy(localConn, remoteConn)
            }(localConn)
        }
    default:
        log.Fatalf("[ERRO] %v", parsedURL.Scheme)
    }
}