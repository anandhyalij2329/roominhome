package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var phoneRE = regexp.MustCompile(`^[6-9]\d{9}$`)

type sendOTPRequest struct {
	Channel string `json:"channel"` // email | phone
	Target  string `json:"target"`
}

type verifyOTPRequest struct {
	Channel string `json:"channel"`
	Target  string `json:"target"`
	Code    string `json:"code"`
}

func normalizePhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.TrimPrefix(s, "+91")
	s = strings.TrimPrefix(s, "91")
	if strings.HasPrefix(s, "0") && len(s) == 11 {
		s = s[1:]
	}
	if !phoneRE.MatchString(s) {
		return "", fmt.Errorf("enter a valid 10-digit Indian mobile number")
	}
	return s, nil
}

func normalizeEmail(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address == "" || !strings.Contains(addr.Address, ".") {
		return "", fmt.Errorf("enter a valid email address")
	}
	return addr.Address, nil
}

func (a *API) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	target := strings.TrimSpace(req.Target)

	var err error
	switch channel {
	case "email":
		target, err = normalizeEmail(target)
	case "phone":
		target, err = normalizePhone(target)
	default:
		writeError(w, http.StatusBadRequest, "channel must be email or phone")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	codeNum, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate code")
		return
	}
	code := fmt.Sprintf("%06d", codeNum.Int64())
	id := uuid.NewString()
	expires := time.Now().UTC().Add(10 * time.Minute)

	_, _ = a.DB.Exec(`UPDATE otps SET used = 1 WHERE channel = ? AND target = ? AND used = 0`, channel, target)
	_, err = a.DB.Exec(
		`INSERT INTO otps (id, channel, target, code, expires_at, used, created_at) VALUES (?, ?, ?, ?, ?, 0, ?)`,
		id, channel, target, code, expires, time.Now().UTC(),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save otp")
		return
	}

	resp := map[string]any{
		"ok":      true,
		"message": "Verification code sent",
		"channel": channel,
		"target":  target,
	}
	// Free hosting: no SMS/email provider — expose demo code so users can verify.
	if a.OTPDemo {
		resp["demo_code"] = code
		resp["message"] = "Demo mode: use the code shown below"
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	target := strings.TrimSpace(req.Target)
	code := strings.TrimSpace(req.Code)

	var err error
	switch channel {
	case "email":
		target, err = normalizeEmail(target)
	case "phone":
		target, err = normalizePhone(target)
	default:
		writeError(w, http.StatusBadRequest, "channel must be email or phone")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(code) != 6 {
		writeError(w, http.StatusBadRequest, "enter the 6-digit code")
		return
	}

	var id string
	var expires time.Time
	err = a.DB.QueryRow(
		`SELECT id, expires_at FROM otps WHERE channel = ? AND target = ? AND code = ? AND used = 0 ORDER BY created_at DESC LIMIT 1`,
		channel, target, code,
	).Scan(&id, &expires)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	if time.Now().UTC().After(expires) {
		writeError(w, http.StatusBadRequest, "code expired — request a new one")
		return
	}
	_, _ = a.DB.Exec(`UPDATE otps SET used = 1 WHERE id = ?`, id)

	token := a.makeVerifiedToken(channel, target)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"verified_token":  token,
		"channel":         channel,
		"target":          target,
	})
}

func (a *API) makeVerifiedToken(channel, target string) string {
	// HMAC token valid ~2 hours for registration / property contact
	exp := time.Now().UTC().Add(2 * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%s|%d", channel, target, exp)
	mac := hmac.New(sha256.New, []byte(a.JWTSecret+"|otp-verify"))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return payload + "|" + sig
}

func (a *API) checkVerifiedToken(channel, target, token string) bool {
	parts := strings.Split(token, "|")
	if len(parts) != 4 {
		return false
	}
	if parts[0] != channel || parts[1] != target {
		return false
	}
	var exp int64
	if _, err := fmt.Sscanf(parts[2], "%d", &exp); err != nil {
		return false
	}
	if time.Now().UTC().Unix() > exp {
		return false
	}
	payload := parts[0] + "|" + parts[1] + "|" + parts[2]
	mac := hmac.New(sha256.New, []byte(a.JWTSecret+"|otp-verify"))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(parts[3]))
}
