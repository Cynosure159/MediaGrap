package webhooks

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid integration configuration")
var ErrConflict = errors.New("integration state changed")
var ErrResolution = errors.New("webhook DNS resolution failed")
var ErrUnavailable = errors.New("webhook signing key unavailable")

// NetworkPolicy is deployment-owned; endpoint CRUD cannot relax it.
type NetworkPolicy struct {
	lookup         func(context.Context, string) ([]net.IPAddr, error)
	AllowHTTP      bool
	AllowedTargets []string // exact host:port; private addresses additionally require a CIDR
	AllowedCIDRs   []netip.Prefix
}

func defaultPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

func (p NetworkPolicy) validate(ctx context.Context, raw string) (*url.URL, []net.IPAddr, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u.Hostname() == "" || u.User != nil || strings.Contains(u.Hostname(), "%") || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, nil, ErrInvalid
	}
	port := defaultPort(u)
	allowed := false
	for _, target := range p.AllowedTargets {
		if strings.EqualFold(target, net.JoinHostPort(u.Hostname(), port)) {
			allowed = true
		}
	}
	if u.Scheme == "http" && (!p.AllowHTTP || !allowed) {
		return nil, nil, ErrInvalid
	}
	lookup := p.lookup
	if lookup == nil {
		lookup = net.DefaultResolver.LookupIPAddr
	}
	addresses, err := lookup(ctx, u.Hostname())
	if err != nil {
		return nil, nil, ErrResolution
	}
	if len(addresses) == 0 {
		return nil, nil, ErrResolution
	}
	for _, ip := range addresses {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok {
			return nil, nil, ErrInvalid
		}
		addr = addr.Unmap()
		if !addr.IsGlobalUnicast() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
			return nil, nil, ErrInvalid
		}
		// Includes CGNAT and reserved space often used by metadata services.
		if addr == netip.MustParseAddr("100.100.100.200") {
			return nil, nil, ErrInvalid
		}
		restricted := addr.IsPrivate() || netip.MustParsePrefix("100.64.0.0/10").Contains(addr) || netip.MustParsePrefix("240.0.0.0/4").Contains(addr)
		if restricted {
			matched := false
			for _, cidr := range p.AllowedCIDRs {
				if cidr.Contains(addr) {
					matched = true
				}
			}
			if !allowed || !matched {
				return nil, nil, ErrInvalid
			}
		}
	}
	return u, addresses, nil
}
func (p NetworkPolicy) client(ctx context.Context, raw string) (*http.Client, error) {
	u, ips, err := p.validate(ctx, raw)
	if err != nil {
		return nil, err
	}
	port := defaultPort(u)
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, MaxResponseHeaderBytes: 16 << 10, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: u.Hostname()}, ResponseHeaderTimeout: 10 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			var last error
			for _, ip := range ips {
				conn, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				last = err
			}
			return nil, last
		}}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}
func Signature(secret []byte, timestamp string, body []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(timestamp + "."))
	h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
func loadCipher(path string) (cipher.AEAD, error) {
	if path == "" {
		return nil, ErrUnavailable
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, ErrUnavailable
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrUnavailable
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(string(key)))
	if err != nil || len(decoded) != 32 {
		return nil, ErrUnavailable
	}
	block, err := aes.NewCipher(decoded)
	if err != nil {
		return nil, ErrUnavailable
	}
	return cipher.NewGCM(block)
}
func encrypt(a cipher.AEAD, keyID string, secret []byte) []byte {
	nonce := make([]byte, a.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}
	return a.Seal(nonce, nonce, secret, []byte(keyID))
}
func decrypt(a cipher.AEAD, keyID string, data []byte) ([]byte, error) {
	if a == nil || len(data) < a.NonceSize() {
		return nil, ErrUnavailable
	}
	value, err := a.Open(nil, data[:a.NonceSize()], data[a.NonceSize():], []byte(keyID))
	if err != nil {
		return nil, ErrUnavailable
	}
	return value, nil
}
