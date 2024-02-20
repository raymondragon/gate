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

    addr = flag.String("a", ":1", "addr")
    ibnd = flag.String("i", ":1001", "i")
    obnd = flag.String("o", ":1000", "o")
    path = flag.String("p", "./", "path")
    mute = sync.Mutex{}

    //bind = flag.String("b", ":10000", "bind")
    //ipst = flag.String("i", "IPlist", "iplist")
    //tars = flag.String("t", "", "target")
)
func main() {
    flag.Parse()
    if *tars == "" {
        log.Fatal("[ERR-0] Target Server Info Required")
    }
    listener, err := net.Listen("tcp", *bind)
    if err != nil {
        log.Fatal("[ERR-1] ", err)
    }
    defer listener.Close()
    for {
        clientConn, err := listener.Accept()
        if err != nil {
            log.Println("[ERR-2] ", err)
            continue
        }
        go handleClient(clientConn)
    }
}
func handleClient(clientConn net.Conn) {
    defer clientConn.Close()
    clientIP := clientConn.RemoteAddr().(*net.TCPAddr).IP.String()
    if !inIPlist(clientIP, *ipst) {
        log.Println("[ERR-3] ", clientIP)
        return
    }
    serverConn, err := net.Dial("tcp", *tars)
    if err != nil {
        log.Println("[ERR-4] ", err)
        return
    }
    defer serverConn.Close()
    go io.CopyBuffer(serverConn, clientConn, nil)
    io.CopyBuffer(clientConn, serverConn, nil)
}
func inIPlist(ip string, iplist string) bool {
    file, err := os.Open(iplist)
    if err != nil {
        log.Println("[ERR-5] ", err)
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
        log.Println("[ERR-6] ", err)
    }
    return false
}



package main
import (

)
var (

)
func main() {
    flag.Parse()
    file, err := os.OpenFile("IPlist", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("[ERR-0] ", err)
    }
    defer file.Close()
    http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
        ip, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Println("[ERR-1] ", err)
            http.Error(w, "[ERR-1]", http.StatusInternalServerError)
            return
        }
        if _, err := w.Write([]byte(ip+"\n")); err != nil {
            log.Println("[ERR-2] ", err)
            http.Error(w, "[ERR-2]", http.StatusInternalServerError)
            return
        }
        mute.Lock()
        defer mute.Unlock()
        if _, err := file.WriteString(ip+"\n"); err != nil {
            log.Println("[ERR-3] ", err)
            http.Error(w, "[ERR-3]", http.StatusInternalServerError)
            return
        }
    })
    log.Fatal(http.ListenAndServe(*addr, nil))
}