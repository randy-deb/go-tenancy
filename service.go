package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type key int

const tenantKey key = 0

type Tenant struct {
	Id          string
	Name        string
	Host        string
	VirtualPath string
}

var allTenants []Tenant

func findTenant(host string, virtualPath string) *Tenant {
	for _, item := range allTenants {
		if item.Host == host && item.VirtualPath == virtualPath {
			return &item
		}
	}
	return nil
}

func tenantResolverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		segments := strings.Split(path, "/")
		virtualPath := ""
		if len(segments) >= 2 {
			virtualPath = segments[1]
		}

		tenant := findTenant(r.Host, virtualPath)
		if tenant == nil && virtualPath != "" {
			tenant = findTenant(r.Host, "")
		}
		if tenant != nil {
			fmt.Printf("Tenant resolve: %s\r\n", tenant.Name)

			if tenant.VirtualPath != "" {
				fmt.Printf("Rewrite tenant uri\r\n")
				path = "/" + strings.Join(segments[2:], "/")
				r.URL.Path = path
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctx = context.WithValue(ctx, tenantKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.NotFound(w, r)
		}
	})
}

func testHandler(w http.ResponseWriter, r *http.Request) {

	contextData := r.Context().Value(tenantKey)
	tenant, ok := contextData.(*Tenant)

	if ok {
		fmt.Fprintf(w, "test %v", tenant.Name)
	} else {
		fmt.Fprintf(w, "NOK")
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Parse arguments
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	log.Println("Starting")

	allTenants = []Tenant{
		{Id: "1", Name: "Dev", Host: "localhost:5100", VirtualPath: "dev"},
		{Id: "2", Name: "Stg", Host: "localhost:5100", VirtualPath: "stg"},
	}

	// Setup the router
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/test", testHandler).Methods("GET")
	router.HandleFunc("/hc", healthHandler)
	router.Use(loggingMiddleware)

	// Create the tenant resolver middleware
	tenantRouter := tenantResolverMiddleware(router)

	// Create the server
	server := &http.Server{
		Handler:      tenantRouter,
		Addr:         "0.0.0.0:5100",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	// Wait for cancel signal (CTRL+C)
	cancelSignal := make(chan os.Signal, 1)
	signal.Notify(cancelSignal, os.Interrupt)
	<-cancelSignal

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	server.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}
