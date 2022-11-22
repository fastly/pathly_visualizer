package rest_api

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

var errMessageTooLong = errors.New("message is too long")

func readJsonRequestBody[T any](ctx *gin.Context, limit int) (value T, ok bool) {
	requestBytes, err := readAtMost(ctx.Request.Body, limit)
	if err != nil {
		if err == errMessageTooLong {
			ctx.String(http.StatusBadRequest, "Request too long\n")
		} else {
			ctx.Status(http.StatusInternalServerError)
			_ = ctx.Error(err)
		}
		return
	}

	if err := json.Unmarshal(requestBytes, &value); err != nil {
		ctx.String(http.StatusBadRequest, "Request is not valid JSON: %s\n", err.Error())
		return
	}

	ok = true
	return
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func readAtMost(r io.Reader, limit int) ([]byte, error) {
	b := make([]byte, 0, min(512, limit))
	for {
		if len(b) >= limit {
			return b, errMessageTooLong
		}
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):min(limit, cap(b))])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}
