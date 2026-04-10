package service

import "strings"

func trimStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	vv := strings.TrimSpace(*v)
	if vv == "" {
		return nil
	}
	return &vv
}

func trimStringValue(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func isPlaceholderEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	return (strings.HasPrefix(email, "u_") || strings.HasPrefix(email, "org_")) && strings.HasSuffix(email, "@app.local")
}

func defaultUsernameForPhone(phone string) *string {
	return trimStringValue(phone)
}

func defaultUsernameForEmail(email string) *string {
	if isPlaceholderEmail(email) {
		return nil
	}
	return trimStringValue(email)
}

func defaultUsernameForBackoffice(email, phone string) *string {
	if username := defaultUsernameForEmail(email); username != nil {
		return username
	}
	return defaultUsernameForPhone(phone)
}

func sameUsernameAsValue(username *string, expected string) bool {
	if username == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(*username), strings.TrimSpace(expected))
}
