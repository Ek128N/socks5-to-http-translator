

# SOCKS5-to-HTTP Proxy Bridge

 Windows-native proxy bridge that accepts **SOCKS5** connections and forwards them through an upstream **HTTP proxy** using the HTTP CONNECT method.
 Run as a console application or Windows service with YAML configuration.

---

## 🤌 Overview

This tool acts as a protocol bridge, translating SOCKS5 client traffic into HTTP proxy requests. 
Once established, data flows as a tunnel between client and destination via the HTTP proxy.


```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SOCKS5 Client │◄──►│  Proxy Bridge   │◄──►│   HTTP Proxy    │◄──►│   Destination   │
│  (e.g. Browser) │    │     (Tool)      │    │ (Corporate/ISP) │    │     Server      │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
   SOCKS5 Protocol     Protocol Translation       HTTP CONNECT           HTTP/HTTPS
```
🤔 It is actually more of a protocol translator than a true bridge or tunnel.

## 📦 Features

* **SOCKS5** client support (no authentication).
* Uses **HTTP proxy** with **CONNECT** method for upstream forwarding.
* Supports optional **HTTP Basic Authentication** to upstream proxy.
* **TCP traffic only** (⚠️UDP not supported).
* Works as a Windows service or console app.
* Configuration via YAML.

## ⚙️ Configuration

`config.yaml` (placed alongside `proxy.exe` after build):

```yaml
listen_address: "127.0.0.1:1080"

http_proxy:
  host: "proxy.server.com"
  port: 3128
  username: "user"
  password: "pass"

timeouts:
  dial: 10
  idle: 60
```

* **listen\_address** — where the proxy bridge listens for SOCKS5 clients.
* **http\_proxy** — upstream HTTP proxy settings.
* **timeouts** — in seconds.

---

## 🎰 Usage

Use via the **service.bat**  (must be run as 🥸 **Administrator**):

```bash
scripts\service.bat
```
* Once installed as service, it auto-starts with Windows.
* Set actions like “Restart service” on failures.
---

## 🪟 Windows Service Management

* View service status:

  ```powershell
  sc query SOCKSHTTPBridge
  ```

* View logs:

   * Service writes logs to the Windows Event Viewer or use console mode for real-time logs.

* Set recovery options:

   * Open **services.msc**, locate `SOCKSHTTPBridge`, right-click → Properties → Recovery tab.
   * Set actions like “Restart service” on failures.

---
## ⛑️ Building

From project root:

```bash
scripts\build-prod.bat
```

This will:

* Compile `proxy.exe` into `bin/`.
* Copy the latest `config.yaml` to `bin/`.

---

## 🛠 Debugging

* Use **console mode** (`choice 1` in service.bat) to see connection logs.
* Ensure `config.yaml` is in the same folder as `proxy.exe`.
* Ensure the service runs with correct permissions (Administrator).

---
## How I'm using it
I'm using this tool as a workaround for YouTube Music Desktop App, which only supports SOCKS5 proxies and doesn’t handle HTTP proxies.
This tool acts as a protocol translator, allowing the app to work with an authenticated HTTP proxy by converting SOCKS5 traffic to HTTP.

⚠️ Note: This tool only supports TCP traffic. It’s suitable for streaming music or similar use cases, but downloading videos or handling UDP-based protocols is not supported.

---
## 📃 TODO
- [ ] 📦 Setup zip distribution
- [ ] 🔌 Support for more protocols (UDP relay / TCP wrapping?)
