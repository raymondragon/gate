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
    addr = flag.String("a", ":90", "addr")
    crtf = flag.String("c", "crt.pem", "")
    ibnd = flag.String("i", "", "inbound")
    keyf = flag.String("k", "key.pem", "")
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
    _, err = tls.LoadX509KeyPair(*crtf, *keyf)
    switch {
    case portAddr != portObnd && err != nil :
        log.Println("[WAR-00]")
        log.Printf("[LISTEN] %v%v\n", *addr, *path)
        go ListenAndAuthTcp()
    case portAddr != portObnd && err == nil :
        log.Printf("[LISTEN] %v%v\n", *addr, *path)
        go ListenAndAuthTls()
    default :
        log.Println("[WAR-01]")
    }
    switch {
    case *ibnd != "" && err != nil :
        log.Println("[WAR-02]")
        log.Printf("[LISTEN] %v <-> %v\n", *obnd, *ibnd)
        go ListenAndCopyTcp()
    case *ibnd != "" && err == nil :
        log.Printf("[LISTEN] %v <-> %v\n", *obnd, *ibnd)
        go ListenAndCopyTls()
    default :
        log.Println("[WAR-03]")
    }
    select {}
}
func ListenAndAuthTcp() {
    file, err := os.OpenFile("IPlist", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("[ERR-10]")
    }
    defer file.Close()
    http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
        ip, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Println("[ERR-11]")
            http.Error(w, "[ERR-11]", 500)
            return
        }
        if _, err := w.Write([]byte(ip+"\n")); err != nil {
            log.Println("[ERR-12]")
            http.Error(w, "[ERR-12]", 500)
            return
        }
        mute.Lock()
        defer mute.Unlock()
        if _, err := file.WriteString(ip+"\n"); err != nil {
            log.Println("[ERR-13]")
            http.Error(w, "[ERR-13]", 500)
            return
        }
    })
    log.Fatal(http.ListenAndServe(*addr, nil))
}
func ListenAndAuthTls() {
    file, err := os.OpenFile("IPlist", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("[ERR-20]")
    }
    defer file.Close()
    http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
        ip, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Println("[ERR-21]")
            http.Error(w, "[ERR-21]", 500)
            return
        }
        if _, err := w.Write([]byte(ip+"\n")); err != nil {
            log.Println("[ERR-22]")
            http.Error(w, "[ERR-22]", 500)
            return
        }
        mute.Lock()
        defer mute.Unlock()
        if _, err := file.WriteString(ip+"\n"); err != nil {
            log.Println("[ERR-23]")
            http.Error(w, "[ERR-23]", 500)
            return
        }
    })
    log.Fatal(http.ListenAndServeTLS(*addr, *crtf, *keyf, nil))
}
func ListenAndCopyTcp() {
    listener, err := net.Listen("tcp", *obnd)
    if err != nil {
        log.Fatal("[ERR-30]")
    }
    defer listener.Close()
    for {
        outConn, err := listener.Accept()
        if err != nil {
            log.Println("[WAR-30]")
            continue
        }
        go handleOut(outConn)
    }
}
func ListenAndCopyTls() {
    cert, _ := tls.LoadX509KeyPair(*crtf, *keyf)
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
    }
    listener, err := tls.Listen("tcp", *obnd, tlsConfig)
    if err != nil {
        log.Fatal("[ERR-40]")
    }
    defer listener.Close()
    for {
        outConn, err := listener.Accept()
        if err != nil {
            log.Println("[WAR-40]")
            continue
        }
        go handleOut(outConn)
    }
}
func handleOut(outConn net.Conn) {
    defer outConn.Close()
    clientIP := outConn.RemoteAddr().(*net.TCPAddr).IP.String()
    if !inIPlist(clientIP, "IPlist") {
        log.Println("[WAR-50]")
        return
    }
    inConn, err := net.Dial("tcp", *ibnd)
    if err != nil {
        log.Println("[WAR-51]")
        tlsConfig := &tls.Config{
            InsecureSkipVerify: true,
        }
        inConn, err = tls.Dial("tcp", *ibnd, tlsConfig)
        if err != nil {
            log.Println("[WAR-52]")
            return
        }
    }
    defer inConn.Close()
    io.Copy(inConn, outConn)
    io.Copy(outConn, inConn)
}
func inIPlist(ip string, iplist string) bool {
    file, err := os.Open(iplist)
    if err != nil {
        log.Println("[WAR-60]")
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
        log.Println("[WAR-61]")
    }
    return false
}