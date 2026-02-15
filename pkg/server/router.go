package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pagpeter/trackme/pkg/tls"
	"github.com/pagpeter/trackme/pkg/types"
	"github.com/pagpeter/trackme/pkg/utils"
)

func Log(msg string) {
	t := time.Now()
	formatted := t.Format("2006-01-02 15:04:05")
	fmt.Printf("[%v] %v\n", formatted, msg)
}

// WriteLog appends a JSON string to the configured log file.
func WriteLog(msg string, logFile string) error {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(msg + "\n\n")
	return err
}

func cleanIP(ip string) string {
	return strings.Replace(strings.Replace(ip, "]", "", -1), "[", "", -1)
}

// Router returns bytes, content type, and error that should be sent to the client
func Router(path string, res types.Response, srv *Server) ([]byte, string, error) {
	if v, ok := srv.GetTCPFingerprints().Load(res.IP); ok {
		res.TCPIP = v.(types.TCPIPDetails)
	}
	if res.TLS != nil {
		// Use QUIC JA4 for HTTP/3 connections
		if res.HTTPVersion == "h3" {
			res.TLS.JA4 = tls.CalculateJa4QUIC(res.TLS)
			res.TLS.JA4_r = tls.CalculateJa4QUIC_r(res.TLS)
		} else {
			res.TLS.JA4 = tls.CalculateJa4(res.TLS)
			res.TLS.JA4_r = tls.CalculateJa4_r(res.TLS)
		}
		Log(fmt.Sprintf("%v %v %v %v %v", cleanIP(res.IP), res.Method, res.HTTPVersion, res.Path, res.TLS.JA3Hash))
	} else {
		Log(fmt.Sprintf("%v %v %v %v %v", cleanIP(res.IP), res.Method, res.HTTPVersion, res.Path, "-"))
	}
	if srv.GetConfig().LogFile != "" && res.Path != "/favicon.ico" {
		data, err := json.Marshal(res)
		if err != nil {
			log.Printf("failed to marshal request for file logging: %v", err)
		} else if err := WriteLog(string(data), srv.GetConfig().LogFile); err != nil {
			log.Printf("failed to write request log file: %v", err)
		}
	}

	u, err := url.Parse("https://tls.peet.ws" + path)
	var m map[string][]string
	if err != nil || u == nil {
		m = make(map[string][]string)
	} else {
		m, err = url.ParseQuery(u.RawQuery)
		if err != nil {
			m = make(map[string][]string)
		}
	}

	paths := getAllPaths()
	if u != nil {
		if val, ok := paths[u.Path]; ok {
			return val(res, m)
		}
	}
	// 404
	b, err := utils.ReadFile("static/404.html")
	if err != nil {
		return []byte(`{"error": "page not found"}`), "application/json", nil
	}
	return []byte(b), "text/html", nil
}
