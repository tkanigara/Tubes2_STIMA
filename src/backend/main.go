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

	RunScraping();
	runFilter();
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
	http.HandleFunc("/api/search", searchHandler) // Daftarkan handler dari handlers.go
	http.HandleFunc("/api/image", imageHandler)
	// Tambahkan handler lain jika ada nanti

	// --- Jalankan Server ---
	port := "8080" // Port yang akan digunakan server
	log.Printf("Server backend berjalan di http://localhost:%s\n", port)
	log.Printf("Server frontend berjalan di http://localhost:3000\n")
	err = http.ListenAndServe(":"+port, nil) // Jalankan server
	if err != nil {
		log.Fatalf("FATAL: Gagal menjalankan server: %v", err)
	}
}