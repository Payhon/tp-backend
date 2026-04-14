package service

import (
	"testing"

	"project/internal/model"

	"gorm.io/gorm"
)

func TestResolvePasswordLoginEmailReturnsEmailInput(t *testing.T) {
	userSvc := &User{}

	email, err := userSvc.ResolvePasswordLoginEmail("demo@example.com")
	if err != nil {
		t.Fatalf("ResolvePasswordLoginEmail returned error: %v", err)
	}
	if email != "demo@example.com" {
		t.Fatalf("expected email to stay unchanged, got %q", email)
	}
}

func TestResolvePasswordLoginEmailRejectsInvalidNumericInput(t *testing.T) {
	userSvc := &User{}

	if _, err := userSvc.ResolvePasswordLoginEmail("1300661500"); err == nil {
		t.Fatal("expected invalid numeric identifier to be rejected")
	}
}

func TestResolvePasswordLoginEmailPrefersUsernameOverPhone(t *testing.T) {
	input := "13006617500"
	userSvc := &User{}

	origGetByUsername := dalGetUsersByUsername
	origGetEmailByPhone := userGetUserEmailByPhoneNumber
	defer func() {
		dalGetUsersByUsername = origGetByUsername
		userGetUserEmailByPhoneNumber = origGetEmailByPhone
	}()

	dalGetUsersByUsername = func(username string) (*model.User, error) {
		if username != input {
			t.Fatalf("unexpected username lookup: %s", username)
		}
		return &model.User{Email: "username@app.local"}, nil
	}
	userGetUserEmailByPhoneNumber = func(_ *User, phoneNumber string) (string, error) {
		t.Fatalf("phone fallback should not be called, got %s", phoneNumber)
		return "", nil
	}

	email, err := userSvc.ResolvePasswordLoginEmail(input)
	if err != nil {
		t.Fatalf("ResolvePasswordLoginEmail returned error: %v", err)
	}
	if email != "username@app.local" {
		t.Fatalf("expected username email, got %q", email)
	}
}

func TestResolvePasswordLoginEmailFallsBackToPhone(t *testing.T) {
	input := "13006617500"
	userSvc := &User{}

	origGetByUsername := dalGetUsersByUsername
	origGetEmailByPhone := userGetUserEmailByPhoneNumber
	defer func() {
		dalGetUsersByUsername = origGetByUsername
		userGetUserEmailByPhoneNumber = origGetEmailByPhone
	}()

	dalGetUsersByUsername = func(username string) (*model.User, error) {
		if username != input {
			t.Fatalf("unexpected username lookup: %s", username)
		}
		return nil, gorm.ErrRecordNotFound
	}
	userGetUserEmailByPhoneNumber = func(_ *User, phoneNumber string) (string, error) {
		if phoneNumber != input {
			t.Fatalf("unexpected phone lookup: %s", phoneNumber)
		}
		return "phone@app.local", nil
	}

	email, err := userSvc.ResolvePasswordLoginEmail(input)
	if err != nil {
		t.Fatalf("ResolvePasswordLoginEmail returned error: %v", err)
	}
	if email != "phone@app.local" {
		t.Fatalf("expected phone email, got %q", email)
	}
}
