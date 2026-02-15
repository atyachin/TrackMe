package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/pagpeter/trackme/pkg/types"
	"github.com/pagpeter/trackme/pkg/utils"
)

// RouteHandler is the function signature for route handlers
type RouteHandler func(types.Response, url.Values) ([]byte, string, error)

var (
	ErrTLSNotAvailable = errors.New("TLS details not available")
)

func staticFile(file string) RouteHandler {
	return func(types.Response, url.Values) ([]byte, string, error) {
		b, err := utils.ReadFile(file)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read file %s: %w", file, err)
		}
		return b, "text/html", nil
	}
}

func apiAll(res types.Response, _ url.Values) ([]byte, string, error) {
	return []byte(res.ToJson()), "application/json", nil
}

func apiTLS(res types.Response, _ url.Values) ([]byte, string, error) {
	return []byte(types.Response{
		TLS: res.TLS,
	}.ToJson()), "application/json", nil
}

func apiClean(res types.Response, _ url.Values) ([]byte, string, error) {
	akamai := "-"
	hash := "-"
	if res.HTTPVersion == "h2" && res.Http2 != nil {
		akamai = res.Http2.AkamaiFingerprint
		hash = utils.GetMD5Hash(res.Http2.AkamaiFingerprint)
	} else if res.HTTPVersion == "h3" && res.Http3 != nil {
		akamai = res.Http3.AkamaiFingerprint
		hash = res.Http3.AkamaiFingerprintHash
	}

	smallRes := types.SmallResponse{
		Akamai:      akamai,
		AkamaiHash:  hash,
		HTTPVersion: res.HTTPVersion,
	}

	if res.TLS != nil {
		smallRes.JA3 = res.TLS.JA3
		smallRes.JA3Hash = res.TLS.JA3Hash
		smallRes.JA4 = res.TLS.JA4
		smallRes.JA4_r = res.TLS.JA4_r
		smallRes.PeetPrint = res.TLS.PeetPrint
		smallRes.PeetPrintHash = res.TLS.PeetPrintHash
	}

	return []byte(smallRes.ToJson()), "application/json", nil
}

func apiRaw(res types.Response, _ url.Values) ([]byte, string, error) {
	if res.TLS == nil {
		return nil, "", ErrTLSNotAvailable
	}
	return []byte(fmt.Sprintf(`{"raw": "%s", "raw_b64": "%s"}`, res.TLS.RawBytes, res.TLS.RawB64)), "application/json", nil
}

func index(r types.Response, v url.Values) ([]byte, string, error) {
	res, ct, err := staticFile("static/index.html")(r, v)
	if err != nil {
		return nil, "", err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return []byte(strings.ReplaceAll(string(res), "/*DATA*/", string(data))), ct, nil
}

// apiEmptyGif returns a 1x1 transparent GIF and logs the full request payload.
func apiEmptyGif(res types.Response, _ url.Values) ([]byte, string, error) {
	emptyGif := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
		0x01, 0x00, 0x80, 0xff, 0x00, 0xc0, 0xc0, 0xc0,
		0x00, 0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
		0x44, 0x01, 0x00, 0x3b,
	}

	if data, err := json.Marshal(res); err == nil {
		Log(string(data))
	}

	return emptyGif, "image/gif", nil
}

func getAllPaths() map[string]RouteHandler {
	return map[string]RouteHandler{
		"/":              index,
		"/explore":       staticFile("static/explore.html"),
		"/api/all":       apiAll,
		"/api/tls":       apiTLS,
		"/api/clean":     apiClean,
		"/api/raw":       apiRaw,
		"/pixel.gif":     apiEmptyGif,
		"/analytics.gif": apiEmptyGif,
	}
}
