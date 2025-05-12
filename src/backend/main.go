// src/backend/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http" // Import net/http
)

func main() {
	scrapeOnly := flag.Bool("scrapeonly", false, "Run scraping and filtering then exit")
	flag.Parse()

	RunScraping()
	runFilter()
	if *scrapeOnly {
		log.Println("Scraping dan filtering selesai (mode scrapeonly). Aplikasi akan keluar.")
		return // Keluar setelah scraping dan filter jika flag aktif
	}
	log.Println("=== MEMULAI SERVER BACKEND ===")
	dataDirPath := "data"
	err := InitData(dataDirPath) // Dari data.go
	if err != nil {
		log.Fatalf("FATAL: Gagal memuat data awal aplikasi dari '%s': %v", dataDirPath, err)
	}
	fmt.Println("Data awal berhasil dimuat.")
	BuildGraph(GetRecipeMap()) // Dari graph.go
	fmt.Println("Struktur graf siap digunakan.")
	// --- Setup Rute API ---
	// Add CORS middleware to handle preflight requests and set proper headers
	corsMiddleware := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

			// Set CORS headers for all responses
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}

	// Create a new serveMux to apply middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", searchHandler) // Daftarkan handler dari handlers.go
	mux.HandleFunc("/api/image", imageHandler)

	// Test route for checking connectivity
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Ping received")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","message":"Backend is running"}`))
	})

	// Apply middleware
	handler := corsMiddleware(mux)

	// --- Jalankan Server ---
	port := "8080" // Port yang akan digunakan server
	log.Printf("Server backend berjalan di http://0.0.0.0:%s\n", port)
	log.Printf("Server frontend berjalan di http://localhost:3000\n")
	err = http.ListenAndServe("0.0.0.0:"+port, handler) // Jalankan server dengan middleware
	if err != nil {
		log.Fatalf("FATAL: Gagal menjalankan server: %v", err)
	}
}
