package ratio_setting

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

const (
	AutoRouteSequential = "sequential"
	AutoRouteRandom     = "random"
	AutoRouteNearest    = "nearest"
	NoModelSpecified    = "specified"
)

type GroupPolicy struct {
	Endpoint        string `json:"endpoint"`
	RequireKey      *bool  `json:"require_key,omitempty"`
	AutoRoute       bool   `json:"auto_route"`
	RouteStrategy   string `json:"route_strategy,omitempty"`
	FallbackTimes   *int   `json:"fallback_times,omitempty"`
	FallbackGroup   string `json:"fallback_group,omitempty"`
	FallbackModel   string `json:"fallback_model,omitempty"`
	IgnoreModel     bool   `json:"ignore_model,omitempty"`
	AllowNoModel    bool   `json:"allow_no_model,omitempty"`
	NoModelStrategy string `json:"no_model_strategy,omitempty"`
	NoModel         string `json:"no_model,omitempty"`
}

var groupPolicyMap = types.NewRWMap[string, GroupPolicy]()

func GetGroupPolicy(group string) GroupPolicy {
	policy, ok := groupPolicyMap.Get(group)
	if !ok {
		return GroupPolicy{Endpoint: "/", RequireKey: common.GetPointer(true), RouteStrategy: AutoRouteSequential}
	}
	if policy.Endpoint == "" {
		policy.Endpoint = "/"
	}
	if policy.RequireKey == nil {
		policy.RequireKey = common.GetPointer(true)
	}
	if policy.RouteStrategy == "" {
		policy.RouteStrategy = AutoRouteSequential
	}
	return policy
}

func GroupPolicies2JSONString() string {
	return groupPolicyMap.MarshalJSONString()
}

func ValidateGroupPoliciesJSON(value string) error {
	var policies map[string]GroupPolicy
	if err := common.Unmarshal([]byte(value), &policies); err != nil {
		return err
	}

	for group, policy := range policies {
		if strings.TrimSpace(group) == "" {
			return fmt.Errorf("group policy name must not be empty")
		}
		endpoint := normalizeGroupEndpoint(policy.Endpoint)
		if !validGroupEndpoint(endpoint) {
			return fmt.Errorf("invalid endpoint for group %s", group)
		}
		if policy.RouteStrategy != "" && policy.RouteStrategy != AutoRouteSequential && policy.RouteStrategy != AutoRouteRandom && policy.RouteStrategy != AutoRouteNearest {
			return fmt.Errorf("invalid auto route strategy for group %s", group)
		}
		if policy.FallbackTimes != nil && *policy.FallbackTimes < 0 {
			return fmt.Errorf("fallback times must not be negative for group %s", group)
		}
		if policy.FallbackGroup == group {
			return fmt.Errorf("fallback group must differ from group %s", group)
		}
		if policy.NoModelStrategy != "" && policy.NoModelStrategy != AutoRouteSequential && policy.NoModelStrategy != AutoRouteRandom && policy.NoModelStrategy != NoModelSpecified {
			return fmt.Errorf("invalid no-model strategy for group %s", group)
		}
		if policy.NoModelStrategy == NoModelSpecified && strings.TrimSpace(policy.NoModel) == "" {
			return fmt.Errorf("specified no-model strategy requires a model for group %s", group)
		}
		policies[group] = GroupPolicy{
			Endpoint: endpoint, RequireKey: policy.RequireKey, AutoRoute: policy.AutoRoute,
			RouteStrategy: policy.RouteStrategy, FallbackTimes: policy.FallbackTimes,
			FallbackGroup: strings.TrimSpace(policy.FallbackGroup), FallbackModel: strings.TrimSpace(policy.FallbackModel),
			IgnoreModel: policy.IgnoreModel, AllowNoModel: policy.AllowNoModel,
			NoModelStrategy: policy.NoModelStrategy, NoModel: strings.TrimSpace(policy.NoModel),
		}
	}
	return nil
}

func UpdateGroupPoliciesByJSONString(value string) error {
	if err := ValidateGroupPoliciesJSON(value); err != nil {
		return err
	}
	return types.LoadFromJsonString(groupPolicyMap, value)
}

// NormalizeGroupEndpointPath maps an entry alias to its canonical relay path.
// The matched endpoint is returned separately; it never identifies a group.
func NormalizeGroupEndpointPath(path string) (string, string, bool) {
	endpoint := groupEndpointForPath(path)
	if endpoint == "" {
		return path, "", false
	}
	if endpoint == "/" {
		if path == "/" {
			return "/v1", endpoint, true
		}
		return path, endpoint, true
	}

	remainder := strings.TrimPrefix(path, endpoint)
	if remainder == "" || remainder == "/" {
		return "/v1", endpoint, true
	}
	if remainder == "/v1" ||
		strings.HasPrefix(remainder, "/v1/") ||
		remainder == "/v1beta" ||
		strings.HasPrefix(remainder, "/v1beta/") {
		return remainder, endpoint, true
	}
	return "/v1" + remainder, endpoint, true
}

func groupEndpointForPath(path string) string {
	endpoints := make(map[string]struct{})
	for _, policy := range groupPolicyMap.ReadAll() {
		endpoint := normalizeGroupEndpoint(policy.Endpoint)
		if groupEndpointMatches(path, endpoint) {
			endpoints[endpoint] = struct{}{}
		}
	}
	if _, ok := endpoints["/"]; ok {
		return "/"
	}
	if len(endpoints) == 0 {
		if groupEndpointMatches(path, "/") {
			return "/"
		}
		return ""
	}
	matched := make([]string, 0, len(endpoints))
	for endpoint := range endpoints {
		matched = append(matched, endpoint)
	}
	sort.Slice(matched, func(i, j int) bool {
		if len(matched[i]) == len(matched[j]) {
			return matched[i] < matched[j]
		}
		return len(matched[i]) > len(matched[j])
	})
	return matched[0]
}

func GroupEndpointMatches(group, endpoint string) bool {
	return normalizeGroupEndpoint(GetGroupPolicy(group).Endpoint) == normalizeGroupEndpoint(endpoint)
}

func GroupPolicyForEndpoint(group, endpoint string) (GroupPolicy, bool) {
	policy := GetGroupPolicy(group)
	return policy, normalizeGroupEndpoint(policy.Endpoint) == normalizeGroupEndpoint(endpoint)
}

func GroupPoliciesForEndpoint(endpoint string) []GroupPolicy {
	endpoint = normalizeGroupEndpoint(endpoint)
	policies := make([]GroupPolicy, 0)
	for group, policy := range groupPolicyMap.ReadAll() {
		if normalizeGroupEndpoint(policy.Endpoint) == endpoint {
			policies = append(policies, GetGroupPolicy(group))
		}
	}
	return policies
}

func normalizeGroupEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || endpoint == "/" {
		return "/"
	}
	return strings.TrimRight(endpoint, "/")
}

func validGroupEndpoint(endpoint string) bool {
	if endpoint == "/" {
		return true
	}
	if !strings.HasPrefix(endpoint, "/") || strings.ContainsAny(endpoint, "?#\\") || strings.Contains(endpoint, "//") || strings.Contains(endpoint, "..") {
		return false
	}
	parsed, err := url.ParseRequestURI(endpoint)
	return err == nil && parsed.Path == endpoint && parsed.RawQuery == "" && parsed.Fragment == ""
}

func groupEndpointMatches(path, endpoint string) bool {
	endpoint = normalizeGroupEndpoint(endpoint)
	if endpoint == "/" {
		return path == "/" || path == "/v1" || strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/v1beta/") || path == "/v1beta"
	}
	return path == endpoint || path == endpoint+"/v1" || strings.HasPrefix(path, endpoint+"/")
}
