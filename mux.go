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

	listener, err := net.Listen("tcp", "0.0.0.0:"+publicPort)
	if err != nil {
		log.Fatalf("[Mux] Gagal listen di port %s: %v", publicPort, err)
	}
	defer listener.Close()

	log.Printf("[Mux] Jalan di 0.0.0.0:%s -> SSL:%s | Sisa Traffic (WS/Payload):%s", publicPort, sslTarget, wsTarget)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(clientConn, sslTarget, wsTarget)
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

func handleClient(client net.Conn, sslTarget, wsTarget string) {
	tweakSocket(client)
	defer client.Close()

	reader := bufio.NewReaderSize(client, 65536)

	// 🔥 FIX: Hapus batas waktu 3 detik di sini agar jabat tangan awal via proxy Railway tidak terputus di jalan
	peekBytes, err := reader.Peek(4)

	var targetAddr string
	var label string

	// Proteksi anti crash tetap aktif jika data kosong / koneksi bermasalah
	if err != nil || len(peekBytes) == 0 {
		targetAddr = wsTarget
		label = "WS-Proxy (Error/Empty)"
	} else if peekBytes[0] == TLSHandshakeByte {
		// 1. JALUR SNI / SSL: Jika terdeteksi TLS Handshake, lempar ke Stunnel (2443)
		targetAddr = sslTarget
		label = "SSL/Stunnel (SNI Traffic)"
	} else if bytes.HasPrefix(peekBytes, []byte("SSH-")) {
		// 2. JALUR RAW SSH: Jika langsung nembak SSH biasa dari Termux/Bitvise murni
		targetAddr = "127.0.0.1:22"
		label = "Raw OpenSSH (Port 22)"
	} else {
		// 3. JALUR PAYLOAD / SISA TRAFFIC: Langsung hajar masuk ke port 8880 (ws-proxy)
		targetAddr = wsTarget
		label = "WS-Proxy / Payload Backend"
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
	
	// Alirkan data dari buffer reader ke backend
	go func() {
		_, _ = io.Copy(backendConn, reader) 
		done <- struct{}{}
	}()
	
	// Alirkan data balik dari backend ke client
	go func() {
		_, _ = io.Copy(client, backendConn)
		done <- struct{}{}
	}()

	<-done
}
