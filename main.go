package main

import (
	server "bootstrap/backend"
	"bootstrap/config"
	"log"
	"net/http"
	"strings"
)

func main() {
	// Load config.yaml file
	conf, err := config.Parse("./config.yaml")
	if err != nil {
		log.Fatal("Error parsing config file: ", err)
	}

	site, err := server.New(nil, conf)
	if err != nil {
		log.Fatal("Error creating server: ", err)
	}

	var https bool = conf.CertFile != ""
	server := &http.Server{
		Addr: conf.Addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If the domain name contains www. redirect to one without.
			host := r.Host
			if strings.HasPrefix(host, "www.") {
				url := *r.URL
				url.Host = host[4:]
				if https {
					url.Scheme = "https"
				} else {
					url.Scheme = "http"
				}
				http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
				return
			}
			site.ServeHTTP(w, r)
		}),
	}

	log.Println("Starting server on " + conf.Addr)

	if https {
		// Running HTTPS server.
		//
		// A server to redirect traffic from HTTP to HTTPS. Started only if the
		// main server is on port 443.
		if conf.Addr[strings.Index(conf.Addr, ":"):] == ":443" {
			redirectServer := &http.Server{
				Addr: ":80",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					url := *r.URL
					url.Scheme = "https"
					url.Host = r.Host
					http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
				}),
			}
			go func() {
				if err = redirectServer.ListenAndServe(); err != nil {
					log.Fatal("Error starting redirect server: ", err)
				}
			}()
		}
		if err := server.ListenAndServeTLS(conf.CertFile, conf.KeyFile); err != nil {
			log.Fatal("Error starting server (TLS): ", err)
		}
	} else {
		// Running HTTP server.
		if err := server.ListenAndServe(); err != nil {
			log.Fatal("Error starting server: ", err)
		}
	}
}
