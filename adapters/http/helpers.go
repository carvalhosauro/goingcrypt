package http

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = func() *validator.Validate {
	v := validator.New()
	// use the json tag name instead of the struct field name
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}()

type errorResponse struct {
	Error string `json:"error"`
}

type validationErrorResponse struct {
	Errors map[string]string `json:"errors"`
}

func readBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func validateStruct(v any) map[string]string {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return map[string]string{"_": "invalid input"}
	}
	errs := make(map[string]string, len(ve))
	for _, fe := range ve {
		errs[fe.Field()] = fe.Tag()
	}
	return errs
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeValidationError(w http.ResponseWriter, errs map[string]string) {
	writeJSON(w, http.StatusBadRequest, validationErrorResponse{Errors: errs})
}

// returns the client ip address from r.RemoteAddr
// proxy headers (X-Forwarded-For, X-Real-IP) are used only as fallback
// and should be treated as untrustworthy without proxy validation
func extractIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.SplitN(xff, ",", 2)[0]
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
