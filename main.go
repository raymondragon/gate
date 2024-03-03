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
        log.Fatalf("[ERRO-0] %v", err)
    }
    switch *authAddr {
    case "":
        if *remtAddr != "" {
            log.Printf("[INFO-0] %v <-> %v", *loclAddr, *remtAddr)
            ListenAndCopy(*loclAddr, *remtAddr, *authAddr)
        } else {
            log.Fatalf("[ERRO-1] %v", "None Remote Service")
        }
    default:
        _, portAuth, err := net.SplitHostPort(*authAddr)
        if err != nil {
            log.Fatalf("[ERRO-2] %v", err)
        }
        if portAuth != portLocl {
            log.Printf("[INFO-1] Auth -> %v%v", *authAddr, *authPath)
            go ListenAndAuth(*authAddr, *authPath)
        } else {
            log.Fatalf("[ERRO-3] %v", "Server Port Conflict")
        }
        if *remtAddr != "" {
            log.Printf("[INFO-2] %v <-> %v", *loclAddr, *remtAddr)
            ListenAndCopy(*loclAddr, *remtAddr, *authAddr)
        } else {
            log.Printf("[WARN-0] %v", "Remote Service Unprovided")
        }
        select {}
    }
}
func ListenAndAuth(authAddr string, authPath string) {
    http.HandleFunc(authPath, func(w http.ResponseWriter, r *http.Request) {
        clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Printf("[WARN-1] %v", err)
            http.Error(w, "[WARN-1]", 500)
            return
        }
        if _, err := w.Write([]byte(clientIP + "\n")); err != nil {
            log.Printf("[WARN-2] %v", err)
            http.Error(w, "[WARN-2]", 500)
            return
        }
        file, err := os.OpenFile("IPlist", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            log.Printf("[WARN-3] %v", err)
            return
        }
        defer file.Close()
        if inIPlist(clientIP, "IPlist") {
            log.Printf("[WARN-4] %v", clientIP)
            return
        }
        if _, err := file.WriteString(clientIP + "\n"); err != nil {
            log.Printf("[WARN-5] %v", err)
            return
        }
    })
    if err := http.ListenAndServe(authAddr, nil); err != nil {
        log.Fatalf("[ERRO-4] %v", err)
    }
}
func ListenAndCopy(loclAddr string, remtAddr string, authAddr string) {
    listener, err := net.Listen("tcp", loclAddr)
    if err != nil {
        log.Printf("[WARN-6] %v", err)
        return
    }
    defer listener.Close()
    for {
        loclConn, err := listener.Accept()
        if err != nil {
            log.Printf("[WARN-7] %v", err)
            continue
        }
        go func(loclConn net.Conn) {
            defer loclConn.Close()
            clientIP := loclConn.RemoteAddr().(*net.TCPAddr).IP.String()
            if authAddr != "" && !inIPlist(clientIP, "IPlist") {
                log.Printf("[WARN-8] %v", clientIP)
                return
            }
            remtConn, err := net.Dial("tcp", remtAddr)
            if err != nil {
                log.Printf("[WARN-9] %v", err)
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