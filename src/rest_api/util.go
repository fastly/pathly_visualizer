package rest_api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/config"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"net/http"
)

func readJsonRequestBody[T any](ctx *gin.Context) (value T, ok bool) {
	requestSizeLimit := config.RequestByteLimit.GetInt()
	requestBytes, err := util.ReadAtMost(ctx.Request.Body, requestSizeLimit)
	if err != nil {
		if err == util.ErrMessageTooLong {
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
