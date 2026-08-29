package ip

import (
	"net"
	"net/http"
	"strings"
)

func GetIP(request *http.Request) string {
	xForwardedFor := request.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}
	ip, _, errSplitHost := net.SplitHostPort(request.RemoteAddr)
	if errSplitHost != nil {
		return request.RemoteAddr
	}
	return ip
}

func CompareIP(oldIP, newIP string) bool {
	ipPartsOld := strings.Split(oldIP, ".")
	ipPartsNew := strings.Split(newIP, ".")
	if len(ipPartsNew) < 2 || len(ipPartsOld) < 2 {
		return true
	}
	if ipPartsOld[0] != ipPartsNew[0] || ipPartsOld[1] != ipPartsNew[1] {
		return false
	}
	return true
}
