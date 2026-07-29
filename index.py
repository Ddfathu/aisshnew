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
        self.send_response(200)
        self.send_header("Content-type", "text/html; charset=utf-8")
        self.end_headers()
        
        # 1. Ambil URL Argo Tunnel
        tunnel_url = "Menunggu Argo Tunnel siap..."
        status_color = "#ef4444"
        status_text = "OFFLINE / CONNECTING"
        
        if os.path.exists(LOG_PATH):
            with open(LOG_PATH, "r") as f:
                log_content = f.read()
                match = re.search(r'https://[a-zA-Z0-9-]+\.trycloudflare\.com', log_content)
                if match:
                    tunnel_url = match.group(0)
                    status_color = "#4ade80"
                    status_text = "TUNNEL ONLINE"

        # 2. Ambil Info Hardware & SSH dari JSON
        hw_info = {
            "cpu_model": "Loading...",
            "ram_total": "Loading...",
            "ram_used": "Loading...",
            "disk_usage": "Loading...",
            "uptime": "Loading...",
            "ssh_online": "0"
        }
        
        if os.path.exists(STATS_PATH):
            try:
                with open(STATS_PATH, "r") as f:
                    hw_info = json.load(f)
            except Exception:
                pass

        # Template HTML UI Premium Dashboard Monitor
        html = f"""
        <!DOCTYPE html>
        <html lang="id">
        <head>
            <meta charset="UTF-8">
            <title>⚡ PREMIUM SSH RAILWAY PANEL ⚡</title>
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <meta http-equiv="refresh" content="2">
            <style>
                * {{ box-sizing: border-box; margin: 0; padding: 0; }}
                body {{ 
                    font-family: '-apple-system', BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
                    background: #090d16; 
                    color: #f8fafc; 
                    display: flex; 
                    justify-content: center; 
                    align-items: center; 
                    min-height: 100vh;
                    padding: 15px;
                }}
                .container {{ 
                    background: #111827; 
                    width: 100%;
                    max-width: 500px; 
                    padding: 25px; 
                    border-radius: 16px; 
                    box-shadow: 0 10px 30px -5px rgba(0, 0, 0, 0.8); 
                    border: 1px solid #1f2937;
                }}
                .header {{ text-align: center; margin-bottom: 20px; }}
                h1 {{ font-size: 20px; color: #38bdf8; text-transform: uppercase; letter-spacing: 1px; }}
                .dev-tag {{ font-size: 11px; color: #64748b; margin-top: 4px; font-weight: bold; }}
                
                .status-container {{ text-align: center; margin-bottom: 15px; }}
                .status-badge {{ 
                    display: inline-block; background: #1f2937; padding: 5px 12px; 
                    border-radius: 50px; font-size: 11px; font-weight: bold; border: 1px solid #334155;
                }}
                .status-dot {{
                    height: 8px; width: 8px; background-color: {status_color};
                    border-radius: 50%; display: inline-block; margin-right: 6px;
                    box-shadow: 0 0 8px {status_color};
                }}

                .stats-grid {{ 
                    display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 20px;
                }}
                .stat-card {{ 
                    background: #1f2937; padding: 12px; border-radius: 8px; 
                    border: 1px solid #334155; text-align: left;
                }}
                .stat-title {{ font-size: 11px; color: #94a3b8; text-transform: uppercase; }}
                .stat-value {{ font-size: 14px; font-weight: bold; color: #f1f5f9; margin-top: 4px; }}
                .stat-value.highlight {{ color: #a855f7; }} /* Warna ungu untuk user online */

                .url-section {{ background: #030712; border: 1px solid #38bdf8; padding: 15px; border-radius: 8px; margin-bottom: 15px; text-align: center; }}
                .url-box {{ font-family: monospace; font-size: 14px; word-break: break-all; color: #38bdf8; font-weight: bold; margin-top: 5px; }}
                
                .note {{ font-size: 11px; color: #64748b; text-align: center; line-height: 1.4; }}
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
                        <span class="status-dot"></span>
                        <span style="color: {status_color}">{status_text}</span>
                    </div>
                </div>

                <!-- PANEL HARDWARE & MONITORING USER -->
                <div class="stats-grid">
                    <div class="stat-card" style="grid-column: span 2;">
                        <div class="stat-title">CPU Model</div>
                        <div class="stat-value" style="font-size:12px; color:#38bdf8;">{hw_info['cpu_model']}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-title">RAM Used / Total</div>
                        <div class="stat-value">{hw_info['ram_used']} / {hw_info['ram_total']}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-title">Disk Usage (/)</div>
                        <div class="stat-value">{hw_info['disk_usage']}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-title">Server Uptime</div>
                        <div class="stat-value" style="font-size:12px;">{hw_info['uptime']}</div>
                    </div>
                    <div class="stat-card" style="border-color: #a855f7;">
                        <div class="stat-title" style="color:#d8b4fe;">SSH Online Users</div>
                        <div class="stat-value highlight" style="font-size:18px;">👥 {hw_info['ssh_online']} Users</div>
                    </div>
                </div>
                
                <div class="url-section">
                    <div style="font-size:12px; color:#94a3b8; font-weight:500;">Copy Quick Tunnel URL ke Worker:</div>
                    <div class="url-box">{tunnel_url}</div>
                </div>
                
                <p class="note">Panel otomatis refresh real-time setiap 2 detik.<br>Mendeteksi spesifikasi hardware & statistik user secara langsung.</p>
            </div>
        </body>
        </html>
        """
        self.wfile.write(html.encode('utf-8'))

if __name__ == "__main__":
    with socketserver.TCPServer(("0.0.0.0", PORT), DashboardHandler) as httpd:
        print(f"Premium Panel UI running at port {PORT}")
        httpd.serve_forever()
