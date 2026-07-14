package controller

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func preparePlayground(c *gin.Context, relayFormat types.RelayFormat) *types.NewAPIError {
	if c.GetBool("use_access_token") {
		return types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, nil, nil)
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	userId := c.GetInt("id")
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	if err = middleware.SetupContextForToken(c, tempToken); err != nil {
		return types.NewError(err, types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}
	return nil
}

func respondPlaygroundError(c *gin.Context, newAPIError *types.NewAPIError) {
	c.JSON(newAPIError.StatusCode, gin.H{
		"error": newAPIError.ToOpenAIError(),
	})
}

func Playground(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatOpenAI); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	Relay(c, types.RelayFormatOpenAI)
}

func PlaygroundImage(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatOpenAIImage); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	Relay(c, types.RelayFormatOpenAIImage)
}

func PlaygroundVideo(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatTask); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	RelayTask(c)
}

func PlaygroundVideoFetch(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatTask); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	RelayTaskFetch(c)
}
