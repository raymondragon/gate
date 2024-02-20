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
    "sync"
)
var (
    addr = flag.String("a", ":8080", "addr")
    ibnd = flag.String("i", "", "inbound")
    obnd = flag.String("o", ":1", "outbound")
    path = flag.String("p", "/auth", "path")
    mute = sync.Mutex{}
)
func main() {
    flag.Parse()
    _, portAddr, err := net.SplitHostPort(*addr)
    if err != nil {
        log.Fatal("[ERR-00] ", err)
    }
    _, portObnd, err := net.SplitHostPort(*obnd)
    if err != nil {
        log.Fatal("[ERR-01] ", err)
    }
    if portAddr == portObnd {
        log.Fatal("[ERR-10]")
    } else {
        log.Printf("[LISTEN] %v%v\n", *addr, *path)
        ListenAndAuth()
    }
    if *ibnd == "" {
        log.Println("[WAR-20]")
    } else {
        log.Printf("[INBOUND] %v [OUTBOUND] %v\n", *ibnd, *obnd)
        ListenAndCopy()
    }
}
func ListenAndAuth() {
    file, err := os.OpenFile("IPlist", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("[ERR-11] ", err)
    }
    defer file.Close()
    http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
        ip, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Println("[ERR-12] ", err)
            http.Error(w, "[ERR-12]", http.StatusInternalServerError)
            return
        }
        if _, err := w.Write([]byte(ip+"\n")); err != nil {
            log.Println("[ERR-13] ", err)
            http.Error(w, "[ERR-13]", http.StatusInternalServerError)
            return
        }
        mute.Lock()
        defer mute.Unlock()
        if _, err := file.WriteString(ip+"\n"); err != nil {
            log.Println("[ERR-14] ", err)
            http.Error(w, "[ERR-14]", http.StatusInternalServerError)
            return
        }
    })
    log.Fatal(http.ListenAndServe(*addr, nil))
}
func ListenAndCopy() {
    listener, err := net.Listen("tcp", *obnd)
    if err != nil {
        log.Fatal("[ERR-21] ", err)
    }
    defer listener.Close()
    for {
        outConn, err := listener.Accept()
        if err != nil {
            log.Println("[WAR-22] ", err)
            continue
        }
        go handleOut(outConn)
    }
}
func handleOut(outConn net.Conn) {
    defer outConn.Close()
    clientIP := outConn.RemoteAddr().(*net.TCPAddr).IP.String()
    if !inIPlist(clientIP, "IPlist") {
        log.Println("[WAR-23] ", clientIP)
        return
    }
    inConn, err := net.Dial("tcp", *ibnd)
    if err != nil {
        log.Println("[WAR-24] ", err)
        return
    }
    defer inConn.Close()
    go io.CopyBuffer(inConn, outConn, nil)
    io.CopyBuffer(outConn, inConn, nil)
}
func inIPlist(ip string, iplist string) bool {
    file, err := os.Open(iplist)
    if err != nil {
        log.Println("[WAR-25] ", err)
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
        log.Println("[WAR-26] ", err)
    }
    return false
}