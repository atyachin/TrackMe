package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
)

func Log(msg string) {
	t := time.Now()
	formatted := t.Format("2006-01-02 15:04:05")
	fmt.Printf("[%v] %v\n", formatted, msg)
}

// WriteLog append msg to log_file in a new line. pass input as json
func WriteLog(msg string, log_file string) error {
	// Open the file in append mode. If it doesn't exist, create it with permissions 0666.
	file, err := os.OpenFile(log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the text to the file
	msg += "\n\n"
	_, err = io.WriteString(file, msg)
	if err != nil {
		return err
	}
	return nil
}

func cleanIP(ip string) string {
	return strings.Replace(strings.Replace(ip, "]", "", -1), "[", "", -1)
}

// Router returns bytes and content type that should be sent to the client
func Router(path string, res Response) ([]byte, string) {
	if v, ok := TCPFingerprints.Load(cleanIP(res.IP)); ok {
		res.TCPIP = v.(TCPIPDetails)
	}
	res.TLS.JA4 = CalculateJa4(res.TLS)
	res.TLS.JA4_r = CalculateJa4_r(res.TLS)
	// res.Donate = "Please consider donating to keep this API running. Visit https://tls.peet.ws"
	Log(fmt.Sprintf("%v %v %v %v %v", cleanIP(res.IP), res.Method, res.HTTPVersion, res.Path, res.TLS.JA3Hash))
	// if GetUserAgent(res) == "" {
	//	return []byte("{\"error\": \"No user-agent\"}"), "text/html"
	// }
	if LoadedConfig.LogToDB && res.Path != "/favicon.ico" {
		SaveRequest(res)
	}
	if LoadedConfig.LogFile != "" && res.Path != "/favicon.ico" {
		data, err := json.Marshal(res)
		if err != nil {
			log.Fatalf("Error occurred during marshaling. Error: %s", err.Error())
		} else {
			WriteLog(string(data), LoadedConfig.LogFile)
		}
	}

	u, _ := url.Parse("https://tls.peet.ws" + path)
	m, _ := url.ParseQuery(u.RawQuery)

	paths := getAllPaths()
	if val, ok := paths[u.Path]; ok {
		return val(res, m)
	}
	// 404
	b, _ := ReadFile("static/404.html")
	return []byte(strings.ReplaceAll(string(b), "/*DATA*/", fmt.Sprintf("%v", GetTotalRequestCount()))), "text/html"
}
