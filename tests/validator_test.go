package tests

import (
	"testing"

	"novaflow/core"
)

type TestUser struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18,max=120"`
	Score float64 `validate:"numeric"`
}

func TestValidateStruct(t *testing.T) {
	v := core.NewValidator(nil)

	user := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
		Score: 95.5,
	}

	v.ValidateStruct(user)
	if !v.Passes() {
		t.Fatalf("expected validation to pass, got errors: %v", v.Errors())
	}

	invalidUser := TestUser{
		Name:  "",
		Email: "invalid-email",
		Age:   15,
	}

	v2 := core.NewValidator(nil)
	v2.ValidateStruct(invalidUser)
	if v2.Passes() {
		t.Fatal("expected validation to fail")
	}

	errs := v2.Errors()
	if _, ok := errs["name"]; !ok {
		t.Error("expected error for required name")
	}
	if _, ok := errs["email"]; !ok {
		t.Error("expected error for invalid email")
	}
	if _, ok := errs["age"]; !ok {
		t.Error("expected error for age < 18")
	}
}
