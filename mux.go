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
		// Jika diawali dengan byte TLS Handshake, lempar ke Stunnel
		targetAddr = sslTarget
		label = "SSL/Stunnel"
	} else if bytes.HasPrefix(peekBytes, []byte("SSH-")) {
		// Jika menggunakan Raw SSH biasa (Langsung tembak dari Termux/Bitvise tanpa payload)
		targetAddr = "127.0.0.1:22"
		label = "Raw OpenSSH (Port 22)"
	} else {
		// --- SMART INTELLIGENT ROUTER UNTUK PAYLOAD & HTTP ---
		// Kita intip buffer yang agak besar (1024 byte) untuk memastikan HTTP Header sudah terbaca sepenuhnya
		maxPeek := 1024
		if reader.Buffered() > maxPeek {
			maxPeek = reader.Buffered()
		}
		
		bufferedBytes, _ := reader.Peek(maxPeek)
		
		if bytes.Contains(bufferedBytes, []byte("dfathu.web.id")) || bytes.Contains(bufferedBytes, []byte("GET /api/")) {
			// Jika request dari browser biasa atau API log web
			targetAddr = "127.0.0.1:8081"
			label = "Web UI Python (Argo Host Route)"
		} else if bytes.Contains(bufferedBytes, []byte("7300")) || bytes.Contains(bufferedBytes, []byte("badvpn")) || bytes.Contains(bufferedBytes, []byte("UDPGW")) {
			// Deteksi Game Mode BadVPN UDPGW
			targetAddr = udpgwTarget
			label = "BadVPN-UDPGW (Game Mode)"
		} else if bytes.Contains(bufferedBytes, []byte("Upgrade: websocket")) || bytes.Contains(bufferedBytes, []byte("Connection: Upgrade")) {
			// Deteksi SSH over WebSocket murni
			targetAddr = wsTarget
			label = "WS-Proxy (Websocket Upgrade)"
		} else {
			// Tampungan terakhir untuk HTTP Payload kustom (HTTP Custom, DarkTunnel, dll)
			targetAddr = wsTarget
			label = "WS-Proxy (HTTP Payload)"
		}
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
