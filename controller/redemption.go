package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func validateRedemptionQuota(quota int) error {
	if quota <= 0 || quota > common.MaxQuota {
		return errors.New("兑换码额度必须大于 0 且不能超过系统额度上限")
	}
	return nil
}

func normalizeNewRedemptionFundingSource(source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		source = model.RedemptionFundingSourcePaid
	}
	if err := model.ValidateNewRedemptionFundingSource(source); err != nil {
		return "", err
	}
	return source, nil
}

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.GetAllRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchRedemptions(keyword, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
	return
}

func AddRedemption(c *gin.Context) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
		return
	}
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgAuthInsufficientPrivilege)
		return
	}

	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionNameLength)
		return
	}
	if redemption.Count <= 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
		return
	}
	if redemption.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	if err := validateRedemptionQuota(redemption.Quota); err != nil {
		common.ApiError(c, err)
		return
	}
	fundingSource, err := normalizeNewRedemptionFundingSource(redemption.FundingSource)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	keys := make([]string, 0, redemption.Count)
	redemptions := make([]model.Redemption, 0, redemption.Count)
	for i := 0; i < redemption.Count; i++ {
		key := common.GetUUID()
		redemptions = append(redemptions, model.Redemption{
			UserId:        c.GetInt("id"),
			Name:          redemption.Name,
			Key:           key,
			CreatedTime:   common.GetTimestamp(),
			Quota:         redemption.Quota,
			ExpiredTime:   redemption.ExpiredTime,
			FundingSource: fundingSource,
		})
		keys = append(keys, key)
	}
	if err := model.InsertRedemptions(redemptions); err != nil {
		common.SysError("failed to insert redemptions: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgRedemptionCreateFailed),
			"data":    []string{},
		})
		return
	}
	recordManageAudit(c, "redemption.create", map[string]interface{}{
		"name":           redemption.Name,
		"count":          redemption.Count,
		"quota":          logger.LogQuota(redemption.Quota),
		"funding_source": fundingSource,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
	return
}

func DeleteRedemption(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgAuthInsufficientPrivilege)
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	err = model.DeleteRedemptionById(id, redemption.FundingSource, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateRedemption(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgAuthInsufficientPrivilege)
		return
	}
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanRedemption, err := model.GetRedemptionById(redemption.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.ValidateNewRedemptionFundingSource(cleanRedemption.FundingSource); err != nil {
		common.ApiError(c, model.ErrRedemptionStateChanged)
		return
	}
	expectedFundingSource := cleanRedemption.FundingSource
	requestedFundingSource := strings.TrimSpace(redemption.FundingSource)
	if requestedFundingSource != "" {
		if err := model.ValidateNewRedemptionFundingSource(requestedFundingSource); err != nil {
			common.ApiError(c, err)
			return
		}
		if requestedFundingSource != cleanRedemption.FundingSource {
			common.ApiError(c, model.ErrRedemptionStateChanged)
			return
		}
	}
	expectedStatus := cleanRedemption.Status
	if statusOnly == "" {
		if err := validateRedemptionQuota(redemption.Quota); err != nil {
			common.ApiError(c, err)
			return
		}
		if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		cleanRedemption.Name = redemption.Name
		cleanRedemption.Quota = redemption.Quota
		cleanRedemption.ExpiredTime = redemption.ExpiredTime
	}
	if statusOnly != "" {
		if expectedStatus == common.RedemptionCodeStatusUsed ||
			(redemption.Status != common.RedemptionCodeStatusEnabled && redemption.Status != common.RedemptionCodeStatusDisabled) {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
		err = model.UpdateRedemptionStatus(
			cleanRedemption.Id,
			expectedStatus,
			redemption.Status,
			expectedFundingSource,
			true,
		)
		cleanRedemption.Status = redemption.Status
	} else {
		err = cleanRedemption.UpdateDetails(expectedStatus, expectedFundingSource)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, cleanRedemption.Id, "redemption.update", map[string]interface{}{
		"funding_source": cleanRedemption.FundingSource,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
	return
}

func DeleteInvalidRedemption(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgAuthInsufficientPrivilege)
		return
	}
	rows, err := model.DeleteInvalidRedemptions(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}
