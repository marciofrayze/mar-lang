package runtime

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"belm/internal/model"
)

// resolveSystemAuth resolves a bearer token into a system-admin session.
func (r *Runtime) resolveSystemAuth(req *http.Request) (authSession, error) {
	token := parseBearerToken(req.Header.Get("Authorization"))
	if token == "" {
		return authSession{}, nil
	}

	row, ok, err := queryRow(
		r.DB,
		`SELECT token, admin_user_id, email, expires_at, revoked FROM belm_admin_sessions WHERE token = ? LIMIT 1`,
		token,
	)
	if err != nil {
		return authSession{}, err
	}
	if !ok {
		return authSession{}, nil
	}
	if revoked, _ := toInt64(row["revoked"]); revoked == 1 {
		return authSession{}, nil
	}
	expiresAt, _ := toInt64(row["expires_at"])
	if expiresAt < time.Now().UnixMilli() {
		return authSession{}, nil
	}

	userID := row["admin_user_id"]
	adminRow, ok, err := r.loadSystemAdminByID(userID)
	if err != nil {
		return authSession{}, err
	}
	if !ok {
		return authSession{}, nil
	}
	email, _ := row["email"].(string)
	role := adminRow["role"]
	if role == nil {
		role = "admin"
	}

	return authSession{
		Authenticated: true,
		Token:         token,
		Email:         email,
		UserID:        userID,
		Role:          role,
		User:          adminRow,
	}, nil
}

func (r *Runtime) loadSystemAdminByEmail(email string) (map[string]any, bool, error) {
	return queryRow(r.DB, `SELECT * FROM belm_admin_users WHERE email = ? LIMIT 1`, email)
}

func (r *Runtime) loadSystemAdminByID(id any) (map[string]any, bool, error) {
	return queryRow(r.DB, `SELECT * FROM belm_admin_users WHERE id = ? LIMIT 1`, id)
}

func (r *Runtime) countSystemAdminUsers() (int64, error) {
	row, ok, err := queryRow(r.DB, `SELECT COUNT(*) AS total FROM belm_admin_users`)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	total, _ := toInt64(row["total"])
	return total, nil
}

// handleSystemAuthRequestCode creates and delivers a one-time login code for system admin users.
func (r *Runtime) handleSystemAuthRequestCode(w http.ResponseWriter, payload map[string]any) error {
	totalUsers, err := r.countSystemAdminUsers()
	if err != nil {
		return err
	}
	if totalUsers == 0 {
		return &apiError{
			Status:  http.StatusConflict,
			Message: "No system admin users exist yet. Create the first admin using /_belm/admin/bootstrap",
		}
	}

	email, err := parseAuthEmail(payload)
	if err != nil {
		return err
	}
	adminUser, found, err := r.loadSystemAdminByEmail(email)
	if err != nil {
		return err
	}
	if !found {
		r.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "If this email exists, a code was sent."})
		return nil
	}

	adminUserID := adminUser["id"]
	return r.issueSystemAuthCode(w, email, adminUserID, "If this email exists, a code was sent.")
}

// handleSystemBootstrapAdmin creates the first system admin user and sends a login code.
func (r *Runtime) handleSystemBootstrapAdmin(w http.ResponseWriter, payload map[string]any) error {
	totalUsers, err := r.countSystemAdminUsers()
	if err != nil {
		return err
	}
	if totalUsers > 0 {
		return &apiError{Status: http.StatusConflict, Message: "Bootstrap is only allowed when there are no system admin users"}
	}

	email, err := parseAuthEmail(payload)
	if err != nil {
		return err
	}

	user, found, err := r.createSystemAdminUser(email, "admin")
	if err != nil {
		return err
	}
	if !found {
		return &apiError{Status: http.StatusUnprocessableEntity, Message: "Could not create first system admin user"}
	}

	return r.issueSystemAuthCode(w, email, user["id"], "First system admin user created. A login code was sent.")
}

func (r *Runtime) createSystemAdminUser(email, role string) (map[string]any, bool, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, false, &apiError{Status: http.StatusBadRequest, Message: "email is required"}
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = "admin"
	}

	user, found, err := r.loadSystemAdminByEmail(email)
	if err != nil || found {
		return user, found, err
	}

	now := time.Now().UnixMilli()
	if _, err := r.DB.Exec(`INSERT INTO belm_admin_users (email, role, created_at) VALUES (?, ?, ?)`, email, role, now); err != nil {
		// If a concurrent request created the same user, load and continue.
		user, found, loadErr := r.loadSystemAdminByEmail(email)
		if loadErr == nil && found {
			return user, true, nil
		}
		return nil, false, err
	}
	return r.loadSystemAdminByEmail(email)
}

func (r *Runtime) issueSystemAuthCode(w http.ResponseWriter, email string, userID any, message string) error {
	cfg := r.systemAuthConfig()

	code, err := randomCode6()
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	expiresAt := now + int64(cfg.CodeTTLMinutes)*60_000
	_, err = r.DB.Exec(
		`INSERT INTO belm_admin_codes (email, admin_user_id, code, expires_at, used, created_at) VALUES (?, ?, ?, ?, 0, ?)`,
		email,
		userID,
		code,
		expiresAt,
		now,
	)
	if err != nil {
		return err
	}

	if err := r.deliverSystemEmailCode(email, code, cfg); err != nil {
		return err
	}

	resp := map[string]any{"ok": true, "message": message}
	if cfg.DevExposeCode {
		resp["devCode"] = code
	}
	r.writeJSON(w, http.StatusOK, resp)
	return nil
}

// handleSystemAuthLogin verifies a system-admin email+code pair and issues a system session token.
func (r *Runtime) handleSystemAuthLogin(w http.ResponseWriter, payload map[string]any) error {
	emailRaw, ok := payload["email"].(string)
	if !ok {
		return &apiError{Status: http.StatusBadRequest, Message: "email is required"}
	}
	codeRaw, ok := payload["code"].(string)
	if !ok {
		return &apiError{Status: http.StatusBadRequest, Message: "code is required"}
	}

	email := normalizeEmail(emailRaw)
	code := strings.TrimSpace(codeRaw)
	if email == "" {
		return &apiError{Status: http.StatusBadRequest, Message: "email is required"}
	}
	if code == "" {
		return &apiError{Status: http.StatusBadRequest, Message: "code is required"}
	}

	row, ok, err := queryRow(
		r.DB,
		`SELECT id, admin_user_id, code, expires_at, used FROM belm_admin_codes WHERE email = ? ORDER BY id DESC LIMIT 1`,
		email,
	)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	if !ok {
		return &apiError{Status: http.StatusUnauthorized, Message: "Invalid or expired code"}
	}
	used, _ := toInt64(row["used"])
	expiresAt, _ := toInt64(row["expires_at"])
	storedCode, _ := row["code"].(string)
	if used == 1 || expiresAt < now || storedCode != code {
		return &apiError{Status: http.StatusUnauthorized, Message: "Invalid or expired code"}
	}

	codeID, _ := toInt64(row["id"])
	adminUserID := row["admin_user_id"]
	if _, err := r.DB.Exec(`UPDATE belm_admin_codes SET used = 1 WHERE id = ?`, codeID); err != nil {
		return err
	}

	adminRow, found, err := r.loadSystemAdminByID(adminUserID)
	if err != nil {
		return err
	}
	if !found {
		return &apiError{Status: http.StatusUnauthorized, Message: "Invalid or expired code"}
	}

	token, err := randomToken(32)
	if err != nil {
		return err
	}
	cfg := r.systemAuthConfig()
	sessionExpiresAt := now + int64(cfg.SessionTTLHours)*60*60*1000
	if _, err := r.DB.Exec(
		`INSERT INTO belm_admin_sessions (token, admin_user_id, email, expires_at, revoked, created_at) VALUES (?, ?, ?, ?, 0, ?)`,
		token,
		adminUserID,
		email,
		sessionExpiresAt,
		now,
	); err != nil {
		return err
	}

	r.writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"token":     token,
		"expiresAt": sessionExpiresAt,
		"user":      adminRow,
	})
	return nil
}

// handleSystemAuthLogout revokes the caller system-admin session token.
func (r *Runtime) handleSystemAuthLogout(w http.ResponseWriter, auth authSession) error {
	if !auth.Authenticated || auth.Token == "" {
		return &apiError{Status: http.StatusUnauthorized, Message: "Authentication required"}
	}
	if _, err := r.DB.Exec(`UPDATE belm_admin_sessions SET revoked = 1 WHERE token = ?`, auth.Token); err != nil {
		return err
	}
	r.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	return nil
}

func (r *Runtime) systemAuthConfig() model.AuthConfig {
	cfg := model.AuthConfig{
		CodeTTLMinutes:  10,
		SessionTTLHours: 24,
		EmailTransport:  "console",
		EmailFrom:       "no-reply@belm.local",
		EmailSubject:    "Your Belm Admin login code",
		SendmailPath:    "/usr/sbin/sendmail",
		DevExposeCode:   true,
	}
	if r.App.Auth != nil {
		if r.App.Auth.CodeTTLMinutes > 0 {
			cfg.CodeTTLMinutes = r.App.Auth.CodeTTLMinutes
		}
		if r.App.Auth.SessionTTLHours > 0 {
			cfg.SessionTTLHours = r.App.Auth.SessionTTLHours
		}
		if strings.TrimSpace(r.App.Auth.EmailTransport) != "" {
			cfg.EmailTransport = strings.TrimSpace(r.App.Auth.EmailTransport)
		}
		if strings.TrimSpace(r.App.Auth.EmailFrom) != "" {
			cfg.EmailFrom = strings.TrimSpace(r.App.Auth.EmailFrom)
		}
		if strings.TrimSpace(r.App.Auth.EmailSubject) != "" {
			cfg.EmailSubject = strings.TrimSpace(r.App.Auth.EmailSubject)
		}
		if strings.TrimSpace(r.App.Auth.SendmailPath) != "" {
			cfg.SendmailPath = strings.TrimSpace(r.App.Auth.SendmailPath)
		}
		cfg.DevExposeCode = r.App.Auth.DevExposeCode
	}
	return cfg
}

func (r *Runtime) deliverSystemEmailCode(toEmail, code string, cfg model.AuthConfig) error {
	if cfg.DevExposeCode {
		r.printSystemAuthLogHeader()
		useColor := supportsANSI()
		fmt.Printf("  %s %s  %s %s\n",
			colorize(useColor, ansiLabel, "Dev code:"),
			colorize(useColor, ansiCommand, code),
			colorize(useColor, ansiLabel, "Email:"),
			toEmail,
		)
	}

	switch cfg.EmailTransport {
	case "console":
		r.printSystemAuthLogHeader()
		useColor := supportsANSI()
		fmt.Printf("  %s %s  %s %s  %s %s\n",
			colorize(useColor, ansiLabel, "Email:"),
			colorize(useColor, ansiCommand, "sent"),
			colorize(useColor, ansiLabel, "Transport:"),
			"console",
			colorize(useColor, ansiLabel, "To:"),
			toEmail,
		)
		return nil
	case "sendmail":
		if err := sendWithSendmail(cfg.SendmailPath, cfg.EmailFrom, cfg.EmailSubject, toEmail, code, cfg.CodeTTLMinutes); err != nil {
			return err
		}
		r.printSystemAuthLogHeader()
		useColor := supportsANSI()
		fmt.Printf("  %s %s  %s %s  %s %s\n",
			colorize(useColor, ansiLabel, "Email:"),
			colorize(useColor, ansiCommand, "sent"),
			colorize(useColor, ansiLabel, "Transport:"),
			"sendmail",
			colorize(useColor, ansiLabel, "To:"),
			toEmail,
		)
		return nil
	default:
		return fmt.Errorf("unsupported email transport %q", cfg.EmailTransport)
	}
}

func (r *Runtime) printSystemAuthLogHeader() {
	useColor := supportsANSI()
	r.systemAuthLogOnce.Do(func() {
		fmt.Println()
		fmt.Printf("%s\n", colorize(useColor, ansiSection, "System auth logs"))
	})
}
