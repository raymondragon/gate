package main
import (
    "bufio"
    "crypto/tls"
    "flag"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "strings"
    "sync"
)
var (
    addr = flag.String("a", ":80", "addr")
    ibnd = flag.String("i", "", "inbound")
    obnd = flag.String("o", ":10", "outbound")
    path = flag.String("p", "/iplist", "path")
    mute = sync.Mutex{}
)
func main() {
    flag.Parse()
    _, portAddr, err := net.SplitHostPort(*addr)
    if err != nil {
        log.Fatal("[ERR-00]")
    }
    _, portObnd, err := net.SplitHostPort(*obnd)
    if err != nil {
        log.Fatal("[ERR-01]")
    }
    if portAddr == portObnd {
        log.Fatal("[ERR-10]")
    } else {
        log.Printf("[LISTEN] %v%v\n", *addr, *path)
        go ListenAndAuth()
    }
    _, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
    switch {
    case *ibnd != "" && err != nil :
        log.Println("[WAR-20]")
        log.Printf("[LISTEN] %v <-> %v\n", *obnd, *ibnd)
        go ListenAndCopyTls()
    case *ibnd != "" && err == nil :
        log.Printf("[LISTEN] %v <-> %v\n", *obnd, *ibnd)
        go ListenAndCopyTcp()
    default :
        log.Println("[WAR-21]")
    }
    select {}
}
func ListenAndAuth() {
    file, err := os.OpenFile("IPlist", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("[ERR-11]")
    }
    defer file.Close()
    http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
        ip, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Println("[ERR-12]")
            http.Error(w, "[ERR-12]", 500)
            return
        }
        if _, err := w.Write([]byte(ip+"\n")); err != nil {
            log.Println("[ERR-13]")
            http.Error(w, "[ERR-13]", 500)
            return
        }
        mute.Lock()
        defer mute.Unlock()
        if _, err := file.WriteString(ip+"\n"); err != nil {
            log.Println("[ERR-14]")
            http.Error(w, "[ERR-14]", 500)
            return
        }
    })
    log.Fatal(http.ListenAndServe(*addr, nil))
}
func ListenAndCopyTls() {
    cert, _ := tls.LoadX509KeyPair("cert.pem", "key.pem")
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
    }
    listener, err := tls.Listen("tcp", *obnd, tlsConfig)
    if err != nil {
        log.Fatal("[ERR-21]")
    }
    defer listener.Close()
    for {
        outConn, err := listener.Accept()
        if err != nil {
            log.Println("[WAR-22]")
            continue
        }
        go handleOut(outConn)
    }
}
func ListenAndCopyTcp() {
    listener, err := net.Listen("tcp", *obnd)
    if err != nil {
        log.Fatal("[ERR-21]")
    }
    defer listener.Close()
    for {
        outConn, err := listener.Accept()
        if err != nil {
            log.Println("[WAR-22]")
            continue
        }
        go handleOut(outConn)
    }
}
func handleOut(outConn net.Conn) {
    defer outConn.Close()
    clientIP := outConn.RemoteAddr().(*net.TCPAddr).IP.String()
    if !inIPlist(clientIP, "IPlist") {
        log.Println("[WAR-23]")
        return
    }
    inConn, err := net.Dial("tcp", *ibnd)
    if err != nil {
        log.Println("[WAR-24]")
        tlsConfig := &tls.Config{
            InsecureSkipVerify: true,
        }
        inConn, err = tls.Dial("tcp", *ibnd, tlsConfig)
        if err != nil {
            log.Println("[WAR-25]")
            return
        }
    }
    defer inConn.Close()
    go io.CopyBuffer(inConn, outConn, nil)
    io.CopyBuffer(outConn, inConn, nil)
}
func inIPlist(ip string, iplist string) bool {
    file, err := os.Open(iplist)
    if err != nil {
        log.Println("[WAR-26]")
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
        log.Println("[WAR-27]")
    }
    return false
}