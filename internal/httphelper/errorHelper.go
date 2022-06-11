package httphelper

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/exceptions"
)

func LogAndRespondErr(ctx context.Context, w http.ResponseWriter, httpError, err error, logMessages ...interface{}) {
	logger.Error(ctx, err, logMessages)
	respondWithError(ctx, w, httpError)
}

func respondWithError(ctx context.Context, w http.ResponseWriter, err error) {
	type ErrResp struct {
		Error         string   `json:"error,omitempty"`
		TransactionID string   `json:"transactionID,omitempty"`
		FailedIDs     []string `json:"failedIDs,omitempty"`
	}

	transactionID := fmt.Sprint(ctx.Value(logger.Settings.TransactionKey))
	errResp := &ErrResp{Error: err.Error(), TransactionID: transactionID}

	switch {
	case oneOf(err, exceptions.ErrInvalidJobPriority):
		RespondJSON(ctx, w, http.StatusBadRequest, errResp)
	case oneOf(err, exceptions.ErrUnauthorised):
		RespondJSON(ctx, w, http.StatusUnauthorized, errResp)
	default:
		errResp.Error = exceptions.ErrInternalServerError.Error()
		RespondJSON(ctx, w, http.StatusInternalServerError, errResp)
	}
}

func oneOf(err error, errs ...error) bool {
	for _, errr := range errs {
		if errors.Is(err, errr) {
			return true
		}
	}

	return false
}
