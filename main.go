package main

import (
    "bufio"
    "flag"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "strings"
)

var (
    fowdAdd = flag.String("F", "", "Forward")
    lisnAdd = flag.String("L", ":10101", "Listen")
    authAdd = flag.String("auth", ":8080", "Auth")
    pathStr = flag.String("path", "/auth", "Path")
)

func main() {
    flag.Parse()
    _, portAuth, err := net.SplitHostPort(*authAdd)
    if err != nil {
        log.Fatalf("[ERR-00] %v", err)
    }
    _, portLisn, err := net.SplitHostPort(*lisnAdd)
    if err != nil {
        log.Fatalf("[ERR-01] %v", err)
    }
    if portAuth != portLisn {
        log.Printf("[LISTEN] %v%v", *authAdd, *pathStr)
        go ListenAndAuth()
    } else {
        log.Fatal("[ERR-02] Server Port Conflict")
    }
    if *fowdAdd != "" {
        log.Printf("[LISTEN] %v <-> %v", *lisnAdd, *fowdAdd)
        ListenAndCopy()
    } else {
        log.Println("[WAR-00] No Forward Service")
    }
    select {}
}

func ListenAndAuth() {
    http.HandleFunc(*pathStr, func(w http.ResponseWriter, r *http.Request) {
        clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Printf("[WAR-10] %v", err)
            http.Error(w, "[WAR-10]", 500)
            return
        }
        if _, err := w.Write([]byte(clientIP + "\n")); err != nil {
            log.Printf("[WAR-11] %v", err)
            http.Error(w, "[WAR-11]", 500)
            return
        }
        file, err := os.OpenFile("IPlist", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            log.Printf("[WAR-12] %v", err)
            return
        }
        defer file.Close()
        if inIPlist(clientIP, "IPlist") {
            log.Printf("[WAR-13] %v", clientIP)
            return
        }
        if _, err := file.WriteString(clientIP + "\n"); err != nil {
            log.Printf("[WAR-14] %v", err)
            return
        }
    })
    if err := http.ListenAndServe(*authAdd, nil); err != nil {
        log.Fatalf("[ERR-10] %v", err)
    }
}

func ListenAndCopy() {
    listener, err := net.Listen("tcp", *lisnAdd)
    if err != nil {
        log.Printf("[WAR-20] %v", err)
        return
    }
    defer listener.Close()
    for {
        outConn, err := listener.Accept()
        if err != nil {
            log.Printf("[WAR-21] %v", err)
            continue
        }
        go func(outConn net.Conn) {
            defer outConn.Close()
            clientIP := outConn.RemoteAddr().(*net.TCPAddr).IP.String()
            if !inIPlist(clientIP, "IPlist") {
                log.Printf("[WAR-22] %v", clientIP)
                return
            }
            inConn, err := net.Dial("tcp", *fowdAdd)
            if err != nil {
                log.Printf("[WAR-23] %v", err)
                return
            }
            defer inConn.Close()
            go io.Copy(inConn, outConn)
            io.Copy(outConn, inConn)
        }(outConn)
    }
}

func inIPlist(ip string, list string) bool {
    file, err := os.Open(list)
    if err != nil {
        if os.IsNotExist(err) {
            file, err := os.Create(list)
            if err != nil {
                return false
            }
            defer file.Close()
            return false
        }
        return false
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        if strings.TrimSpace(scanner.Text()) == ip {
            return true
        }
    }
    return false
}