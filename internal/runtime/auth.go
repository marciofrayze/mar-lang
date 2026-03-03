package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// resolveAuth resolves a bearer token into an active session and hydrated auth user.
func (r *Runtime) resolveAuth(req *http.Request) (authSession, error) {
	if !r.authEnabled() || r.authUser == nil {
		return authSession{}, nil
	}
	token := parseBearerToken(req.Header.Get("Authorization"))
	if token == "" {
		return authSession{}, nil
	}

	row, ok, err := queryRow(
		r.DB,
		`SELECT token, user_id, email, expires_at, revoked FROM belm_sessions WHERE token = ? LIMIT 1`,
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

	userID := row["user_id"]
	userRow, ok, err := r.loadAuthUserByID(userID)
	if err != nil {
		return authSession{}, err
	}
	if !ok {
		return authSession{}, nil
	}
	user := decodeEntityRow(r.authUser, userRow)
	role := any(nil)
	if r.App.Auth.RoleField != "" {
		role = user[r.App.Auth.RoleField]
	}

	email, _ := row["email"].(string)
	return authSession{
		Authenticated: true,
		Token:         token,
		Email:         email,
		UserID:        userID,
		Role:          role,
		User:          user,
	}, nil
}

// parseBearerToken extracts the token from an Authorization header.
func parseBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func (r *Runtime) loadAuthUserByEmail(email string) (map[string]any, bool, error) {
	table, _ := quoteIdentifier(r.authUser.Table)
	emailField, _ := quoteIdentifier(r.App.Auth.EmailField)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ? LIMIT 1", table, emailField)
	return queryRow(r.DB, query, email)
}

func (r *Runtime) loadAuthUserByID(id any) (map[string]any, bool, error) {
	table, _ := quoteIdentifier(r.authUser.Table)
	pk, _ := quoteIdentifier(r.authUser.PrimaryKey)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ? LIMIT 1", table, pk)
	return queryRow(r.DB, query, id)
}

func (r *Runtime) countAuthUsers() (int64, error) {
	if r.authUser == nil {
		return 0, nil
	}
	table, _ := quoteIdentifier(r.authUser.Table)
	row, ok, err := queryRow(r.DB, fmt.Sprintf("SELECT COUNT(*) AS total FROM %s", table))
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	total, _ := toInt64(row["total"])
	return total, nil
}

func parseAuthEmail(payload map[string]any) (string, error) {
	emailRaw, ok := payload["email"].(string)
	if !ok {
		return "", &apiError{Status: http.StatusBadRequest, Message: "email is required"}
	}
	email := normalizeEmail(emailRaw)
	if email == "" {
		return "", &apiError{Status: http.StatusBadRequest, Message: "email is required"}
	}
	return email, nil
}

// handleAuthRequestCode creates and delivers a one-time login code for an existing user email.
func (r *Runtime) handleAuthRequestCode(w http.ResponseWriter, payload map[string]any) error {
	if !r.authEnabled() {
		return &apiError{Status: http.StatusNotFound, Message: "Authentication is not enabled"}
	}
	email, err := parseAuthEmail(payload)
	if err != nil {
		return err
	}

	user, found, err := r.loadOrCreateAuthUserForRequestCode(email)
	if err != nil {
		return err
	}
	if !found {
		r.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "If this email exists, a code was sent."})
		return nil
	}
	userID := user[r.authUser.PrimaryKey]
	return r.issueAuthCode(w, email, userID, "If this email exists, a code was sent.")
}

// handleBootstrapAdmin creates the first auth user with role admin and sends a login code.
func (r *Runtime) handleBootstrapAdmin(w http.ResponseWriter, payload map[string]any) error {
	if !r.authEnabled() {
		return &apiError{Status: http.StatusNotFound, Message: "Authentication is not enabled"}
	}
	if r.authUser == nil {
		return &apiError{Status: http.StatusInternalServerError, Message: "Auth user entity is not configured"}
	}
	if strings.TrimSpace(r.App.Auth.RoleField) == "" {
		return &apiError{Status: http.StatusBadRequest, Message: "auth.role_field is required for admin bootstrap"}
	}

	totalUsers, err := r.countAuthUsers()
	if err != nil {
		return err
	}
	if totalUsers > 0 {
		return &apiError{Status: http.StatusConflict, Message: "Bootstrap is only allowed when there are no users"}
	}

	email, err := parseAuthEmail(payload)
	if err != nil {
		return err
	}

	user, found, err := r.tryAutoCreateAuthUserWithRole(email, "admin")
	if err != nil {
		return err
	}
	if !found {
		return &apiError{Status: http.StatusUnprocessableEntity, Message: "Could not auto-create admin user. Add optional/default fields or create one manually."}
	}
	userID := user[r.authUser.PrimaryKey]
	return r.issueAuthCode(w, email, userID, "First admin user created. A login code was sent.")
}

func (r *Runtime) issueAuthCode(w http.ResponseWriter, email string, userID any, message string) error {
	code, err := randomCode6()
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	expiresAt := now + int64(r.App.Auth.CodeTTLMinutes)*60_000
	_, err = r.DB.Exec(`INSERT INTO belm_auth_codes (email, user_id, code, expires_at, used, created_at) VALUES (?, ?, ?, ?, 0, ?)`, email, userID, code, expiresAt, now)
	if err != nil {
		return err
	}
	if err := r.deliverEmailCode(email, code); err != nil {
		return err
	}

	resp := map[string]any{"ok": true, "message": message}
	if r.App.Auth.DevExposeCode {
		resp["devCode"] = code
	}
	r.writeJSON(w, http.StatusOK, resp)
	return nil
}

// loadOrCreateAuthUserForRequestCode loads an auth user by email or auto-creates it when possible.
// When no auth users exist yet, bootstrap-admin must be used to create the first admin user.
func (r *Runtime) loadOrCreateAuthUserForRequestCode(email string) (map[string]any, bool, error) {
	user, found, err := r.loadAuthUserByEmail(email)
	if err != nil || found {
		return user, found, err
	}

	totalUsers, err := r.countAuthUsers()
	if err != nil {
		return nil, false, err
	}
	if totalUsers == 0 {
		return nil, false, &apiError{
			Status:  http.StatusConflict,
			Message: "No users exist yet. Create the first admin using /_belm/bootstrap-admin",
		}
	}
	return r.tryAutoCreateAuthUser(email)
}

// tryAutoCreateAuthUser creates a minimal auth user for passwordless first-login flows.
// It only succeeds when all required fields can be safely inferred from auth config.
func (r *Runtime) tryAutoCreateAuthUser(email string) (map[string]any, bool, error) {
	return r.tryAutoCreateAuthUserWithRole(email, "user")
}

func (r *Runtime) tryAutoCreateAuthUserWithRole(email, roleValue string) (map[string]any, bool, error) {
	if r.authUser == nil {
		return nil, false, nil
	}

	columns := make([]string, 0, len(r.authUser.Fields))
	placeholders := make([]string, 0, len(r.authUser.Fields))
	values := make([]any, 0, len(r.authUser.Fields))
	ctx := entityNullContext(r.authUser)
	hasEmailField := false

	for _, field := range r.authUser.Fields {
		if field.Primary && field.Auto {
			continue
		}

		quoted, err := quoteIdentifier(field.Name)
		if err != nil {
			return nil, false, err
		}

		switch {
		case field.Name == r.App.Auth.EmailField:
			columns = append(columns, quoted)
			placeholders = append(placeholders, "?")
			values = append(values, email)
			ctx[field.Name] = email
			hasEmailField = true
		case r.App.Auth.RoleField != "" && field.Name == r.App.Auth.RoleField:
			if field.Type != "String" {
				return nil, false, nil
			}
			columns = append(columns, quoted)
			placeholders = append(placeholders, "?")
			values = append(values, roleValue)
			ctx[field.Name] = roleValue
		case field.Optional:
			// Keep optional fields nil for auto-provisioned users.
		default:
			// Required field that cannot be inferred automatically.
			return nil, false, nil
		}
	}

	if !hasEmailField || len(columns) == 0 {
		return nil, false, nil
	}

	if err := r.validateEntityRules(r.authUser, ctx); err != nil {
		return nil, false, nil
	}

	table, err := quoteIdentifier(r.authUser.Table)
	if err != nil {
		return nil, false, err
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	if _, err := r.DB.Exec(insertSQL, values...); err != nil {
		// If a concurrent request created the same user, load and continue.
		user, found, loadErr := r.loadAuthUserByEmail(email)
		if loadErr == nil && found {
			return user, true, nil
		}
		return nil, false, err
	}

	user, found, err := r.loadAuthUserByEmail(email)
	if err != nil {
		return nil, false, err
	}
	return user, found, nil
}

// handleAuthLogin verifies an email+code pair and issues a session token.
func (r *Runtime) handleAuthLogin(w http.ResponseWriter, payload map[string]any) error {
	if !r.authEnabled() {
		return &apiError{Status: http.StatusNotFound, Message: "Authentication is not enabled"}
	}
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

	row, ok, err := queryRow(r.DB, `SELECT id, user_id, code, expires_at, used FROM belm_auth_codes WHERE email = ? ORDER BY id DESC LIMIT 1`, email)
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
	userID := row["user_id"]

	if _, err := r.DB.Exec(`UPDATE belm_auth_codes SET used = 1 WHERE id = ?`, codeID); err != nil {
		return err
	}
	userRow, found, err := r.loadAuthUserByID(userID)
	if err != nil {
		return err
	}
	if !found {
		return &apiError{Status: http.StatusUnauthorized, Message: "Invalid or expired code"}
	}
	decodedUser := decodeEntityRow(r.authUser, userRow)

	token, err := randomToken(32)
	if err != nil {
		return err
	}
	sessionExpiresAt := now + int64(r.App.Auth.SessionTTLHours)*60*60*1000
	if _, err := r.DB.Exec(`INSERT INTO belm_sessions (token, user_id, email, expires_at, revoked, created_at) VALUES (?, ?, ?, ?, 0, ?)`, token, userID, email, sessionExpiresAt, now); err != nil {
		return err
	}

	r.writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"token":     token,
		"expiresAt": sessionExpiresAt,
		"user":      decodedUser,
	})
	return nil
}

// handleAuthLogout revokes the caller session token.
func (r *Runtime) handleAuthLogout(w http.ResponseWriter, auth authSession) error {
	if !r.authEnabled() {
		return &apiError{Status: http.StatusNotFound, Message: "Authentication is not enabled"}
	}
	if !auth.Authenticated || auth.Token == "" {
		return &apiError{Status: http.StatusUnauthorized, Message: "Authentication required"}
	}
	if _, err := r.DB.Exec(`UPDATE belm_sessions SET revoked = 1 WHERE token = ?`, auth.Token); err != nil {
		return err
	}
	r.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	return nil
}

// deliverEmailCode dispatches login codes through the configured transport.
func (r *Runtime) deliverEmailCode(toEmail, code string) error {
	if r.App.Auth.DevExposeCode {
		r.printAuthLogHeader()
		useColor := supportsANSI()
		fmt.Printf("  %s %s  %s %s\n",
			colorize(useColor, ansiLabel, "Dev code:"),
			colorize(useColor, ansiCommand, code),
			colorize(useColor, ansiLabel, "Email:"),
			toEmail,
		)
	}

	switch r.App.Auth.EmailTransport {
	case "console":
		r.printAuthLogHeader()
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
		if err := sendWithSendmail(r.App.Auth.SendmailPath, r.App.Auth.EmailFrom, r.App.Auth.EmailSubject, toEmail, code, r.App.Auth.CodeTTLMinutes); err != nil {
			return err
		}
		r.printAuthLogHeader()
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
		return fmt.Errorf("unsupported email transport %q", r.App.Auth.EmailTransport)
	}
}

func (r *Runtime) printAuthLogHeader() {
	useColor := supportsANSI()
	r.authLogOnce.Do(func() {
		fmt.Println()
		fmt.Printf("%s\n", colorize(useColor, ansiSection, "Auth logs"))
	})
}

// sendWithSendmail sends plain-text email by invoking the local sendmail binary.
func sendWithSendmail(sendmailPath, from, subject, to, code string, ttlMinutes int) error {
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Your login code is:",
		code,
		"",
		fmt.Sprintf("This code expires in %d minute(s).", ttlMinutes),
		"",
		"If you did not request this code, ignore this email.",
	}, "\n")

	cmd := exec.Command(sendmailPath, "-t", "-i")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := stdin.Write([]byte(msg)); err != nil {
		_ = stdin.Close()
		return err
	}
	if err := stdin.Close(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// randomCode6 returns a zero-padded 6-digit cryptographically random code.
func randomCode6() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// randomToken returns a hex-encoded cryptographically random token.
func randomToken(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
