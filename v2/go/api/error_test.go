package api_test

import (
	"errors"
	"fmt"
	"testing"

	convAPI "github.com/sofmon/convention/v2/go/api"
	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func Test_error(t *testing.T) {

	ctx := convCtx.New(convAuth.Claims{})

	err := convAPI.NewError(ctx, 404, "custom_code", "test error", nil)

	if !convAPI.ErrorHasCode(err, "custom_code") {
		t.Errorf("expected error to have code 'custom_code', got %s", err.(*convAPI.Error).Code)
	}

	if convAPI.ErrorHasCode(err, "wrong") {
		t.Error("expected error not to have code 'wrong'")
	}

	wrapped := fmt.Errorf("wrapped error: %w", err)
	if !convAPI.ErrorHasCode(wrapped, "custom_code") {
		t.Error("expected to catch error with code 'custom_code' in wrapped error")
	}

	err = errors.New("test error")

	if convAPI.ErrorHasCode(err, "wrong") {
		t.Error("expected error not to have code 'wrong'")
	}

}
