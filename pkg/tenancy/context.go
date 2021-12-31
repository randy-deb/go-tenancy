package tenancy

import (
	"context"
	"net/http"
	"strings"
)

type key int

const tenantKey key = 0

type Request struct {
	*http.Request
}

func UrlRewrite(r *http.Request, t *Tenant) *http.Request {
	if t.VirtualPath != "" {
		segments := strings.Split(r.URL.Path, "/")
		path := "/" + strings.Join(segments[2:], "/")
		r.URL.Path = path
	}
	return r
}

func SetTenant(r *http.Request, t *Tenant) *http.Request {
	ctx := context.WithValue(r.Context(), tenantKey, t)
	return r.WithContext(ctx)
}

func GetTenant(r *http.Request) *Tenant {
	contextData := r.Context().Value(tenantKey)
	tenant, ok := contextData.(*Tenant)

	if !ok {
		return nil
	}
	return tenant
}
