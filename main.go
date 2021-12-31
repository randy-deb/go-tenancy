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

	"github.com/deborggraever/go-tenancy/pkg/tenancy"
	"github.com/gorilla/mux"
)

var tenantStore tenancy.TenantStore

func tenantResolverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		segments := strings.Split(path, "/")
		virtualPath := ""
		if len(segments) >= 2 {
			virtualPath = segments[1]
		}

		tenant, err := tenantStore.Resolve("http", r.Host, virtualPath)
		if err != nil {
			tenant, err = tenantStore.Resolve("http", r.Host, "")
		}
		if err != nil {
			fmt.Printf("Tenant not resolved\n")

			http.NotFound(w, r)
			return
		}

		fmt.Printf("Tenant resolved: %s\n", tenant.Name)

		r = tenancy.UrlRewrite(r, tenant)
		r = tenancy.SetTenant(r, tenant)
		next.ServeHTTP(w, r)
	})
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	tenant := tenancy.GetTenant(r)
	fmt.Fprintf(w, "Tenant: %v\n", tenant.Name)
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

	tenantStore = tenancy.NewInMemoryTenantStore()

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
