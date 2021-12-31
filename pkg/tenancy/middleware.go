package tenancy

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type contextKey struct {
	name string
}

var TenantContextKey = &contextKey{"tenant"}

type Tenant struct {
	Id          string
	Scheme      string
	Name        string
	Host        string
	VirtualPath string
}

type TenantStore interface {
	Resolve(scheme string, host string, path string) (*Tenant, error)
}

func NewMiddleware(h http.Handler) *Middleware {
	m := &Middleware{
		baseHandler: h,
	}
	m.Handler = m.invoke(m.baseHandler)
	return m
}

type Middleware struct {
	store       TenantStore
	baseHandler http.Handler
	Handler     http.Handler
}

func (m *Middleware) invoke(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme := r.URL.Scheme
		host := r.Host
		path := r.URL.Path
		segments := strings.Split(path, "/")
		virtualPath := ""
		if len(segments) >= 2 {
			virtualPath = segments[1]
		}

		if m.store == nil {
			fmt.Printf("TenantStore not set\n")
		} else {
			tenant, err := m.store.Resolve("http", host, virtualPath)
			if err != nil {
				tenant, err = m.store.Resolve("http", host, "")
			}
			if err != nil {
				fmt.Printf("Tenant not resolved\n")
			} else {
				fmt.Printf("Tenant resolved: %s (%v://%v/%v)\n", tenant.Name, scheme, host, virtualPath)

				r = setTenantUri(r, tenant)
				r = SetTenant(r, tenant)
				next.ServeHTTP(w, r)
				return
			}
		}

		http.NotFound(w, r)
	})
}

func (m *Middleware) SetStore(s TenantStore) {
	m.store = s
}

func (m *Middleware) GetStore() TenantStore {
	return m.store
}

func setTenantUri(r *http.Request, t *Tenant) *http.Request {
	if t.VirtualPath != "" {
		segments := strings.Split(r.URL.Path, "/")
		path := "/" + strings.Join(segments[2:], "/")
		r.URL.Path = path
	}
	return r
}

func SetTenant(r *http.Request, t *Tenant) *http.Request {
	ctx := context.WithValue(r.Context(), TenantContextKey, t)
	return r.WithContext(ctx)
}

func GetTenant(r *http.Request) *Tenant {
	contextData := r.Context().Value(TenantContextKey)
	tenant, ok := contextData.(*Tenant)
	if !ok {
		return nil
	}
	return tenant
}
