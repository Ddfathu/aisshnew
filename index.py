import http.server
import socketserver
import re
import os
import json
import subprocess
import urllib.parse

PORT = 8081
LOG_PATH = "/tmp/cloudflared.log"
STATS_PATH = "/tmp/server_stats.json"

class DashboardHandler(http.server.SimpleHTTPRequestHandler):
    # Log error dibikin pasif biar gak mengebiri server pas ada request masuk
    def log_message(self, format, *args):
        return

    # 🛠️ FITUR MANAGEMENT SSH (ALPINE MODE) 🛠️
    def list_ssh(self):
        try:
            users = []
            with open("/etc/passwd", "r") as f:
                for line in f:
                    parts = line.strip().split(":")
                    username = parts[0]
                    uid = int(parts[2])
                    shell = parts[-1]
                    
                    # Filter user biasa (UID >= 1000 dan bukan user sistem bawaan)
                    if uid >= 1000 and username not in ["nobody", "alpine"]:
                        users.append({"username": username, "uid": uid, "shell": shell})
            return {"status": "success", "total": len(users), "users": users}
        except Exception as e:
            return {"status": "error", "message": str(e)}

    def add_ssh(self, username, password):
        if not username or not password:
            return {"status": "error", "message": "Username dan password wajib diisi!"}
        try:
            # Perintah adduser khas Alpine Linux (-D tanpa interaktif)
            cmd_user = f"adduser -D -s /bin/bash {username}"
            subprocess.run(cmd_user, shell=True, check=True)
            
            # Suntik password ke user baru
            cmd_pass = f"echo '{username}:{password}' | chpasswd"
            subprocess.run(cmd_pass, shell=True, check=True)
            return {"status": "success", "message": f"User {username} berhasil dibuat!"}
        except subprocess.CalledProcessError:
            return {"status": "error", "message": f"Gagal membuat user. Username '{username}' mungkin sudah ada."}
        except Exception as e:
            return {"status": "error", "message": str(e)}

    def delete_ssh(self, username):
        if not username:
            return {"status": "error", "message": "Username wajib diisi!"}
        try:
            # Hapus user di Alpine
            cmd_del = f"deluser {username}"
            subprocess.run(cmd_del, shell=True, check=True)
            
            # Bersihkan sisa folder home
            subprocess.run(f"rm -rf /home/{username}", shell=True)
            return {"status": "success", "message": f"User {username} berhasil dihapus!"}
        except subprocess.CalledProcessError:
            return {"status": "error", "message": f"Gagal menghapus user. User '{username}' tidak ditemukan."}
        except Exception as e:
            return {"status": "error", "message": str(e)}

    def do_GET(self):
        parsed_url = urllib.parse.urlparse(self.path)
        path = parsed_url.path
        query = urllib.parse.parse_qs(parsed_url.query)

        # ==========================================
        # 🟢 ROUTER 1: API MANAGEMENT SSH
        # ==========================================
        if path in ["/api/list", "/api/add", "/api/delete"]:
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            
            response_data = {"status": "error", "message": "Aksi tidak dikenal"}
            
            if path == "/api/list":
                response_data = self.list_ssh()
            elif path == "/api/add":
                username = query.get("user", [None])[0]
                password = query.get("pass", [None])[0]
                response_data = self.add_ssh(username, password)
            elif path == "/api/delete":
                username = query.get("user", [None])[0]
                response_data = self.delete_ssh(username)
                
            try:
                self.wfile.write(json.dumps(response_data).encode('utf-8'))
            except Exception:
                pass
            return

        # ==========================================
        # 🟢 ROUTER 2: API LIVE MONITOR HARDWARE
        # ==========================================
        if path == "/api/stats":
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            
            quick_url = "Menunggu Quick Tunnel siap..."
            status = "ONLINE"
            
            hw_info = {"cpu_model": "Loading...", "ram_total": "0", "ram_used": "0", "disk_usage": "0%", "uptime": "0", "ssh_online": "0", "custom_domain": ""}
            if os.path.exists(STATS_PATH):
                try:
                    with open(STATS_PATH, "r") as f:
                        hw_info = json.load(f)
                except Exception:
                    pass
            
            if os.path.exists(LOG_PATH):
                try:
                    with open(LOG_PATH, "r") as f:
                        match = re.search(r'https://[a-zA-Z0-9-]+\.trycloudflare\.com', f.read())
                        if match:
                            quick_url = match.group(0)
                except Exception:
                    pass
            
            named_url = "Tidak Aktif (Token Kosong)"
            if hw_info.get("custom_domain"):
                named_url = "https://" + hw_info["custom_domain"].replace("https://", "").replace("http://", "")

            response_data = {
                "quick_url": quick_url,
                "named_url": named_url,
                "status": status,
                **hw_info
            }
            try:
                self.wfile.write(json.dumps(response_data).encode('utf-8'))
            except Exception:
                pass
            return

        # ==========================================
        # 🟢 ROUTER 3: TAMPILAN DASHBOARD HTML UTAMA
        # ==========================================
        self.send_response(200)
        self.send_header("Content-type", "text/html; charset=utf-8")
        self.end_headers()
        
        html = """
        <!DOCTYPE html>
        <html lang="id">
        <head>
            <meta charset="UTF-8">
            <title>⚡ PREMIUM SSH RAILWAY PANEL ⚡</title>
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <style>
                * { box-sizing: border-box; margin: 0; padding: 0; }
                body { 
                    font-family: '-apple-system', BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
                    background: #090d16; color: #f8fafc; display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 15px;
                }
                .container { 
                    background: #111827; width: 100%; max-width: 500px; padding: 25px; border-radius: 16px; box-shadow: 0 10px 30px -5px rgba(0, 0, 0, 0.8); border: 1px solid #1f2937;
                }
                .header { text-align: center; margin-bottom: 20px; }
                h1 { font-size: 20px; color: #38bdf8; text-transform: uppercase; letter-spacing: 1px; }
                .dev-tag { font-size: 11px; color: #64748b; margin-top: 4px; font-weight: bold; }
                
                .status-container { text-align: center; margin-bottom: 15px; }
                .status-badge { display: inline-block; background: #1f2937; padding: 5px 12px; border-radius: 50px; font-size: 11px; font-weight: bold; border: 1px solid #334155; }
                .status-dot { height: 8px; width: 8px; background-color: #4ade80; border-radius: 50%; display: inline-block; margin-right: 6px; box-shadow: 0 0 8px #4ade80; }

                .stats-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 20px; }
                .stat-card { background: #1f2937; padding: 12px; border-radius: 8px; border: 1px solid #334155; text-align: left; }
                .stat-title { font-size: 11px; color: #94a3b8; text-transform: uppercase; }
                .stat-value { font-size: 14px; font-weight: bold; color: #f1f5f9; margin-top: 4px; }

                .url-section { background: #030712; border: 1px solid #38bdf8; padding: 12px; border-radius: 8px; margin-bottom: 12px; text-align: center; }
                .url-title { font-size: 11px; color: #94a3b8; font-weight: bold; text-transform: uppercase; }
                .url-box { font-family: monospace; font-size: 13px; word-break: break-all; color: #38bdf8; font-weight: bold; margin: 6px 0; }
                
                .btn-copy { 
                    background: #38bdf8; color: #090d16; border: none; padding: 6px 12px; border-radius: 6px; font-weight: bold; cursor: pointer; font-size: 11px; width: 100%; transition: all 0.2s;
                }
                .btn-copy:active { transform: scale(0.98); }
                .note { font-size: 11px; color: #64748b; text-align: center; line-height: 1.4; margin-top: 10px; }
            </style>
        </head>
        <body>
            <div class="container">
                <div class="header">
                    <h1>👑 DDFATHU DOUBLE MONITOR 👑</h1>
                    <div class="dev-tag">DYNAMIC DUAL-TUNNEL CORE ACTIVE</div>
                </div>
                
                <div class="status-container">
                    <div class="status-badge">
                        <span class="status-dot"></span>
                        <span style="color: #4ade80">TUNNELS ONLINE</span>
                    </div>
                </div>

                <div class="stats-grid">
                    <div class="stat-card" style="grid-column: span 2;">
                        <div class="stat-title">CPU Model</div>
                        <div class="stat-value" id="cpu" style="font-size:12px; color:#38bdf8;">Loading...</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-title">RAM Used / Total</div>
                        <div class="stat-value" id="ram">Loading...</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-title">Disk Usage (/)</div>
                        <div class="stat-value" id="disk">Loading...</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-title">Server Uptime</div>
                        <div class="stat-value" id="uptime" style="font-size:12px;">Loading...</div>
                    </div>
                    <div class="stat-card" style="border-color: #a855f7;">
                        <div class="stat-title" style="color:#d8b4fe;">SSH Online Users</div>
                        <div class="stat-value" id="ssh" style="font-size:18px; color:#a855f7;">👥 0 Users</div>
                    </div>
                </div>
                
                <div class="url-section" style="border-color: #a855f7;">
                    <div class="url-title" style="color: #d8b4fe;">1. Named Tunnel (Domain Utama)</div>
                    <div class="url-box" id="named-url" style="color: #d8b4fe;">Loading...</div>
                    <button class="btn-copy" id="btn-copy-named" style="background:#a855f7; color:#fff;" onclick="copyTxt('named-url', 'btn-copy-named')">📋 COPY DOMAIN UTAMA</button>
                </div>

                <div class="url-section">
                    <div class="url-title">2. Quick Tunnel (Link Acak Bumper Worker)</div>
                    <div class="url-box" id="quick-url">Loading...</div>
                    <button class="btn-copy" id="btn-copy-quick" onclick="copyTxt('quick-url', 'btn-copy-quick')">📋 COPY LINK ACAK WORKER</button>
                </div>
                
                <p class="note">Dua tunnel berjalan beriringan tanpa bentrok.<br>Salin link acak di atas ke dalam bumper Worker lu.</p>
            </div>

            <script>
                async function updateStats() {
                    try {
                        let res = await fetch('/api/stats');
                        let data = await res.json();
                        
                        document.getElementById('cpu').innerText = data.cpu_model;
                        document.getElementById('ram').innerText = data.ram_used + " / " + data.ram_total;
                        document.getElementById('disk').innerText = data.disk_usage;
                        document.getElementById('uptime').innerText = data.uptime;
                        document.getElementById('ssh').innerText = "👥 " + data.ssh_online + " Users";
                        document.getElementById('named-url').innerText = data.named_url;
                        document.getElementById('quick-url').innerText = data.quick_url;
                    } catch(e) { console.log(e); }
                }

                function copyTxt(elementId, btnId) {
                    let urlText = document.getElementById(elementId).innerText;
                    if(!urlText.includes("Menunggu") && !urlText.includes("Tidak Aktif")) {
                        navigator.clipboard.writeText(urlText);
                        let btn = document.getElementById(btnId);
                        let oldText = btn.innerText;
                        btn.innerText = "✅ COPIED!";
                        btn.style.background = "#4ade80";
                        btn.style.color = "#090d16";
                        setTimeout(() => {
                            btn.innerText = oldText;
                            btn.style.background = elementId === 'named-url' ? '#a855f7' : '#38bdf8';
                            btn.style.color = elementId === 'named-url' ? '#fff' : '#090d16';
                        }, 1500);
                    }
                }

                setInterval(updateStats, 2000);
                updateStats();
            </script>
        </body>
        </html>
        """
        try:
            self.wfile.write(html.encode('utf-8'))
        except Exception:
            pass

if __name__ == "__main__":
    # Mengizinkan reuse port agar tidak macet saat kontainer restart lambat
    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.TCPServer(("0.0.0.0", PORT), DashboardHandler) as httpd:
        print(f"Dual Tunnel Panel UI running at port {PORT}")
        httpd.serve_forever()
