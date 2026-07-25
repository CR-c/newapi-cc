package ratio_setting

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

var groupRatioMap = types.NewRWMap[string, float64]()

// groupRealSalesRatioMap stores optional platform settlement ratios used by
// profit analysis. When a group is absent, profit uses the billing group ratio.
var groupRealSalesRatioMap = types.NewRWMap[string, float64]()

var defaultGroupGroupRatio = map[string]map[string]float64{
	"vip": {
		"edit_this": 0.9,
	},
}

var groupGroupRatioMap = types.NewRWMap[string, map[string]float64]()

var defaultGroupSpecialUsableGroup = map[string]map[string]string{
	"vip": {
		"append_1":   "vip_special_group_1",
		"-:remove_1": "vip_removed_group_1",
	},
}

var defaultGroupModelPrice = map[string]map[string]float64{}

type GroupRatioSetting struct {
	GroupRatio              *types.RWMap[string, float64]            `json:"group_ratio"`
	GroupGroupRatio         *types.RWMap[string, map[string]float64] `json:"group_group_ratio"`
	GroupSpecialUsableGroup *types.RWMap[string, map[string]string]  `json:"group_special_usable_group"`
	GroupModelPrice         *types.RWMap[string, map[string]float64] `json:"group_model_price"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	groupSpecialUsableGroup := types.NewRWMap[string, map[string]string]()
	groupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)

	groupRatioMap.AddAll(defaultGroupRatio)
	groupGroupRatioMap.AddAll(defaultGroupGroupRatio)
	groupModelPrice := types.NewRWMap[string, map[string]float64]()
	groupModelPrice.AddAll(defaultGroupModelPrice)

	groupRatioSetting = GroupRatioSetting{
		GroupSpecialUsableGroup: groupSpecialUsableGroup,
		GroupRatio:              groupRatioMap,
		GroupGroupRatio:         groupGroupRatioMap,
		GroupModelPrice:         groupModelPrice,
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func GetGroupRatioSetting() *GroupRatioSetting {
	if groupRatioSetting.GroupSpecialUsableGroup == nil {
		groupRatioSetting.GroupSpecialUsableGroup = types.NewRWMap[string, map[string]string]()
		groupRatioSetting.GroupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)
	}
	if groupRatioSetting.GroupModelPrice == nil {
		groupRatioSetting.GroupModelPrice = types.NewRWMap[string, map[string]float64]()
		groupRatioSetting.GroupModelPrice.AddAll(defaultGroupModelPrice)
	}
	return &groupRatioSetting
}

func GetGroupRatioCopy() map[string]float64 {
	return groupRatioMap.ReadAll()
}

func ContainsGroupRatio(name string) bool {
	_, ok := groupRatioMap.Get(name)
	return ok
}

func GroupRatio2JSONString() string {
	return groupRatioMap.MarshalJSONString()
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupRatioMap, jsonStr)
}

func GetGroupRatio(name string) float64 {
	ratio, ok := groupRatioMap.Get(name)
	if !ok {
		common.SysLog("group ratio not found: " + name)
		return 1
	}
	return ratio
}

func GetGroupGroupRatio(userGroup, usingGroup string) (float64, bool) {
	gp, ok := groupGroupRatioMap.Get(userGroup)
	if !ok {
		return -1, false
	}
	ratio, ok := gp[usingGroup]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func GroupGroupRatio2JSONString() string {
	return groupGroupRatioMap.MarshalJSONString()
}

func UpdateGroupGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupGroupRatioMap, jsonStr)
}

func GetGroupModelPrice(group, modelName string) (float64, bool) {
	prices, ok := GetGroupRatioSetting().GroupModelPrice.Get(group)
	if !ok {
		return 0, false
	}
	price, ok := prices[FormatMatchingModelName(modelName)]
	if !ok || price < 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return 0, false
	}
	return price, true
}

func GroupModelPrice2JSONString() string {
	return GetGroupRatioSetting().GroupModelPrice.MarshalJSONString()
}

func UpdateGroupModelPriceByJSONString(jsonStr string) error {
	if err := ValidateGroupModelPrice(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonStringWithCallback(
		GetGroupRatioSetting().GroupModelPrice,
		jsonStr,
		InvalidateExposedDataCache,
	)
}

func ValidateGroupModelPrice(jsonStr string) error {
	prices := make(map[string]map[string]float64)
	if err := common.Unmarshal([]byte(jsonStr), &prices); err != nil {
		return err
	}
	for group, modelPrices := range prices {
		if strings.TrimSpace(group) == "" {
			return errors.New("group model price contains an empty group")
		}
		for modelName, price := range modelPrices {
			if strings.TrimSpace(modelName) == "" {
				return fmt.Errorf("group model price for %s contains an empty model", group)
			}
			if price < 0 || math.IsNaN(price) || math.IsInf(price, 0) {
				return fmt.Errorf("group model price for %s/%s must be finite and not less than 0", group, modelName)
			}
		}
	}
	return nil
}

func CheckGroupRatio(jsonStr string) error {
	checkGroupRatio := make(map[string]float64)
	err := json.Unmarshal([]byte(jsonStr), &checkGroupRatio)
	if err != nil {
		return err
	}
	for name, ratio := range checkGroupRatio {
		if ratio < 0 {
			return errors.New("group ratio must be not less than 0: " + name)
		}
	}
	return nil
}

func GroupRealSalesRatio2JSONString() string {
	return groupRealSalesRatioMap.MarshalJSONString()
}

func UpdateGroupRealSalesRatioByJSONString(jsonStr string) error {
	if strings.TrimSpace(jsonStr) == "" {
		jsonStr = "{}"
	}
	if err := CheckGroupRatio(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonString(groupRealSalesRatioMap, jsonStr)
}

// GetGroupRealSalesRatio returns the configured real sales ratio for profit
// analysis. ok is false when the group has no explicit override.
func GetGroupRealSalesRatio(name string) (float64, bool) {
	ratio, ok := groupRealSalesRatioMap.Get(name)
	if !ok {
		return 0, false
	}
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0, false
	}
	return ratio, true
}

// ResolveGroupRealSalesRatio returns the ratio used for platform profit revenue.
// Unset or invalid real-sales values fall back to billingGroupRatio (the group's
// billing ratio from the log snapshot or group config).
func ResolveGroupRealSalesRatio(groupName string, billingGroupRatio float64) float64 {
	if ratio, ok := GetGroupRealSalesRatio(groupName); ok {
		return ratio
	}
	if billingGroupRatio > 0 && !math.IsNaN(billingGroupRatio) && !math.IsInf(billingGroupRatio, 0) {
		return billingGroupRatio
	}
	if groupName != "" {
		return GetGroupRatio(groupName)
	}
	return 1
}
