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
    loclAddr = flag.String("L", "", "Local")
    remtAddr = flag.String("R", "", "Remote")
    authAddr = flag.String("addr", "", "Auth")
    authPath = flag.String("path", "", "Auth")
)
func main() {
    flag.Parse()
    _, portLocl, err := net.SplitHostPort(*loclAddr)
    if err != nil {
        log.Fatalf("[ERR-00] %v", err)
    }
    switch *authAddr {
    case "":
        if *remtAddr != "" {
            log.Printf("[LISTEN] %v <-> %v", *loclAddr, *remtAddr)
            ListenAndCopy(*loclAddr, *remtAddr, *authAddr)
        } else {
            log.Printf("[WAR-00] %v", "None Remote Service")
        }
    default:
        _, portAuth, err := net.SplitHostPort(*authAddr)
        if err != nil {
            log.Fatalf("[ERR-01] %v", err)
        }
        if portAuth != portLocl {
            log.Printf("[LISTEN] %v%v", *authAddr, *authPath)
            go ListenAndAuth(*authAddr, *authPath)
        } else {
            log.Fatalf("[ERR-02] %v", "Server Port Conflict")
        }
        if *remtAddr != "" {
            log.Printf("[LISTEN] %v <-> %v", *loclAddr, *remtAddr)
            ListenAndCopy(*loclAddr, *remtAddr, *authAddr)
        } else {
            log.Printf("[WAR-01] %v", "None Remote Service")
        }
        select {}
    }
}
func ListenAndAuth(authAddr string, authPath string) {
    http.HandleFunc(authPath, func(w http.ResponseWriter, r *http.Request) {
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
    if err := http.ListenAndServe(authAddr, nil); err != nil {
        log.Fatalf("[ERR-10] %v", err)
    }
}
func ListenAndCopy(loclAddr string, remtAddr string, authAddr string) {
    listener, err := net.Listen("tcp", loclAddr)
    if err != nil {
        log.Printf("[WAR-20] %v", err)
        return
    }
    defer listener.Close()
    for {
        loclConn, err := listener.Accept()
        if err != nil {
            log.Printf("[WAR-21] %v", err)
            continue
        }
        go func(loclConn net.Conn) {
            defer loclConn.Close()
            clientIP := loclConn.RemoteAddr().(*net.TCPAddr).IP.String()
            if authAddr != "" && !inIPlist(clientIP, "IPlist") {
                log.Printf("[WAR-22] %v", clientIP)
                return
            }
            remtConn, err := net.Dial("tcp", remtAddr)
            if err != nil {
                log.Printf("[WAR-23] %v", err)
                return
            }
            defer remtConn.Close()
            go io.Copy(remtConn, loclConn)
            io.Copy(loclConn, remtConn)
        }(loclConn)
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