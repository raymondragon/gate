package main

import (
    "bufio"
    "crypto/rand"
    "crypto/rsa"
    "crypto/tls"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "flag"
    "io"
    "log"
    "math/big"
    "net"
    "net/http"
    "net/url"
    "os"
    "strings"
    "time"
)

var (
    authURL = flag.String("A", "", "Authorization: http(s)://local:port/secret_path#ipfile")
    tranURL = flag.String("T", "", "Transmission: tcp(udp)://local:port/remote:port#ipfile")
)

type ParsedURL struct {
    Scheme   string
    Hostname string
    Port     string
    Path     string
    Fragment string
}

func main() {
    flag.Parse()
    switch {
    case *authURL != "" && *tranURL != "":
        aURL, err := urlParse(*authURL)
        if err != nil {
            log.Fatalf("[ERRO-0] %v", err)
        }
        tURL, err := urlParse(*tranURL)
        if err != nil {
            log.Fatalf("[ERRO-1] %v", err)
        }
        if aURL.Fragment == "" {
            aURL.Fragment = "IPlist"
        }
        tURL.Fragment = aURL.Fragment
        log.Printf("[INFO-0] %v://%v:%v%v <-> [FILE] %v", aURL.Scheme, aURL.Hostname, aURL.Port, aURL.Path, aURL.Fragment)
        go ListenAndAuth(aURL)
        log.Printf("[INFO-1] %v://%v:%v <-> %v", tURL.Scheme, tURL.Hostname, tURL.Port, strings.TrimPrefix(tURL.Path, "/"))
        ListenAndCopy(tURL, true)
        select {}
    case *authURL != "" && *tranURL == "":
        aURL, err := urlParse(*authURL)
        if err != nil {
            log.Fatalf("[ERRO-2] %v", err)
        }
        log.Printf("[INFO-2] %v://%v:%v%v <-> [FILE] %v", aURL.Scheme, aURL.Hostname, aURL.Port, aURL.Path, aURL.Fragment)
        ListenAndAuth(aURL)
    case *authURL == "" && *tranURL != "":
        tURL, err := urlParse(*tranURL)
        if err != nil {
            log.Fatalf("[ERRO-3] %v", err)
        }
        if tURL.Fragment == "" {
            log.Printf("[INFO-3] %v://%v:%v <-> %v", tURL.Scheme, tURL.Hostname, tURL.Port, strings.TrimPrefix(tURL.Path, "/"))
            ListenAndCopy(tURL, false)
        } else {
            log.Printf("[INFO-4] %v://%v:%v <-> %v [FILE] %v", tURL.Scheme, tURL.Hostname, tURL.Port, strings.TrimPrefix(tURL.Path, "/"), tURL.Fragment)
            ListenAndCopy(tURL, true)
        }
    default:
        log.Fatalf("[ERRO-4] %v", "URL Flag Unprovided")
    }
}

func ListenAndAuth(parsedURL ParsedURL) {
    http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
        clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            log.Printf("[WARN-0] %v", err)
            return
        }
        if _, err := w.Write([]byte(clientIP + "\n")); err != nil {
            log.Printf("[WARN-1] %v", err)
            http.Error(w, "[WARN-1]", 500)
            return
        }
        file, err := os.OpenFile(parsedURL.Fragment, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            log.Printf("[WARN-2] %v", err)
            return
        }
        defer file.Close()
        if inIPlist(clientIP, parsedURL.Fragment) {
            return
        }
        if _, err := file.WriteString(clientIP + "\n"); err != nil {
            log.Printf("[WARN-3] %v", err)
            return
        }
    })
    switch parsedURL.Scheme {
    case "http":
        if err := http.ListenAndServe(parsedURL.Hostname+":"+parsedURL.Port, nil); err != nil {
            log.Fatalf("[ERRO-5] %v", err)
        }
    case "https":
        cert, err := generateCert(parsedURL.Hostname)
        if err != nil {
            log.Fatalf("[ERRO-6] %v", err)
        }
        serv := &http.Server{
            Addr:      parsedURL.Hostname + ":" + parsedURL.Port,
            TLSConfig: &tls.Config{
                Certificates: []tls.Certificate{cert},
            },
        }
        if err := serv.ListenAndServeTLS("", ""); err != nil {
            log.Fatalf("[ERRO-7] %v", err)
        }
    default:
        log.Fatalf("[ERRO-8] %v", "URL Scheme Unsupported")
    }
}

func ListenAndCopy(parsedURL ParsedURL, authEnabled bool) {
    switch parsedURL.Scheme {
    case "tcp":
        localAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Hostname+":"+parsedURL.Port)
        if err != nil {
            log.Fatalf("[ERRO-9] %v", err)
        }
        listener, err := net.ListenTCP("tcp", localAddr)
        if err != nil {
            log.Fatalf("[ERRO-A] %v", err)
        }
        defer listener.Close()
        for {
            localConn, err := listener.Accept()
            if err != nil {
                log.Printf("[WARN-4] %v", err)
                continue
            }
            go func(localConn net.Conn) {
                defer localConn.Close()
                clientIP := localConn.RemoteAddr().(*net.TCPAddr).IP.String()
                if authEnabled && !inIPlist(clientIP, parsedURL.Fragment) {
                    log.Printf("[WARN-5] %v", clientIP)
                    return
                }
                remoteConn, err := net.Dial("tcp", strings.TrimPrefix(parsedURL.Path, "/"))
                if err != nil {
                    log.Fatalf("[ERRO-B] %v", err)
                }
                defer remoteConn.Close()
                go io.Copy(remoteConn, localConn)
                io.Copy(localConn, remoteConn)
            }(localConn)
        }
    case "udp":
        localAddr, err := net.ResolveUDPAddr("udp", parsedURL.Hostname+":"+parsedURL.Port)
        if err != nil {
            log.Fatalf("[ERRO-C] %v", err)
        }
        conn, err := net.ListenUDP("udp", localAddr)
        if err != nil {
            log.Fatalf("[ERRO-D] %v", err)
        }
        defer conn.Close()
        for {
            buf := make([]byte, 1024)
            n, clientAddr, err := conn.ReadFromUDP(buf)
            if err != nil {
                log.Printf("[WARN-6] %v", err)
                continue
            }
            clientIP := clientAddr.IP.String()
            if authEnabled && !inIPlist(clientIP, parsedURL.Fragment) {
                log.Printf("[WARN-7] %v", clientIP)
                continue
            }
            remoteAddr, err := net.ResolveUDPAddr("udp", strings.TrimPrefix(parsedURL.Path, "/"))
            if err != nil {
                log.Fatalf("[ERRO-E] %v", err)
            }
            if _, err := conn.WriteToUDP(buf[:n], remoteAddr); err != nil {
                log.Printf("[WARN-8] %v", err)
            }
        }
    default:
        log.Fatalf("[ERRO-F] %v", "URL Scheme Unsupported")
    }
}

func urlParse(rawURL string) (ParsedURL, error) {
    u, err := url.Parse(rawURL)
    if err != nil {
        return ParsedURL{}, err
    }
    return ParsedURL{
        Scheme:   u.Scheme,
        Hostname: u.Hostname(),
        Port:     u.Port(),
        Path:     u.Path,
        Fragment: u.Fragment,
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

func generateCert(orgName string) (tls.Certificate, error) {
    priv, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return tls.Certificate{}, err
    }
    serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
    if err != nil {
        return tls.Certificate{}, err
    }
    template := x509.Certificate{
        SerialNumber: serialNumber,
        Subject:      pkix.Name{
            Organization: []string{orgName},
        },
        NotBefore:    time.Now(),
        NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
        KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    }
    crtDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
    if err != nil {
        return tls.Certificate{}, err
    }
    crtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: crtDER})
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
    return tls.X509KeyPair(crtPEM, keyPEM)
}
