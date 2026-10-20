package middleware

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

type groupEndpointRequestContextKey struct{}

type GroupEndpointPolicy struct {
	Endpoint string
}

func GroupEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		if endpoint, ok := c.Request.Context().Value(groupEndpointRequestContextKey{}).(string); ok {
			common.SetContextKey(c, constant.ContextKeyGroupPolicy, GroupEndpointPolicy{Endpoint: endpoint})
			c.Next()
			return
		}
		if _, exists := c.Get(string(constant.ContextKeyGroupPolicy)); exists {
			c.Next()
			return
		}
		path, endpoint, ok := ratio_setting.NormalizeGroupEndpointPath(c.Request.URL.Path)
		if ok && path == c.Request.URL.Path {
			common.SetContextKey(c, constant.ContextKeyGroupPolicy, GroupEndpointPolicy{Endpoint: endpoint})
		}
		c.Next()
	}
}

func GroupEndpointDispatcher(router *gin.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		path, endpoint, ok := ratio_setting.NormalizeGroupEndpointPath(c.Request.URL.Path)
		if !ok || path == c.Request.URL.Path {
			c.Next()
			return
		}
		c.Request.URL.Path = path
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), groupEndpointRequestContextKey{}, endpoint))
		router.HandleContext(c)
	}
}

func keylessEndpointConfigured(endpoint string) bool {
	policies := ratio_setting.GroupPoliciesForEndpoint(endpoint)
	if len(policies) == 0 {
		return false
	}
	for _, policy := range policies {
		if policy.RequireKey == nil || *policy.RequireKey {
			return false
		}
	}
	return true
}

func uniqueEnabledRootOwner() (*model.UserBase, bool, error) {
	var owners []model.User
	if err := model.DB.Where("role = ? AND status = ?", common.RoleRootUser, common.UserStatusEnabled).Limit(2).Find(&owners).Error; err != nil {
		return nil, false, err
	}
	if len(owners) != 1 {
		return nil, false, nil
	}
	return owners[0].ToBaseUser(), true, nil
}

func trySetKeylessOwnerContext(c *gin.Context, endpoint string) (bool, error) {
	owner, ownerExists, err := uniqueEnabledRootOwner()
	if err != nil {
		if keylessEndpointConfigured(endpoint) {
			return false, err
		}
		return false, nil
	}
	if !ownerExists {
		if keylessEndpointConfigured(endpoint) {
			return false, errors.New("expected exactly one enabled root user")
		}
		return false, nil
	}
	policy, endpointMatches := ratio_setting.GroupPolicyForEndpoint(owner.Group, endpoint)
	if !endpointMatches || policy.RequireKey == nil || *policy.RequireKey {
		return false, nil
	}
	owner.WriteContext(c)
	c.Set("id", owner.Id)
	c.Set("token_id", 0)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, owner.Group)
	common.SetContextKey(c, constant.ContextKeyForcedFree, true)
	return true, nil
}
