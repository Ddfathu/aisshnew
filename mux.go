package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const (
	TLSHandshakeByte = 0x16
	SocketBuffer     = 524288 // 512KB Kernel Socket Buffer
)

func main() {
	publicPort := os.Getenv("PORT")
	if publicPort == "" {
		publicPort = "8080"
	}

	sslTarget := os.Getenv("SSL_TARGET_HOST") + ":" + os.Getenv("SSL_TARGET_PORT")
	if sslTarget == ":" {
		sslTarget = "127.0.0.1:2443"
	}

	wsTarget := os.Getenv("WS_MUX_TARGET_HOST") + ":" + os.Getenv("WS_MUX_TARGET_PORT")
	if wsTarget == ":" {
		wsTarget = "127.0.0.1:8880"
	}

	udpgwHost := os.Getenv("UDPGW_TARGET_HOST")
	udpgwPort := os.Getenv("UDPGW_TARGET_PORT")
	if udpgwHost == "" {
		udpgwHost = "127.0.0.1"
	}
	if udpgwPort == "" {
		udpgwPort = "7300"
	}
	udpgwTarget := udpgwHost + ":" + udpgwPort

	listener, err := net.Listen("tcp", "0.0.0.0:"+publicPort)
	if err != nil {
		log.Fatalf("[Mux] Gagal listen di port %s: %v", publicPort, err)
	}
	defer listener.Close()

	log.Printf("[Mux] Jalan di 0.0.0.0:%s -> SSL:%s | WS:%s | UDPGW:%s -> Ultra Game Mode", publicPort, sslTarget, wsTarget, udpgwTarget)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(clientConn, sslTarget, wsTarget, udpgwTarget)
	}
}

func tweakSocket(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(10 * time.Second)
		_ = tcpConn.SetReadBuffer(SocketBuffer)
		_ = tcpConn.SetWriteBuffer(SocketBuffer)
	}
}

func handleClient(client net.Conn, sslTarget, wsTarget, udpgwTarget string) {
	tweakSocket(client)
	defer client.Close()

	// Buffer pembaca dinaikkan ke 64KB untuk stabilitas traffic tinggi
	reader := bufio.NewReaderSize(client, 65536)

	// Batasi waktu ngintip byte pertama (Anti-Stuck / Anti-Sunek)
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	
	// Intip 4 byte pertama untuk deteksi protokol dasar
	peekBytes, err := reader.Peek(4)
	
	// Reset kembali deadline ke normal agar koneksi tidak putus di tengah jalan
	_ = client.SetReadDeadline(time.Time{})

	var targetAddr string
	var label string

	if err != nil {
		targetAddr = wsTarget
		label = "WS-Proxy (Default/Timeout)"
	} else if peekBytes[0] == TLSHandshakeByte {
		// 1. JALUR SNI / SSL: Jika terdeteksi TLS Handshake, langsung lempar ke Stunnel (2443)
		targetAddr = sslTarget
		label = "SSL/Stunnel (SNI Traffic)"
	} else if bytes.HasPrefix(peekBytes, []byte("SSH-")) {
		// 2. JALUR RAW SSH: Jika langsung nembak SSH biasa (Direct tanpa payload)
		targetAddr = "127.0.0.1:22"
		label = "Raw OpenSSH (Port 22)"
	} else if isHTTPMethod(peekBytes) {
		// 3. JALUR HTTP / SSH PAYLOAD VVPN (DarkTunnel, HTTP Custom, dll)
		// Jika diawali teks HTTP Method (GET, POST, CONN, MKCO), amankan rutenya.
		
		maxPeek := 1024
		if reader.Buffered() > maxPeek {
			maxPeek = reader.Buffered()
		}
		bufferedBytes, _ := reader.Peek(maxPeek)
		
		if bytes.Contains(bufferedBytes, []byte("dfathu.web.id")) || bytes.Contains(bufferedBytes, []byte("GET /api/")) {
			// Jalur khusus untuk Web UI Python Dashboard kamu
			targetAddr = "127.0.0.1:8081"
			label = "Web UI Python"
		} else if bytes.Contains(bufferedBytes, []byte("7300")) || bytes.Contains(bufferedBytes, []byte("badvpn")) || bytes.Contains(bufferedBytes, []byte("UDPGW")) {
			// Deteksi Game Mode BadVPN UDPGW
			targetAddr = udpgwTarget
			label = "BadVPN-UDPGW (Game Mode)"
		} else {
			// SANGAT PENTING: Semua sisa Payload VPN / Trik Split otomatis dipaksa masuk ke WS Proxy (8880)
			targetAddr = wsTarget
			label = "WS-Proxy (SSH Payload/Split)"
		}
	} else {
		// Tampungan terakhir jika data tidak dikenal
		targetAddr = wsTarget
		label = "WS-Proxy (Unknown Plaintext)"
	}

	log.Printf("[Mux] Koneksi dari %s dialihkan ke %s (%s)", client.RemoteAddr(), label, targetAddr)

	// Hubungkan ke backend target
	backendConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		log.Printf("[Mux] Gagal konek ke backend %s: %v", label, err)
		return
	}
	tweakSocket(backendConn)
	defer backendConn.Close()

	done := make(chan struct{}, 2)
	
	// Alirkan data dari buffer reader (termasuk data yang di-peek) ke backend
	go func() {
		_, _ = io.Copy(backendConn, reader) 
		done <- struct{}{}
	}()
	
	// Alirkan data balik dari backend ke client
	go func() {
		_, _ = io.Copy(client, backendConn)
		done <- struct{}{}
	}()

	// Tunggu sampai salah satu koneksi selesai/terputus
	<-done
}

// Fungsi pembantu untuk mendeteksi apakah data awal berbentuk HTTP Method (Payload)
func isHTTPMethod(b []byte) bool {
	methods := [][]byte{
		[]byte("GET"), []byte("POST"), []byte("CONN"), 
		[]byte("HEAD"), []byte("PUT"), []byte("MKCO"),
	}
	for _, m := range methods {
		if bytes.HasPrefix(b, m) {
			return true
		}
	}
	return false
}
