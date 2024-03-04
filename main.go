package main
import (
    "bufio"
    "flag"
    "io"
    "log"
    "net"
    "net/http"
    "net/url"
    "os"
    "strings"
)
var (
    authRurl = flag.String("A", "", "Auth")
    loclAddr = flag.String("L", "", "Local")
    remtAddr = flag.String("R", "", "Remote")
)
type ParsedURL struct {
    Hostname string
    Port     string
    Path     string
}
func main() {
    flag.Parse()
    _, portLocl, err := net.SplitHostPort(*loclAddr)
    if err != nil {
        log.Fatalf("[ERRO-0] %v", err)
    }
    switch *authRurl {
    case "":
        if *remtAddr != "" {
            log.Printf("[INFO-0] %v <-> %v", *loclAddr, *remtAddr)
            ListenAndCopy(*loclAddr, *remtAddr, *authRurl)
        } else {
            log.Fatalf("[ERRO-1] %v", "None Remote Service")
        }
    default:
        parsedURL, err := urlParse(*authRurl)
        if err != nil {
            log.Fatalf("[ERRO-2] %v", err)
        }
        if parsedURL.Port != portLocl {
            log.Printf("[INFO-1] %v", *authRurl)
            go ListenAndAuth(parsedURL.Hostname, parsedURL.Port, parsedURL.Path)
        } else {
            log.Fatalf("[ERRO-3] %v", "Server Port Conflict")
        }
        if *remtAddr != "" {
            log.Printf("[INFO-2] %v <-> %v", *loclAddr, *remtAddr)
            ListenAndCopy(*loclAddr, *remtAddr, *authRurl)
        } else {
            log.Printf("[WARN-0] %v", "Remote Service Unprovided")
        }
        select {}
    }
}
func ListenAndAuth(authName string, authPort string, authPath string) {
    http.HandleFunc(authPath, func(w http.ResponseWriter, r *http.Request) {
        clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Printf("[WARN-1] %v", err)
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
    if err := http.ListenAndServe(authName+":"+authPort, nil); err != nil {
        log.Fatalf("[ERRO-4] %v", err)
    }
}
func ListenAndCopy(loclAddr string, remtAddr string, authRurl string) {
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
            if authRurl != "" && !inIPlist(loclConn.RemoteAddr().(*net.TCPAddr).IP.String(), "IPlist") {
                return
            }
            remtConn, err := net.Dial("tcp", remtAddr)
            if err != nil {
                log.Printf("[WARN-8] %v", err)
                return
            }
            defer remtConn.Close()
            go io.Copy(remtConn, loclConn)
            io.Copy(loclConn, remtConn)
        }(loclConn)
    }
}
func urlParse(rawURL string) (ParsedURL, error) {
    u, err := url.Parse(rawURL)
    if err != nil {
        return ParsedURL{}, err
    }
    return ParsedURL{
        Hostname: u.Hostname(),
        Port:     u.Port(),
        Path:     u.Path,
    }, nil
}
func inIPlist(ip string, list string) bool {
    file, err := os.Open(list)
    if err != nil {
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