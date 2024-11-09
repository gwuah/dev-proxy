package internal

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"
)

type DevProxy struct {
	db        *sql.DB
	logger    *slog.Logger
	transport *http.Transport
}

func NewDevProxy(logger *slog.Logger, db *sql.DB) *DevProxy {
	return &DevProxy{
		db:     db,
		logger: logger,
		transport: &http.Transport{
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func (p *DevProxy) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	outReq := new(http.Request)
	*outReq = *r

	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Upgrade",
	}

	connectionHeaders := outReq.Header.Get("Connection")
	if connectionHeaders != "" {
		for _, h := range strings.Split(connectionHeaders, ",") {
			hopByHopHeaders = append(hopByHopHeaders, strings.TrimSpace(h))
		}
	}

	for _, h := range hopByHopHeaders {
		outReq.Header.Del(h)
	}

	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		outReq.Header.Set("X-Forwarded-For", clientIP)
	}

	resp, err := p.transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func (p *DevProxy) HandleHTTPS(w http.ResponseWriter, r *http.Request, ca *x509.Certificate, caCert *tls.Certificate) {
	hij, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hij.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	host := r.Host

	cert, err := generateCert(host, ca, caCert)
	if err != nil {
		log.Printf("Certificate generation error: %v", err)
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()

	err = tlsConn.Handshake()
	if err != nil {
		log.Printf("Handshake error: %v", err)
		return
	}

	httpsRequest, err := http.ReadRequest(bufio.NewReader(tlsConn))
	if err != nil {
		log.Printf("Error reading HTTPS request: %v", err)
		return
	}

	body, err := io.ReadAll(httpsRequest.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		return
	}
	log.Printf("HTTPS Request Body: %s", string(body))

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	targetConn, err := tls.DialWithDialer(dialer, "tcp", r.Host, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("Error connecting to target: %v", err)
		return
	}
	defer targetConn.Close()

	httpsRequest.Header.Set("Host", host)
	fmt.Fprintf(targetConn, "%s %s HTTP/1.1\r\n", httpsRequest.Method, httpsRequest.URL)
	httpsRequest.Header.Write(targetConn)
	fmt.Fprintf(targetConn, "\r\n")
	targetConn.Write(body)

	go io.Copy(targetConn, tlsConn)
	io.Copy(tlsConn, targetConn)
}

func generateCert(host string, ca *x509.Certificate, caCert *tls.Certificate) (*tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	cleanHost := strings.Split(host, ":")[0]
	dnsNames := []string{cleanHost}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, ca, &privateKey.PublicKey, caCert.PrivateKey)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privateKey,
	}

	return cert, nil
}
