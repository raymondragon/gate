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
    addr = flag.String("a", ":90", "addr")
    ibnd = flag.String("i", "", "inbound")
    obnd = flag.String("o", ":10", "outbound")
    path = flag.String("p", "/ip", "path")
)
func main() {
    flag.Parse()
    _, portAddr, err := net.SplitHostPort(*addr)
    if err != nil {
        log.Fatalf("[ERR-00] %v", err)
    }
    _, portObnd, err := net.SplitHostPort(*obnd)
    if err != nil {
        log.Fatalf("[ERR-01] %v", err)
    }
    if portAddr != portObnd {
        log.Printf("[LISTEN] %v%v", *addr, *path)
        go ListenAndAuth()
    } else {
        log.Fatal("[ERR-02] Server Port Conflict")
    }
    if *ibnd != "" {
        log.Printf("[LISTEN] %v <-> %v", *obnd, *ibnd)
        go ListenAndCopy()
    } else {
        log.Println("[WAR-00] No Inbound Service")
    }
    select {}
}
func ListenAndAuth() {
    http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
        clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Printf("[WAR-10] %v", err)
            http.Error(w, "[WAR-10]", 500)
            return
        }
        if _, err := w.Write([]byte(clientIP+"\n")); err != nil {
            log.Printf("[WAR-11] %v", err)
            http.Error(w, "[WAR-11]", 500)
            return
        }
        if inIPlist(clientIP, "IPlist") {
            log.Printf("[WAR-13] %v", clientIP)
            return
        }
        file, err := os.OpenFile("IPlist", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            log.Printf("[WAR-12] %v", err)
            return
        }
        defer file.Close()
        if _, err := file.WriteString(clientIP+"\n"); err != nil {
            log.Printf("[WAR-14] %v", err)
            return
        }
    })
    if err := http.ListenAndServe(*addr, nil); err != nil {
        log.Fatalf("[ERR-10] %v", err)
    }
}
func ListenAndCopy() {
    listener, err := net.Listen("tcp", *obnd)
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
        go handleOut(outConn)
    }
}
func handleOut(outConn net.Conn) {
    defer outConn.Close()
    clientIP := outConn.RemoteAddr().(*net.TCPAddr).IP.String()
    if !inIPlist(clientIP, "IPlist") {
        log.Printf("[WAR-30] %v", clientIP)
        return
    }
    inConn, err := net.Dial("tcp", *ibnd)
    if err != nil {
        log.Printf("[WAR-31] %v", err)
        return
    }
    defer inConn.Close()
    go io.Copy(inConn, outConn)
    io.Copy(outConn, inConn)
}
func inIPlist(ip string, iplist string) bool {
    file, err := os.OpenFile(iplist, os.O_CREATE|os.O_RDONLY, 0644)
    if err != nil {
        log.Printf("[WAR-40] %v", err)
        return false
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        if strings.TrimSpace(scanner.Text()) == ip {
            return true
        }
    }
    if err := scanner.Err(); err != nil {
        log.Printf("[WAR-41] %v", err)
    }
    return false
}