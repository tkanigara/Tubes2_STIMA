// src/backend/handlers.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url" // Pastikan package ini sudah di-import
	"strconv"
	"strings"
	"time"
	// sync tidak perlu di sini jika tidak digunakan secara langsung di file ini
)

// MultiSearchResponse struct untuk struktur respons JSON ke frontend
type MultiSearchResponse struct {
	SearchTarget   string            `json:"searchTarget"`
	Algorithm      string            `json:"algorithm"`
	Mode           string            `json:"mode"`
	MaxRecipes     int               `json:"maxRecipes,omitempty"` // Hanya ada jika mode multiple
	PathFound      bool              `json:"pathFound"`
	Path           []Recipe          `json:"path,omitempty"`      // Untuk mode shortest
	Paths          [][]Recipe        `json:"paths,omitempty"`     // Untuk mode multiple
	ImageURLs      map[string]string `json:"imageURLs,omitempty"` // URL gambar untuk elemen yang relevan
	NodesVisited   int               `json:"nodesVisited"`
	DurationMillis int64             `json:"durationMillis"`
	Error          string            `json:"error,omitempty"` // Pesan error jika ada
}

// imageHandler berfungsi sebagai proxy untuk mengambil gambar elemen dari URL aslinya.
// Ini membantu menghindari masalah CORS di frontend.
func imageHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers agar frontend bisa mengakses
	w.Header().Set("Access-Control-Allow-Origin", "*")             // Izinkan akses dari origin manapun
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // Header yang diizinkan

	// Hanya izinkan metode GET
	if r.Method != http.MethodGet {
		http.Error(w, "Metode tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	// Ambil nama elemen dari query parameter 'elementName'
	queryParams := r.URL.Query()
	elementName := queryParams.Get("elementName")

	if elementName == "" {
		http.Error(w, "Parameter 'elementName' diperlukan", http.StatusBadRequest)
		return
	}

	log.Printf("Menerima permintaan gambar untuk elemen: %s\n", elementName)

	// Dapatkan map URL gambar dari data yang sudah dimuat (dari data.go)
	imageMap := GetImageMap() // Asumsikan GetImageMap() mengembalikan map[string]string

	// Cari URL gambar asli untuk elemen ini
	originalImageURL, found := imageMap[elementName]
	if !found || originalImageURL == "" {
		log.Printf("URL gambar tidak ditemukan untuk elemen: %s\n", elementName)
		http.Error(w, "URL gambar tidak ditemukan", http.StatusNotFound)
		return
	}

	log.Printf("Mengambil gambar dari URL: %s\n", originalImageURL)

	// Lakukan permintaan HTTP GET ke URL gambar asli DARI BACKEND
	client := http.Client{
		Timeout: 10 * time.Second, // Tambahkan timeout untuk request eksternal
	}
	req, err := http.NewRequest("GET", originalImageURL, nil)
	if err != nil {
		log.Printf("Gagal membuat request ke URL gambar eksternal %s: %v\n", originalImageURL, err)
		http.Error(w, "Gagal mengambil gambar (internal server error)", http.StatusInternalServerError)
		return
	}
	// Opsional: Tambahkan User-Agent agar terlihat seperti browser sungguhan
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyLittleAlchemyApp/1.0; +http://localhost)")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Gagal melakukan permintaan GET ke URL gambar eksternal %s: %v\n", originalImageURL, err)
		http.Error(w, "Gagal mengambil gambar dari sumber eksternal", http.StatusBadGateway) // Atau StatusInternalServerError
		return
	}
	defer resp.Body.Close() // Pastikan body respons ditutup

	// Periksa status code dari respons server sumber gambar
	if resp.StatusCode != http.StatusOK {
		log.Printf("Server sumber gambar mengembalikan status non-OK untuk %s: %d\n", originalImageURL, resp.StatusCode)
		// Teruskan status code error dari server sumber jika memungkinkan, atau gunakan default
		http.Error(w, fmt.Sprintf("Gagal mengambil gambar dari sumber (status: %d)", resp.StatusCode), resp.StatusCode)
		return
	}

	// Salin header Content-Type dari respons server sumber gambar ke respons backend kita
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		// Jika Content-Type tidak ada, coba tebak atau default
		if strings.HasSuffix(strings.ToLower(originalImageURL), ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		} else if strings.HasSuffix(strings.ToLower(originalImageURL), ".png") {
			w.Header().Set("Content-Type", "image/png")
		} // Tambahkan tipe lain jika perlu (jpg, gif, dll.)
	}
	// Anda juga bisa menyalin header lain yang relevan jika perlu (misal Cache-Control)
	// w.Header().Set("Cache-Control", resp.Header.Get("Cache-Control"))

	// Salin body respons (data gambar) dari server sumber ke respons backend
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Gagal menyalin body respons gambar dari %s: %v\n", originalImageURL, err)
		// Tidak mengirim http.Error lagi karena header mungkin sudah terkirim
		return
	}

	log.Printf("Gambar untuk elemen %s berhasil dilayani.\n", elementName)
}

// searchHandler menangani permintaan pencarian resep dari frontend.
func searchHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Hanya izinkan metode GET
	if r.Method != http.MethodGet {
		http.Error(w, "Metode tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	// 1. Ambil Query Parameters
	targetElement := strings.TrimSpace(r.URL.Query().Get("target"))

	// Coba format yang berbeda untuk meningkatkan peluang menemukan elemen
	// Format 1: Title case untuk setiap kata (Grilled Cheese)
	titleCaseTarget := toTitleCase(targetElement)

	// Format 2: Huruf pertama kapital saja (Grilled cheese)
	firstCapTarget := ""
	if len(targetElement) > 0 {
		firstCapTarget = strings.ToUpper(string(targetElement[0]))
		if len(targetElement) > 1 {
			firstCapTarget += strings.ToLower(targetElement[1:])
		}
	}

	// Format 3: Semua huruf kecil (grilled cheese)
	lowerCaseTarget := strings.ToLower(targetElement)

	// Format 4: Semua huruf kapital (GRILLED CHEESE)
	upperCaseTarget := strings.ToUpper(targetElement)

	// Coba semua format satu per satu
	potentialTargets := []string{titleCaseTarget, firstCapTarget, targetElement, lowerCaseTarget, upperCaseTarget}

	// Variabel untuk menyimpan target yang valid
	validTarget := ""

	// Cek satu per satu
	for _, potTarget := range potentialTargets {
		if IsElementExists(potTarget) {
			validTarget = potTarget
			break
		}
	}

	// Jika tidak ada yang cocok, gunakan format title case
	if validTarget == "" {
		validTarget = titleCaseTarget
	}

	// Gunakan validTarget untuk pencarian
	targetElement = validTarget

	algo := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("algo")))
	mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
	maxRecipesStr := r.URL.Query().Get("max")

	// Default values jika parameter tidak ada
	if algo == "" {
		algo = "bfs" // Default ke BFS
	}
	if mode == "" {
		mode = "shortest" // Default ke mode shortest
	}

	// 2. Validasi Input Dasar
	if targetElement == "" {
		http.Error(w, "Parameter 'target' diperlukan", http.StatusBadRequest)
		return
	}
	// Gunakan IsElementExists dari data.go atau file lain yang sesuai
	if !IsElementExists(targetElement) {
		http.Error(w, fmt.Sprintf("Elemen target '%s' tidak valid atau tidak ditemukan", targetElement), http.StatusBadRequest)
		return
	}
	if algo != "bfs" && algo != "dfs" && algo != "bds" { // Validasi algoritma
		http.Error(w, "Parameter 'algo' harus 'bfs', 'dfs', atau 'bds'", http.StatusBadRequest)
		return
	}
	if mode != "shortest" && mode != "multiple" { // Validasi mode
		http.Error(w, "Parameter 'mode' harus 'shortest' atau 'multiple'", http.StatusBadRequest)
		return
	}

	// 3. Proses parameter 'max' jika mode 'multiple'
	maxRecipes := 1 // Default untuk mode 'shortest' atau jika 'max' tidak valid
	if mode == "multiple" {
		if maxRecipesStr != "" {
			var convErr error
			maxRecipes, convErr = strconv.Atoi(maxRecipesStr)
			if convErr != nil || maxRecipes <= 0 {
				http.Error(w, "Parameter 'max' harus berupa angka positif lebih besar dari 0 untuk mode 'multiple'", http.StatusBadRequest)
				return
			}
		} else {
			// Jika mode multiple tapi 'max' tidak disediakan, bisa set default atau error
			// Untuk sekarang, kita error jika tidak ada 'max' di mode multiple
			http.Error(w, "Parameter 'max' diperlukan untuk mode 'multiple'", http.StatusBadRequest)
			return
		}
	}

	// 4. Panggil Fungsi Algoritma & Ukur Waktu
	startTime := time.Now()

	var singlePath []Recipe
	var multiplePaths [][]Recipe
	var nodesVisited int
	var errSearch error // Ubah nama variabel error agar tidak bentrok dengan package 'errors'
	var pathFound bool

	log.Printf("Memulai pencarian: Target=%s, Algo=%s, Mode=%s, MaxRecipes=%d\n", targetElement, algo, mode, maxRecipes)

	// --- Struktur Response Awal ---
	response := MultiSearchResponse{
		SearchTarget: targetElement,
		Algorithm:    algo,
		Mode:         mode,
	}
	if mode == "multiple" {
		response.MaxRecipes = maxRecipes // Set max recipes jika mode multiple
	}

	// --- Logika Pemilihan Algoritma ---
	if algo == "bfs" {
		if mode == "shortest" {
			singlePath, nodesVisited, errSearch = FindPathBFS(targetElement)
			response.Path = singlePath
			// pathFound true jika tidak ada error DAN (path tidak kosong ATAU target adalah elemen dasar)
			pathFound = errSearch == nil && (len(singlePath) > 0 || (len(singlePath) == 0 && isBaseElement(targetElement)))
		} else { // mode == "multiple"
			multiplePaths, nodesVisited, errSearch = FindMultiplePathsBFS(targetElement, maxRecipes)
			response.Paths = multiplePaths
			pathFound = errSearch == nil && (len(multiplePaths) > 0 || (len(multiplePaths) == 0 && isBaseElement(targetElement)))
		}
	} else if algo == "dfs" {
		if mode == "shortest" {
			singlePath, nodesVisited, errSearch = FindPathDFS(targetElement) // Menggunakan DFS Single Path
			response.Path = singlePath
			pathFound = errSearch == nil && (len(singlePath) > 0 || (len(singlePath) == 0 && isBaseElement(targetElement)))
		} else { // mode == "multiple"
			log.Printf("Menjalankan DFS Multiple untuk target: %s, max: %d", targetElement, maxRecipes)
			multiplePaths, nodesVisited, errSearch = FindMultiplePathsDFS(targetElement, maxRecipes)
			response.Paths = multiplePaths
			pathFound = errSearch == nil && (len(multiplePaths) > 0 || (len(multiplePaths) == 0 && isBaseElement(targetElement)))
		}
	} else if algo == "bds" {
		log.Printf("Permintaan BDS diterima untuk Target: %s, Mode: %s, MaxRecipes: %d\n", targetElement, mode, maxRecipes)
		if mode == "shortest" {
			singlePath, nodesVisited, errSearch = FindPathBDS(targetElement) // Panggil placeholder BDS
			response.Path = singlePath
			// Logika pathFound untuk BDS setelah diimplementasikan
			pathFound = errSearch == nil && singlePath != nil && (len(singlePath) > 0 || (len(singlePath) == 0 && isBaseElement(targetElement)))
			if errSearch != nil { // Jika fungsi placeholder mengembalikan error "belum diimplementasikan"
				log.Printf("FindPathBDS error: %v", errSearch)
				pathFound = false // Pastikan pathFound false jika ada error implementasi
			}
		} else { // mode == "multiple"
			multiplePaths, nodesVisited, errSearch = FindMultiplePathsBDS(targetElement, maxRecipes) // Panggil placeholder BDS
			response.Paths = multiplePaths
			// Logika pathFound untuk BDS setelah diimplementasikan
			pathFound = errSearch == nil && multiplePaths != nil && (len(multiplePaths) > 0 || (len(multiplePaths) == 0 && isBaseElement(targetElement)))
			if errSearch != nil { // Jika fungsi placeholder mengembalikan error "belum diimplementasikan"
				log.Printf("FindMultiplePathsBDS error: %v", errSearch)
				pathFound = false // Pastikan pathFound false jika ada error implementasi
			}
		}
	}

	duration := time.Since(startTime)
	log.Printf("Pencarian selesai: Durasi=%v, Nodes Dikeluarkan dari Queue/Stack (Perkiraan)=%d, Path Ditemukan=%t, Error=%v\n", duration, nodesVisited, pathFound, errSearch)

	// --- Isi sisa response ---
	response.PathFound = pathFound
	response.NodesVisited = nodesVisited
	response.DurationMillis = duration.Milliseconds()

	if errSearch != nil {
		response.Error = errSearch.Error()
	}

	// --- Ambil URL Gambar untuk SEMUA elemen yang relevan ---
	if response.PathFound {
		imgMap := GetImageMap() // Pastikan fungsi ini ada dan mengembalikan map[string]string
		elementsInPaths := make(map[string]bool)

		// Kumpulkan semua elemen unik dari semua jalur resep yang berhasil ditemukan
		pathsToProcess := [][]Recipe{}
		if response.Mode == "shortest" && response.Path != nil {
			if len(response.Path) > 0 { // Hanya tambahkan path jika tidak kosong
				pathsToProcess = append(pathsToProcess, response.Path)
			}
		} else if response.Mode == "multiple" && response.Paths != nil {
			if len(response.Paths) > 0 { // Hanya tambahkan paths jika tidak kosong
				pathsToProcess = response.Paths
			}
		}
		// Tambahkan target elemen ke elementsInPaths jika belum ada (khususnya jika elemen dasar)
		elementsInPaths[response.SearchTarget] = true

		for _, path := range pathsToProcess {
			for _, step := range path {
				elementsInPaths[step.Ingredient1] = true
				elementsInPaths[step.Ingredient2] = true
				elementsInPaths[step.Result] = true // Tambahkan juga elemen hasil di setiap langkah
			}
		}

		response.ImageURLs = make(map[string]string) // Inisialisasi map gambar di sini
		for elementName := range elementsInPaths {
			if imgActualUrl, ok := imgMap[elementName]; ok && imgActualUrl != "" {
				// BUAT URL YANG MENGARAH ke endpoint backend proxy /api/image
				proxyUrl := fmt.Sprintf("/api/image?elementName=%s", url.QueryEscape(elementName))
				if _, exists := response.ImageURLs[elementName]; !exists {
					response.ImageURLs[elementName] = proxyUrl
				}
			}
		}
	}
	// END --- Ambil URL Gambar ---

	// Encode Response ke JSON dan Kirim
	w.Header().Set("Content-Type", "application/json")
	jsonResponse, jsonErr := json.MarshalIndent(response, "", "  ") // Gunakan MarshalIndent untuk pretty print
	if jsonErr != nil {
		log.Printf("Error saat marshal JSON response: %v", jsonErr)
		http.Error(w, "Internal Server Error saat membuat respons JSON", http.StatusInternalServerError)
		return
	}

	_, writeErr := w.Write(jsonResponse)
	if writeErr != nil {
		log.Printf("Error saat menulis JSON response: %v", writeErr)
		// Tidak mengirim http.Error lagi karena header mungkin sudah terkirim
	}
}

// Fungsi untuk mengubah format string menjadi Title Case
// (huruf pertama tiap kata besar, sisanya kecil)
// Ganti fungsi toTitleCase dengan fungsi ini
func toTitleCase(input string) string {
	// Pisahkan string menjadi kata-kata
	words := strings.Fields(input)
	result := make([]string, len(words))

	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		// Untuk setiap kata, buat huruf pertama kapital dan sisanya kecil
		firstChar := strings.ToUpper(string(word[0]))
		restOfWord := ""
		if len(word) > 1 {
			restOfWord = strings.ToLower(word[1:])
		}
		result[i] = firstChar + restOfWord
	}

	// Gabungkan kembali menjadi satu string
	return strings.Join(result, " ")
}
