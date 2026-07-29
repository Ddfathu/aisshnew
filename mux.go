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

	// 🛠️ JALUR KUNCI BARU: Ambil konfigurasi port BadVPN UDPGW dari entrypoint.sh
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
		// Kirim target udpgwTarget ke handlerna
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

	// --- OPTIMASI BUFFER: Diperbesar jadi 64KB ---
	reader := bufio.NewReaderSize(client, 65536)

	// Batasi waktu ngintip byte pertama (Anti-Stuck / Anti-Sunek)
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	
	// 🔥 DIUBAH: Intip 4 byte pertama sekaligus biar kebaca string "SSH-" dari Termux
	peekBytes, err := reader.Peek(4)
	
	// Reset kembali deadline ke normal agar koneksi tidak terputus setelah 3 detik
	_ = client.SetReadDeadline(time.Time{})

	var targetAddr string
	var label string

	// Deteksi protokol berdasarkan byte pertama dan isi intipan buffer teks
	if err != nil {
		targetAddr = wsTarget
		label = "WS-Proxy (Default/Timeout)"
	} else if peekBytes[0] == TLSHandshakeByte {
		targetAddr = sslTarget
		label = "SSL/Stunnel"
	} else if bytes.HasPrefix(peekBytes, []byte("SSH-")) {
		// 🔥 BYPASS JALUR RAW SSH: Jika Termux nembak SSH biasa, bypass langsung ke OpenSSH lokal port 22
		targetAddr = "127.0.0.1:22"
		label = "Raw OpenSSH (Port 22)"
	} else {
		// 🛠️ SMART INTELLIGENT ROUTER UNTUK BROWSER & PAYLOAD
		bufferedBytes, peekErr := reader.Peek(reader.Buffered())
		
		// Cek apakah request HTTP membawa header domain kustom lu ATAU menembak API log
		if peekErr == nil && (bytes.Contains(bufferedBytes, []byte("dfathu.web.id")) || bytes.Contains(bufferedBytes, []byte("GET /api/"))) {
			// Belokkan request browser biasa langsung ke port Web server Python internal port 8081
			targetAddr = "127.0.0.1:8081"
			label = "Web UI Python (Argo Host Route)"
		} else if peekErr == nil && (bytes.Contains(bufferedBytes, []byte("7300")) || bytes.Contains(bufferedBytes, []byte("badvpn")) || bytes.Contains(bufferedBytes, []byte("UDPGW"))) {
			// 🛠️ SMART DETECTOR UNTUK GAME / UDPGW
			targetAddr = udpgwTarget
			label = "BadVPN-UDPGW (Game Mode)"
		} else {
			// Payload DarkTunnel / HTTP Custom (GET / polos tanpa domain kustom) otomatis aman lolos ke sini!
			targetAddr = wsTarget
			label = "WS-Proxy"
		}
	}

	log.Printf("[Mux] Koneksi dari %s dialihkan ke %s (%s)", client.RemoteAddr(), label, targetAddr)

	backendConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		log.Printf("[Mux] Gagal konek ke backend %s: %v", label, err)
		return
	}
	tweakSocket(backendConn)
	defer backendConn.Close()

	done := make(chan struct{}, 2)
	
	// Alirkan data dari buffer reader ke backend secara loss
	go func() {
		_, _ = io.Copy(backendConn, reader) 
		done <- struct{}{}
	}()
	
	// Alirkan data dari backend balik ke client
	go func() {
		_, _ = io.Copy(client, backendConn)
		done <- struct{}{}
	}()

	<-done
}
