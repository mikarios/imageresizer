package httphelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/exceptions"
)

func RespondJSON(ctx context.Context, w http.ResponseWriter, code int, payload interface{}) {
	if err := sendResponse(w, code, payload); err != nil {
		logger.Error(ctx, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func sendResponse(w http.ResponseWriter, code int, payload interface{}) error {
	if w == nil {
		return exceptions.ErrInvalidResponseWriter
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("could not unmarshal payload %w", err)
		}

		_, err = w.Write(payloadJSON)
		if err != nil {
			return fmt.Errorf("could not write to responseWriter: %w", err)
		}
	}

	return nil
}
