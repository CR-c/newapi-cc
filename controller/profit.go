package controller

import (
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func parseProfitQuery(c *gin.Context) (model.ProfitQuery, error) {
	query := model.ProfitQuery{
		ModelName: c.Query("model"),
		Group:     c.Query("group"),
	}
	fields := []struct {
		name   string
		target *int64
	}{
		{name: "start_timestamp", target: &query.StartTimestamp},
		{name: "end_timestamp", target: &query.EndTimestamp},
	}
	for _, field := range fields {
		value := c.Query(field.name)
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 0 {
			return query, strconv.ErrSyntax
		}
		*field.target = parsed
	}
	intFields := []struct {
		name   string
		target *int
	}{
		{name: "user_id", target: &query.UserId},
		{name: "channel", target: &query.ChannelId},
	}
	for _, field := range intFields {
		value := c.Query(field.name)
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return query, strconv.ErrSyntax
		}
		*field.target = parsed
	}
	if query.EndTimestamp > 0 && query.StartTimestamp > query.EndTimestamp {
		return query, strconv.ErrSyntax
	}
	return query, nil
}

func GetProfitOverview(c *gin.Context) {
	query, err := parseProfitQuery(c)
	if err != nil {
		common.ApiErrorMsg(c, "invalid profit query")
		return
	}
	generation, err := model.GetProfitAnalysisGeneration()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	query.Generation = &generation
	summary, err := model.GetProfitAggregate(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	dimensions := []string{"user", "model", "group", "channel"}
	breakdowns := make(map[string][]*model.ProfitAggregate, len(dimensions))
	for _, dimension := range dimensions {
		rows, breakdownErr := model.GetProfitBreakdown(query, dimension)
		if breakdownErr != nil {
			common.ApiError(c, breakdownErr)
			return
		}
		breakdowns[dimension] = rows
	}
	enrichProfitBreakdowns(breakdowns)
	common.ApiSuccess(c, gin.H{
		"summary":    summary,
		"by_user":    breakdowns["user"],
		"by_model":   breakdowns["model"],
		"by_group":   breakdowns["group"],
		"by_channel": breakdowns["channel"],
	})
}

func enrichProfitBreakdowns(breakdowns map[string][]*model.ProfitAggregate) {
	userRows := breakdowns["user"]
	if len(userRows) > 0 {
		userIds := make([]int, 0, len(userRows))
		for _, row := range userRows {
			userIds = append(userIds, row.UserId)
		}
		users := make([]model.User, 0, len(userIds))
		if err := model.DB.Select("id", "username").Where("id IN ?", userIds).Find(&users).Error; err == nil {
			names := make(map[int]string, len(users))
			for _, user := range users {
				names[user.Id] = user.Username
			}
			for _, row := range userRows {
				row.Username = names[row.UserId]
			}
		}
	}

	channelRows := breakdowns["channel"]
	if len(channelRows) == 0 {
		return
	}
	channelIds := make([]int, 0, len(channelRows))
	for _, row := range channelRows {
		channelIds = append(channelIds, row.ChannelId)
	}
	channels := make([]model.Channel, 0, len(channelIds))
	if err := model.DB.Select("id", "name").Where("id IN ?", channelIds).Find(&channels).Error; err != nil {
		return
	}
	names := make(map[int]string, len(channels))
	for _, channel := range channels {
		names[channel.Id] = channel.Name
	}
	for _, row := range channelRows {
		row.ChannelName = names[row.ChannelId]
	}
}

func GetModelCostRules(c *gin.Context) {
	rules, err := model.GetModelCostRules()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rules)
}

func GetProfitCostModels(c *gin.Context) {
	modelNames, err := model.GetProfitCostModelNames()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, modelNames)
}

func SaveModelCostRule(c *gin.Context) {
	var input model.ModelCostRule
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiErrorMsg(c, "invalid cost rule")
		return
	}
	rule, err := model.SaveModelCostRule(&input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rule)
}

func BackfillProfitRecords(c *gin.Context) {
	if err := model.BackfillProfitRecords(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ResetProfitAnalysisData(c *gin.Context) {
	resetAt := time.Now().Unix()
	if err := model.ResetProfitAnalysisData(resetAt); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"reset_at": resetAt})
}
