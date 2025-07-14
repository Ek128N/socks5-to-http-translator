package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/sys/windows/svc"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	ListenAddress string `yaml:"listen_address"`

	HTTPProxy struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"http_proxy"`

	Timeouts struct {
		Dial int `yaml:"dial"`
		Idle int `yaml:"idle"`
	} `yaml:"timeouts"`
}

var (
	cfg        Config
	bufferPool = sync.Pool{New: func() interface{} { return make([]byte, 32*1024) }}
	listener   net.Listener
	stopSignal = make(chan struct{})
)

func main() {

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			runCmd(`sc create SOCKSHTTPBridge binPath= "%~dp0proxy.exe" start= auto`)
			return
		case "uninstall":
			runCmd("sc delete SOCKSHTTPBridge")
			return
		case "console":
			startProxy()
			return
		default:
			fmt.Println("Usage: proxy.exe [console|install|uninstall]")
			return
		}
	}

	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine service mode: %v", err)
	}

	if isService {
		log.Println("Running as Windows service")
		err = svc.Run("SOCKSHTTPBridge", &proxyService{})
		if err != nil {
			log.Fatalf("service failed: %v", err)
		}
		return
	}

	// Console mode
	log.Println("Running in console mode")
	startProxy()
}

func startProxy() {
	loadConfig()

	listener, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("SOCKS5 -> HTTP proxy bridge listening on %s", cfg.ListenAddress)

	for {
		select {
		case <-stopSignal:
			log.Println("Shutting down...")
			err := listener.Close()
			if err != nil {
				log.Printf("Failed to close listener: %v", err)
				return
			}
			return
		default:
			err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			if err != nil {
				log.Printf("Failed to set deadline: %v", err)
				return
			}
			conn, err := listener.Accept()
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					continue
				}
				log.Printf("Accept error: %v", err)
				continue
			}
			go handleConn(conn)
		}
	}
}

func loadConfig() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.yaml")

	f, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("Error opening config.yaml: %v", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}
}

func stopProxy() {
	close(stopSignal)
}

func handleConn(conn net.Conn) {
	err := conn.SetDeadline(time.Now().Add(time.Duration(cfg.Timeouts.Idle) * time.Second))
	if err != nil {
		log.Printf("Failed to set deadline: %v", err)
		return
	}
	br := bufio.NewReader(conn)

	//SOCKS5 Handshake
	header := make([]byte, 2)
	if _, err := io.ReadFull(br, header); err != nil {
		return
	}
	nMethods := int(header[1])
	if nMethods > 16 {
		log.Printf("Too many auth methods: %d", nMethods)
		return
	}
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(br, methods); err != nil {
		return
	}

	if !bytes.Contains(methods, []byte{0x00}) {
		_, err := conn.Write([]byte{0x05, 0xFF})
		if err != nil {
			log.Printf("Failed to send CONNECT request: %v", err)
			return
		}
		return
	}

	_, err = conn.Write([]byte{0x05, 0x00})
	if err != nil {
		log.Printf("Failed to send CONNECT request: %v", err)
		return
	} // No Auth

	//SOCKS5 Request
	reqHeader := make([]byte, 4)
	if _, err := io.ReadFull(br, reqHeader); err != nil {
		return
	}

	atyp := reqHeader[3]
	var addr string
	switch atyp {
	case 0x01: // IPv4
		ip := make([]byte, 4)
		if _, err := io.ReadFull(br, ip); err != nil {
			return
		}
		addr = net.IP(ip).String()
	case 0x03: // Domain
		l, _ := br.ReadByte()
		domain := make([]byte, l)
		if _, err := io.ReadFull(br, domain); err != nil {
			return
		}
		addr = string(domain)
	case 0x04: // IPv6
		ip := make([]byte, 16)
		if _, err := io.ReadFull(br, ip); err != nil {
			return
		}
		addr = net.IP(ip).String()
	default:
		return
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(br, portBytes); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBytes)
	fullAddr := fmt.Sprintf("%s:%d", addr, port)
	log.Printf("Request to CONNECT %s", fullAddr)

	// Connect to HTTP Proxy
	proxyConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cfg.HTTPProxy.Host, cfg.HTTPProxy.Port), time.Duration(cfg.Timeouts.Dial)*time.Second)
	if err != nil {
		_, err := conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		if err != nil {
			log.Printf("Failed to send CONNECT request: %v", err)
			return
		}
		return
	}

	// Send CONNECT request
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", fullAddr, fullAddr)
	if cfg.HTTPProxy.Username != "" && cfg.HTTPProxy.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(cfg.HTTPProxy.Username + ":" + cfg.HTTPProxy.Password))
		req += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
	}
	req += "\r\n"

	if _, err := proxyConn.Write([]byte(req)); err != nil {
		log.Printf("Failed to send CONNECT request: %v", err)
		return
	}

	proxyReader := bufio.NewReader(proxyConn)
	statusLine, err := proxyReader.ReadString('\n')
	if err != nil || !strings.Contains(statusLine, "200") {
		log.Printf("HTTP proxy refused CONNECT: %s", strings.TrimSpace(statusLine))
		_, err := conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		if err != nil {
			log.Printf("Failed to send CONNECT request: %v", err)
			return
		}
		proxyConn.Close()
		return
	}

	// Consume headers
	for {
		line, err := proxyReader.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
	}

	//Reply to client: success
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		log.Printf("Failed to send CONNECT request: %v", err)
		return
	}

	//Tunnel (bidirectional)
	go proxy(proxyConn, conn)
	proxy(conn, proxyConn)
}

func proxy(src net.Conn, dst net.Conn) {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		log.Printf("proxy error: %v", err)
	}
	if err := dst.Close(); err != nil {
		log.Printf("error closing dst: %v", err)
	}
	if err := src.Close(); err != nil {
		log.Printf("error closing src: %v", err)
	}
}
