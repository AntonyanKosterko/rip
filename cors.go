package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) bool {
	lower := strings.ToLower(origin)
	// Tauri 2 WebView: tauri://localhost, http(s)://tauri.localhost
	if strings.Contains(lower, "tauri") {
		return true
	}
	if strings.HasPrefix(lower, "http://localhost:") ||
		strings.HasPrefix(lower, "https://localhost:") ||
		strings.HasPrefix(lower, "http://127.0.0.1:") ||
		strings.HasPrefix(lower, "https://127.0.0.1:") {
		return true
	}
	if strings.HasPrefix(lower, "https://antonyankosterko.github.io") {
		return true
	}
	if strings.HasPrefix(lower, "http://192.168.") ||
		strings.HasPrefix(lower, "https://192.168.") ||
		strings.HasPrefix(lower, "http://10.") ||
		strings.HasPrefix(lower, "https://10.") {
		return true
	}
	if strings.HasPrefix(lower, "http://172.") || strings.HasPrefix(lower, "https://172.") {
		return true
	}
	return false
}

func printListenAddresses(port string) {
	fmt.Printf("Сервер слушает 0.0.0.0:%s (все интерфейсы)\n", port)
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			fmt.Printf("  LAN: http://%s:%s\n", ip.String(), port)
		}
	}
	minioURL := getEnv("MINIO_PUBLIC_URL", "http://localhost:9000")
	fmt.Printf("MINIO_PUBLIC_URL=%s\n", minioURL)
}
