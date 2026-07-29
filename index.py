import http.server
import socketserver
import re
import os
import json

PORT = 8081
LOG_PATH = "/tmp/cloudflared.log"
STATS_PATH = "/tmp/server_stats.json"

class DashboardHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        # Fitur API mini untuk update data hardware tanpa refresh halaman
        if self.path == "/api/stats":
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            
            tunnel_url = "Menunggu Argo Tunnel siap..."
            status = "OFFLINE"
            if os.path.exists(LOG_PATH):
                with open(LOG_PATH, "r") as f:
                    match = re.search(r'https://[a-zA-Z0-9-]+\.trycloudflare\.com', f.read())
                    if match:
                        tunnel_url = match.group(0)
                        status = "ONLINE"
                        
            hw_info = {"cpu_model": "Loading...", "ram_total": "0", "ram_used": "0", "disk_usage": "0%", "uptime": "0", "ssh_online": "0"}
            if os.path.exists(STATS_PATH):
                try:
                    with open(STATS_PATH, "r") as f:
                        hw_info = json.load(f)
                except Exception:
                    pass
            
            response_data = {
                "tunnel_url": tunnel_url,
                "status": status,
                **hw_info
            }
            self.wfile.write(json.dumps(response_data).encode('utf-8'))
            return

        # Halaman Utama UI
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
                .status-dot { height: 8px; width: 8px; background-color: #ef4444; border-radius: 50%; display: inline-block; margin-right: 6px; box-shadow: 0 0 8px #ef4444; }

                .stats-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 20px; }
                .stat-card { background: #1f2937; padding: 12px; border-radius: 8px; border: 1px solid #334155; text-align: left; }
                .stat-title { font-size: 11px; color: #94a3b8; text-transform: uppercase; }
                .stat-value { font-size: 14px; font-weight: bold; color: #f1f5f9; margin-top: 4px; }

                .url-section { background: #030712; border: 1px solid #38bdf8; padding: 15px; border-radius: 8px; margin-bottom: 15px; text-align: center; }
                .url-box { font-family: monospace; font-size: 14px; word-break: break-all; color: #38bdf8; font-weight: bold; margin: 8px 0; }
                
                .btn-copy { 
                    background: #38bdf8; color: #090d16; border: none; padding: 8px 16px; border-radius: 6px; font-weight: bold; cursor: pointer; font-size: 12px; width: 100%; transition: all 0.2s;
                }
                .btn-copy:active { transform: scale(0.98); }
                .note { font-size: 11px; color: #64748b; text-align: center; line-height: 1.4; }
            </style>
        </head>
        <body>
            <div class="container">
                <div class="header">
                    <h1>👑 DDFATHU TUNNEL MONITOR 👑</h1>
                    <div class="dev-tag">CORE MULTIPLEXER v3.2 ACTIVE</div>
                </div>
                
                <div class="status-container">
                    <div class="status-badge">
                        <span class="status-dot" id="dot"></span>
                        <span id="status-text" style="color: #ef4444">OFFLINE</span>
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
                
                <div class="url-section">
                    <div style="font-size:12px; color:#94a3b8; font-weight:500;">Copy Quick Tunnel URL ke Worker:</div>
                    <div class="url-box" id="tunnel-url">Loading...</div>
                    <button class="btn-copy" id="copy-btn" onclick="copyUrl()">📋 COPY URL</button>
                </div>
                
                <p class="note">Data hardware terupdate otomatis di latar belakang.<br>Halaman tidak akan me-refresh total sehingga aman di-copy.</p>
            </div>

            <script>
                async function updateStats() {
                    try {
                        let res = await fetch('/api/stats');
                        let data = await res.json();
                        
                        // Update status tunnel
                        let dot = document.getElementById('dot');
                        let txt = document.getElementById('status-text');
                        if(data.status === "ONLINE") {
                            dot.style.backgroundColor = "#4ade80";
                            dot.style.boxShadow = "0 0 8px #4ade80";
                            txt.style.color = "#4ade80";
                            txt.innerText = "TUNNEL ONLINE";
                        } else {
                            dot.style.backgroundColor = "#ef4444";
                            dot.style.boxShadow = "0 0 8px #ef4444";
                            txt.style.color = "#ef4444";
                            txt.innerText = "OFFLINE";
                        }
                        
                        // Update teks hardware
                        document.getElementById('cpu').innerText = data.cpu_model;
                        document.getElementById('ram').innerText = data.ram_used + " / " + data.ram_total;
                        document.getElementById('disk').innerText = data.disk_usage;
                        document.getElementById('uptime').innerText = data.uptime;
                        document.getElementById('ssh').innerText = "👥 " + data.ssh_online + " Users";
                        document.getElementById('tunnel-url').innerText = data.tunnel_url;
                    } catch(e) { console.log(e); }
                }

                function copyUrl() {
                    let urlText = document.getElementById('tunnel-url').innerText;
                    if(urlText.includes("trycloudflare.com")) {
                        navigator.clipboard.writeText(urlText);
                        let btn = document.getElementById('copy-btn');
                        btn.innerText = "✅ COPIED!";
                        btn.style.background = "#4ade80";
                        setTimeout(() => {
                            btn.innerText = "📋 COPY URL";
                            btn.style.background = "#38bdf8";
                        }, 2000);
                    } else {
                        alert("Tunnel belum siap atau link tidak valid!");
                    }
                }

                // Jalankan update berkala tiap 2 detik tanpa kedip
                setInterval(updateStats, 2000);
                updateStats();
            </script>
        </body>
        </html>
        """
        self.wfile.write(html.encode('utf-8'))

if __name__ == "__main__":
    with socketserver.TCPServer(("0.0.0.0", PORT), DashboardHandler) as httpd:
        print(f"Premium Panel UI running at port {PORT}")
        httpd.serve_forever()
