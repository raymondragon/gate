package main
import (
    "bufio"
    "flag"
    "io"
    "log"
    "net"
    "os"
    "strings"
)
var (
    bind = flag.String("b", ":10000", "bind")
    ipst = flag.String("i", "IPlist", "iplist")
    tars = flag.String("t", "", "target")
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