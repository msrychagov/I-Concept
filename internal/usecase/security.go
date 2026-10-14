package usecase

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

func passwordHash(password, salt string) string {
	key, err := pbkdf2.Key(sha256.New, password, []byte(salt), 600000, 32)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(key)
}
func token() string { b := make([]byte, 32); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

const inviteAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

func newInviteCode() string {
	b := make([]byte, 10)
	random := make([]byte, 10)
	_, _ = rand.Read(random)
	for i := range b {
		b[i] = inviteAlphabet[int(random[i])&31]
	}
	return string(b[:5]) + "-" + string(b[5:])
}

func normalizeInviteCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "", " ", "").Replace(value)
	if len(value) != 10 {
		return ""
	}
	for _, c := range value {
		if !strings.ContainsRune(inviteAlphabet, c) {
			return ""
		}
	}
	return value
}

func newStudentInviteToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func normalizeStudentInvite(value string) string {
	value = strings.TrimSpace(value)
	if parsed, err := url.Parse(value); err == nil && parsed.Query().Get("invite") != "" {
		value = parsed.Query().Get("invite")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return ""
	}
	return value
}
