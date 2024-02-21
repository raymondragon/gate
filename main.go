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
        log.Fatal("[ERR-00]", err)
    }
    _, portObnd, err := net.SplitHostPort(*obnd)
    if err != nil {
        log.Fatal("[ERR-01]", err)
    }
    if portAddr != portObnd {
        log.Printf("[LISTEN] %v%v\n", *addr, *path)
        go ListenAndAuth()
    } else {
        log.Fatal("[ERR-03]", "Server Port Conflict")
    }
    if *ibnd != "" {
        log.Printf("[LISTEN] %v <-> %v\n", *obnd, *ibnd)
        go ListenAndCopy()
    } else {
        log.Println("[WAR-00]", "No Inbound Service")
    }
    select {}
}
func ListenAndAuth() {
    file, err := os.OpenFile("IPlist", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("[ERR-10]", err)
    }
    defer file.Close()
    http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
        ip, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Println("[ERR-11]", err)
            http.Error(w, "[ERR-11]", 500)
            return
        }
        if _, err := w.Write([]byte(ip+"\n")); err != nil {
            log.Println("[ERR-12]", err)
            http.Error(w, "[ERR-12]", 500)
            return
        }
        mute.Lock()
        defer mute.Unlock()
        if _, err := file.WriteString(ip+"\n"); err != nil {
            log.Println("[ERR-13]", err)
            http.Error(w, "[ERR-13]", 500)
            return
        }
    })
    if _, err := tls.LoadX509KeyPair(*crtf, *keyf); err!=nil {
        log.Println("[WAR-10]", err)
        log.Fatal(http.ListenAndServe(*addr, nil))
    } else {
        log.Fatal(http.ListenAndServeTLS(*addr, *crtf, *keyf, nil))
    }
}
func ListenAndCopy() {
    var listener net.Listener
    if _, err := tls.LoadX509KeyPair(*crtf, *keyf); err!=nil {
        log.Println("[WAR-20]", err)
        listener, err = net.Listen("tcp", *obnd)
        if err != nil {
            log.Println("[WAR-21]", err)
        } else {
            cert, _ := tls.LoadX509KeyPair(*crtf, *keyf)
            listener, err = tls.Listen("tcp", *obnd, &tls.Config{
                Certificates: []tls.Certificate{cert},
            })
            if err != nil {
                log.Fatal("[ERR-20]", err)
            }
        }
    }
    defer listener.Close()
    for {
        outConn, err := listener.Accept()
        if err != nil {
            log.Println("[WAR-21]", err)
            continue
        }
        go handleOut(outConn)
    }
}
func handleOut(outConn net.Conn) {
    defer outConn.Close()
    clientIP := outConn.RemoteAddr().(*net.TCPAddr).IP.String()
    if !inIPlist(clientIP, "IPlist") {
        log.Println("[WAR-30]", clientIP)
        return
    }
    inConn, err := net.Dial("tcp", *ibnd)
    if err != nil {
        log.Println("[WAR-31]", err)
        inConn, err = tls.Dial("tcp", *ibnd, &tls.Config{
            InsecureSkipVerify: true,
        })
        if err != nil {
            log.Println("[WAR-32]", err)
            return
        }
    }
    defer inConn.Close()
    go io.Copy(inConn, outConn)
    io.Copy(outConn, inConn)
}
func inIPlist(ip string, iplist string) bool {
    file, err := os.Open(iplist)
    if err != nil {
        log.Println("[WAR-40]", err)
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
        log.Println("[WAR-41]", err)
    }
    return false
}