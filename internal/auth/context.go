package auth

import "github.com/gin-gonic/gin"

const principalContextKey = "auth_principal"

func WithPrincipal(c *gin.Context, principal Principal) {
	c.Set(principalContextKey, principal)
	c.Set("user_id", principal.UserID)
}

func PrincipalFromContext(c *gin.Context) (Principal, bool) {
	value, ok := c.Get(principalContextKey)
	if !ok {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok
}

func HasAnyRole(principal Principal, roles ...string) bool {
	if len(roles) == 0 {
		return true
	}
	for _, userRole := range principal.Roles {
		if userRole == "admin" {
			return true
		}
		for _, required := range roles {
			if userRole == required {
				return true
			}
		}
	}
	return false
}
