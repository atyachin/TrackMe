package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
)

func staticFile(file string) func(Response, url.Values) ([]byte, string) {
	return func(Response, url.Values) ([]byte, string) {
		b, _ := ReadFile(file)
		return b, "text/html"
	}
}

func apiAll(res Response, _ url.Values) ([]byte, string) {
	return []byte(res.ToJson()), "application/json"
}

func apiTLS(res Response, _ url.Values) ([]byte, string) {
	return []byte(Response{
		TLS: res.TLS,
	}.ToJson()), "application/json"
}

func apiClean(res Response, _ url.Values) ([]byte, string) {
	akamai := "-"
	hash := "-"
	if res.HTTPVersion == "h2" {
		akamai = res.Http2.AkamaiFingerprint
		hash = GetMD5Hash(res.Http2.AkamaiFingerprint)
	}
	return []byte(SmallResponse{
		JA3:           res.TLS.JA3,
		JA3Hash:       res.TLS.JA3Hash,
		Akamai:        akamai,
		AkamaiHash:    hash,
		PeetPrint:     res.TLS.PeetPrint,
		PeetPrintHash: res.TLS.PeetPrintHash,
	}.ToJson()), "application/json"
}

func apiRequestCount(_ Response, _ url.Values) ([]byte, string) {
	if !connectedToDB {
		return []byte("{\"error\": \"Not connected to database.\"}"), "application/json"
	}
	return []byte(fmt.Sprintf(`{"total_requests": %v}`, GetTotalRequestCount())), "application/json"
}

func apiSearchJA3(_ Response, u url.Values) ([]byte, string) {
	if !connectedToDB {
		return []byte("{\"error\": \"Not connected to database.\"}"), "application/json"
	}
	by := getParam("by", u)
	if by == "" {
		return []byte("{\"error\": \"No 'by' param present\"}"), "application/json"
	}
	res := GetByJa3(by)
	j, _ := json.MarshalIndent(res, "", "\t")
	return j, "application/json"
}

func apiSearchH2(_ Response, u url.Values) ([]byte, string) {
	if !connectedToDB {
		return []byte("{\"error\": \"Not connected to database.\"}"), "application/json"
	}
	by := getParam("by", u)
	if by == "" {
		return []byte("{\"error\": \"No 'by' param present\"}"), "application/json"
	}
	res := GetByH2(by)
	j, _ := json.MarshalIndent(res, "", "\t")
	return j, "application/json"
}

func apiSearchPeetPrint(_ Response, u url.Values) ([]byte, string) {
	if !connectedToDB {
		return []byte("{\"error\": \"Not connected to database.\"}"), "application/json"
	}
	by := getParam("by", u)
	if by == "" {
		return []byte("{\"error\": \"No 'by' param present\"}"), "application/json"
	}
	res := GetByPeetPrint(by)
	j, _ := json.MarshalIndent(res, "", "\t")
	return j, "application/json"
}

func apiSearchUserAgent(_ Response, u url.Values) ([]byte, string) {
	if !connectedToDB {
		return []byte("{\"error\": \"Not connected to database.\"}"), "application/json"
	}
	by := getParam("by", u)
	if by == "" {
		return []byte("{\"error\": \"No 'by' param present\"}"), "application/json"
	}
	res := GetByUserAgent(by)
	j, _ := json.MarshalIndent(res, "", "\t")
	return j, "application/json"
}

func index(r Response, v url.Values) ([]byte, string) {
	res, ct := staticFile("static/index.html")(r, v)
	data, _ := json.Marshal(r)
	return []byte(strings.ReplaceAll(string(res), "/*DATA*/", string(data))), ct
}

// write all to log/db and return an empty pixel
func apiEmptyGif(res Response, _ url.Values) ([]byte, string) {
	// Define the byte slice for a 1x1 transparent GIF
	var emptyGif = []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
		0x01, 0x00, 0x80, 0xff, 0x00, 0xc0, 0xc0, 0xc0,
		0x00, 0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
		0x44, 0x01, 0x00, 0x3b,
	}
	data, err := json.Marshal(res)
	if err != nil {
		log.Fatalf("Error occurred during marshaling. Error: %s", err.Error())
	} else {
		Log(string(data)) // todo: write to file/db
	}
	return emptyGif, "image/gif"
}

func getAllPaths() map[string]func(Response, url.Values) ([]byte, string) {
	return map[string]func(Response, url.Values) ([]byte, string){
		"/":                     index,
		"/explore":              staticFile("static/explore.html"),
		"/api/all":              apiAll,
		"/api/tls":              apiTLS,
		"/api/clean":            apiClean,
		"/api/request-count":    apiRequestCount,
		"/api/search-ja3":       apiSearchJA3,
		"/api/search-h2":        apiSearchH2,
		"/api/search-peetprint": apiSearchPeetPrint,
		"/api/search-useragent": apiSearchUserAgent,
		"/pixel.gif":            apiEmptyGif,
	}
}
