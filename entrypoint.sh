#!/bin/bash

# 🔥 KUNCI UTAMA ANTI REKONEK: Buka paksa limit socket & stack size Alpine Linux
ulimit -n 65535
ulimit -s unlimited

# =================================================================
# 🚀 ULTRA TURBO KERNEL v3.2 (PURE STANDARD FOR GOLANG + OPENSSH) 🚀
# =================================================================
echo "[*] Mengaktifkan TCP BBR dan Fair Queuing..."
sysctl -w net.core.default_qdisc=fq 2>/dev/null
sysctl -w net.ipv4.tcp_congestion_control=bbr 2>/dev/null

echo "[*] Mengoptimalkan ukuran buffer TCP Kernel (BUFFER RAKSASA)..."
sysctl -w net.ipv4.tcp_rmem="4096 8388608 16777216" 2>/dev/null
sysctl -w net.ipv4.tcp_wmem="4096 8388608 16777216" 2>/dev/null
sysctl -w net.core.rmem_max=16777216 2>/dev/null
sysctl -w net.core.wmem_max=16777216 2>/dev/null

# Kelonggaran antrean kartu jaringan agar engine Go-routine melesat lempeng
sysctl -w net.core.netdev_max_backlog=50000 2>/dev/null
sysctl -w net.ipv4.tcp_max_syn_backlog=8192 2>/dev/null

USER_NAME="${SSH_USER:-dd}"
USER_PASS="${SSH_PASSWORD:-dd}"
PUBLIC_PORT="${PORT:-8080}"
SSL_INTERNAL_PORT="${SSL_INTERNAL_PORT:-2443}"
WS_INTERNAL_PORT="8880"
UI_PORT="8081"
LOG_CF="/tmp/cloudflared.log"
LOG_NAMED="/tmp/named_tunnel.log"
STATS_JSON="/tmp/server_stats.json"

echo "[*] Membuat sertifikat SSL Stunnel dinamis..."
mkdir -p /etc/stunnel /var/run/stunnel
openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 \
    -subj "/C=ID/ST=Jakarta/L=Jakarta/O=RailwaySSH/CN=localhost" \
    -keyout /etc/stunnel/stunnel.pem -out /etc/stunnel/stunnel.pem

chown -R stunnel:stunnel /etc/stunnel /var/run/stunnel
chmod 600 /etc/stunnel/stunnel.pem

echo "[*] Mengonfigurasi User SSH (Alpine Mode)..."
if ! id "$USER_NAME" &>/dev/null; then
    adduser -D -s /bin/bash "$USER_NAME"
    echo "$USER_NAME ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
fi
echo "$USER_NAME:$USER_PASS" | chpasswd

echo "[*] Membuat Banner Rapi untuk OpenSSH..."
cat << 'EOF' > /etc/ssh/ssh_banner
==================================================
              👑 SELAMAT MENIKMATI 👑  
              <br>
              SSH SERVER RAILWAY MOD              
==================================================
 SPESIFIKASI:  
 <br>
 🔹 MULTIPLEXER : GOLANG HIGH-SPEED CORE v3.2  
 <br>
 🔹 OS PLATFORM : LINUX ALPINE (RAM MONSTER MODE)  
 <br>
 🔹 SSH SERVICE : OPENSSH SERVER HIGH COMPAT      
==================================================
          powered by : d e d e f a t h u          
==================================================
EOF

echo "[*] Menyiapkan Host Keys untuk OpenSSH..."
ssh-keygen -A

echo "[*] Membuat konfigurasi OpenSSH Suci Murni (Anti-Rekonek Version)..."
cat << 'EOF' > /etc/ssh/sshd_config
Port 22
ListenAddress 127.0.0.1
PermitRootLogin yes
PasswordAuthentication yes
PermitEmptyPasswords no
ChallengeResponseAuthentication no
UsePAM no
X11Forwarding no
PrintMotd no
AcceptEnv LANG LC_*
Subsystem sftp /usr/lib/ssh/sftp-server
Banner /etc/ssh/ssh_banner

# 🛠 KUNCI UTAMA ANTI TIMEOUT:
UseDNS no
ClientAliveInterval 20
ClientAliveCountMax 3
EOF

echo "[*] Memulai OpenSSH Server di Port Lokal 22..."
/usr/sbin/sshd

echo "[*] Membuat konfigurasi Stunnel..."
cat <<EOF > /etc/stunnel/stunnel.conf
pid = /var/run/stunnel.pid
foreground = yes
debug = 4
setuid = stunnel
setgid = stunnel

[ssh-ssl]
accept = 127.0.0.1:$SSL_INTERNAL_PORT
connect = 127.0.0.1:22
cert = /etc/stunnel/stunnel.pem
EOF

echo "[*] Menambahkan sesuatu di .bashrc..."
cat <<'EOF'>> /etc/bash.bashrc
clear
alias c='clear'
alias x='exit'
alias cls='clear;ls'
menu
EOF
echo "source /etc/bash.bashrc" >> /home/"$USER_NAME"/.bashrc

echo "[*] Memulai Stunnel..."
stunnel /etc/stunnel/stunnel.conf &

# --- 🔥 PUSAT EKSEKUSI DOUBLE TUNNEL FIXED 🔥 ---

# 1. Jalankan Named Tunnel HANYA JIKA token diisi di Railway
if [ -n "$CF_TUNNEL_TOKEN" ]; then
    echo "[*] Menjalankan Cloudflare Named Tunnel (Mode Dinamis via Dashboard)..."
    cloudflared tunnel run --protocol http2 --token "$CF_TUNNEL_TOKEN" > $LOG_NAMED 2>&1 &
fi

# 2. 🔥 PERBAIKAN UTAMA: Mengubah http:// menjadi tcp:// agar SSL/SNI lolos bersih ke Muxer Go
echo "[*] Menjalankan Cloudflare Quick Tunnel (Link Acak TCP Mode)..."
cloudflared tunnel --url "tcp://127.0.0.1:$PUBLIC_PORT" --protocol http2 > $LOG_CF 2>&1 &

# =================================================================

# --- BADVPN UDPGW UNTUK MENDUKUNG TRAFIK UDP / GAME ---
if [ -f /usr/local/bin/badvpn-udpgw ]; then
    echo "[*] Memulai BadVPN udpgw di Port Lokal 7300..."
    /usr/local/bin/badvpn-udpgw --listen-addr 127.0.0.1:7300 --max-clients 500 --max-connections-for-client 20 &
else
    echo "[!] Binary badvpn-udpgw tidak ditemukan!"
fi

echo "[*] Memulai WS-Proxy Engine internal..."
export WS_PORT="$WS_INTERNAL_PORT"
ws-proxy &

# =================================================================
# 📊 BACKGROUND STATS COLLECTOR FOR HARDWARE & USERS 📊
# =================================================================
(
    while true; do
        CPU_MODEL=$(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2 | sed 's/^[ \t]*//')
        [ -z "$CPU_MODEL" ] && CPU_MODEL="Virtual Core (Railway)"
        CPU_CORES=$(grep -c 'processor' /proc/cpuinfo)
        
        RAM_TOTAL=$(free -m | awk '/Mem:/ {print $2}')
        RAM_USED=$(free -m | awk '/Mem:/ {print $3}')
        
        DISK_USAGE=$(df -h / | awk 'NR==2 {print $5}')
        UPTIME=$(uptime | awk -F'(,)|(up)' '{print $2}' | sed 's/^[ \t]*//')
        [ -z "$UPTIME" ] && UPTIME=$(uptime | awk '{print $3}')
        
        SSH_ONLINE=$(netstat -anp 2>/dev/null | grep :22 | grep ESTABLISHED | wc -l)
        [ -z "$SSH_ONLINE" ] && SSH_ONLINE="0"

        cat <<EOF > "$STATS_JSON"
{
  "cpu_model": "$CPU_MODEL ($CPU_CORES Cores)",
  "ram_total": "${RAM_TOTAL} MB",
  "ram_used": "${RAM_USED} MB",
  "disk_usage": "$DISK_USAGE",
  "uptime": "$UPTIME",
  "ssh_online": "$SSH_ONLINE",
  "custom_domain": "${MY_DOMAIN:-}"
}
EOF
        sleep 2
    done
) &

# --- SIAPKAN FOLDER & JALANKAN WEB UI DASHBOARD ---
echo "[*] Menyiapkan lingkungan Web UI..."
mkdir -p /app

echo "[*] Memulai Web UI Dashboard di Port $UI_PORT..."
python3 /app/index.py &

# =================================================================

echo "[*] Memulai Front Muxer Engine Utama (Golang Mode)..."
export PORT="$PUBLIC_PORT"
export SSL_TARGET_HOST="127.0.0.1"
export SSL_TARGET_PORT="$SSL_INTERNAL_PORT"
export WS_MUX_TARGET_HOST="127.0.0.1"
export WS_MUX_TARGET_PORT="$WS_INTERNAL_PORT"

# Jalur Target Pipa Tambahan untuk BadVPN Game ke Muxer Baru
export UDPGW_TARGET_HOST="127.0.0.1"
export UDPGW_TARGET_PORT="7300"

exec mux
