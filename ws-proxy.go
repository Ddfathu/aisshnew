package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	WSMagic    = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	BufferSize = 65536 // 64KB Buffer untuk performa maksimal
)

func main() {
	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "8880"
	}
	sshTarget := "127.0.0.1:22"

	listener, err := net.Listen("tcp", "127.0.0.1:"+wsPort)
	if err != nil {
		log.Fatalf("[WS] Gagal listen internal: %v", err)
	}
	defer listener.Close()

	log.Printf("[WS Engine] Listen internal aktif di 127.0.0.1:%s -> Forward ke SSH: %s", wsPort, sshTarget)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleWS(clientConn, sshTarget)
	}
}

func tweakSocket(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)                     // Matikan Nagle Algorithm (Anti-delay)
		_ = tcpConn.SetKeepAlive(true)                   // Aktifkan TCP Keepalive
		_ = tcpConn.SetKeepAlivePeriod(10 * time.Second) // Cek berkala setiap 10 detik
		_ = tcpConn.SetReadBuffer(BufferSize)
		_ = tcpConn.SetWriteBuffer(BufferSize)
	}
}

func handleWS(client net.Conn, sshTarget string) {
	tweakSocket(client)
	defer client.Close()

	// Baca header lebih fleksibel, jangan patok batas 4096 mati
	headerBuf := make([]byte, BufferSize) 
	
	// Set timeout pembacaan header agar tidak digantung koneksi hantu
	_ = client.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := client.Read(headerBuf)
	_ = client.SetReadDeadline(time.Time{}) // Kembalikan ke normal
	
	if err != nil || n == 0 {
		return
	}

	rawHeaders := string(headerBuf[:n])
	rawLower := strings.ToLower(rawHeaders)

	// Proses jabat tangan WebSocket dengan lebih tangguh
	if strings.Contains(rawLower, "upgrade: websocket") || strings.Contains(rawLower, "websocket") {
		wsKey := ""
		lines := strings.Split(rawHeaders, "\r\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.ToLower(line), "sec-websocket-key:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					wsKey = strings.TrimSpace(parts[1])
				}
				break
			}
		}

		if wsKey == "" {
			wsKey = base64.StdEncoding.EncodeToString([]byte(time.Now().String()))
		}

		h := sha1.New()
		h.Write([]byte(wsKey + WSMagic))
		acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

		response := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"
		_, _ = client.Write([]byte(response))
	} else {
		defaultResp := os.Getenv("WS_RESPONSE")
		if defaultResp == "" {
			defaultResp = "HTTP/1.1 101 Switching Protocols\r\n\r\n"
		}
		_, _ = client.Write([]byte(defaultResp))
	}

	// Hubungkan ke OpenSSH Backend
	sshConn, err := net.DialTimeout("tcp", sshTarget, 5*time.Second)
	if err != nil {
		return
	}
	tweakSocket(sshConn)
	defer sshConn.Close()

	done := make(chan struct{}, 2)

	// --- FIX FILTER: SLIDING BUFFER ANTI "SSH" TERPOTONG ---
	go func() {
		defer func() { done <- struct{}{} }()
		buffer := make([]byte, BufferSize)
		filtering := true
		
		// Kita simpan sisaan byte dari putaran sebelumnya ke sini
		var leftOver []byte 
		var totalRead int

		for {
			n, err := client.Read(buffer)
			if n > 0 {
				data := buffer[:n]
				totalRead += n

				if filtering {
					// Gabungkan sisaan putaran lalu dengan data baru
					combined := append(leftOver, data...)
					
					if idx := bytes.Index(combined, []byte("SSH-")); idx != -1 {
						// Banner SSH Ketemu! Buang semua sampah di kirinya (idx)
						cleanData := combined[idx:]
						
						_, wErr := sshConn.Write(cleanData)
						if wErr != nil {
							return
						}
						
						filtering = false // Matikan filter
						leftOver = nil    // Kosongkan memori leftover
						
					} else if totalRead > 4096 {
						// Jika sudah baca > 4KB dan banner nggak ada, anggap ini speedtest atau direct mode
						_, wErr := sshConn.Write(combined)
						if wErr != nil {
							return
						}
						filtering = false
						leftOver = nil
					} else {
						// Banner belum ketemu, tapi kita harus waspada siapa tau kata "SSH-" terpotong 
						// (misal cuma "SS" di akhir combined). 
						// Kita simpan 3 byte terakhir dari gabungan saat ini ke putaran selanjutnya.
						if len(combined) > 3 {
							leftOver = combined[len(combined)-3:]
						} else {
							leftOver = combined
						}
					}
				} else {
					// Jika sudah nggak filtering (Mode murni)
					_, wErr := sshConn.Write(data)
					if wErr != nil {
						return
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Pipe arah sebaliknya (SSH/Dropbear -> Client) - Gunakan io.Copy untuk efisiensi RAM penuh
	go func() {
		defer func() { done <- struct{}{} }()
		_, _ = io.Copy(client, sshConn)
	}()

	<-done
}
