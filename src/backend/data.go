// src/backend/data.go
package main // Package sama dengan main.go

import (
	"encoding/json"
	"fmt"

	// "log"
	"os"
	"path/filepath"
	"sync" // Import sync untuk Once
)

// --- Definisi Struct ---
// Struct ini HARUS cocok dengan struktur JSON Anda

type Recipe struct {
	Result      string `json:"result"`
	Ingredient1 string `json:"ingredient1"`
	Ingredient2 string `json:"ingredient2"`
}

type ElementImage struct {
	Name     string `json:"name"`
	ImageURL string `json:"imageURL"`
}

// --- Variabel Global untuk menyimpan data yang sudah dimuat ---
// Menggunakan map untuk akses cepat berdasarkan nama elemen

var (
	// recipeMap menyimpan semua resep untuk elemen hasil tertentu
	// Key: Nama Elemen Hasil (string)
	// Value: Slice dari resep ([]Recipe) yang menghasilkan elemen tsb
	recipeMap map[string][]Recipe

	// imageMap menyimpan URL gambar untuk setiap elemen unik
	// Key: Nama Elemen (string)
	// Value: URL Gambar (string)
	imageMap map[string]string

	// allElementNames menyimpan daftar semua nama elemen unik (hasil + bahan)
	// Berguna untuk validasi input atau fitur lain nanti
	allElementNames map[string]bool

	bfsPathCache = make(map[string][]Recipe)

	// loadDataOnce digunakan untuk memastikan data hanya dimuat sekali
	loadDataOnce sync.Once
	loadDataErr  error // Menyimpan error jika terjadi saat loading
)

// --- Fungsi untuk Memuat dan Memproses Data ---

// InitData memuat data dari file JSON.
// Fungsi ini dirancang untuk dipanggil sekali saja saat aplikasi start.
// Pastikan file JSON ada di dalam subdirektori yang ditentukan (dataDir).
func InitData(dataDir string) error {
	loadDataOnce.Do(func() { // Hanya eksekusi kode di dalam ini sekali
		fmt.Println("Memulai pemuatan data awal dari direktori:", dataDir)

		// Load resep
		tempRecipes, err := loadRecipes(filepath.Join(dataDir, "recipes_final_filtered.json"))
		if err != nil {
			loadDataErr = fmt.Errorf("gagal memuat resep: %w", err)
			return // Hentikan jika resep gagal dimuat
		}
		fmt.Printf("Berhasil memuat %d data resep.\n", len(tempRecipes))

		// Load gambar
		tempImages, err := loadImages(filepath.Join(dataDir, "element_images_urls.json"))
		if err != nil {
			loadDataErr = fmt.Errorf("gagal memuat gambar: %w", err)
			return // Hentikan jika gambar gagal dimuat
		}
		fmt.Printf("Berhasil memuat %d data URL gambar.\n", len(tempImages))

		// Proses data ke dalam map untuk akses efisien
		fmt.Println("Memproses data ke dalam struktur map...")
		processDataToMaps(tempRecipes, tempImages)
		fmt.Println("Selesai memproses data.")
	})

	return loadDataErr // Kembalikan error jika ada yg terjadi saat pemuatan pertama kali
}

// Fungsi internal untuk memuat resep dari file
func loadRecipes(filePath string) ([]Recipe, error) {
	fmt.Printf("Membaca file resep: %s\n", filePath)
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file %s: %w", filePath, err)
	}
	var recipes []Recipe
	err = json.Unmarshal(bytes, &recipes)
	if err != nil {
		return nil, fmt.Errorf("gagal unmarshal JSON resep dari %s: %w", filePath, err)
	}
	return recipes, nil
}

// Fungsi internal untuk memuat gambar dari file
func loadImages(filePath string) ([]ElementImage, error) {
	fmt.Printf("Membaca file URL gambar: %s\n", filePath)
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file %s: %w", filePath, err)
	}
	var images []ElementImage
	err = json.Unmarshal(bytes, &images)
	if err != nil {
		return nil, fmt.Errorf("gagal unmarshal JSON gambar dari %s: %w", filePath, err)
	}
	return images, nil
}

// Fungsi internal untuk memproses data slice ke map
func processDataToMaps(recipes []Recipe, images []ElementImage) {
	recipeMap = make(map[string][]Recipe)
	imageMap = make(map[string]string)
	allElementNames = make(map[string]bool)

	// Proses resep
	for _, r := range recipes {
		recipeMap[r.Result] = append(recipeMap[r.Result], r)
		// Catat semua nama elemen yang terlibat
		allElementNames[r.Result] = true
		allElementNames[r.Ingredient1] = true
		allElementNames[r.Ingredient2] = true
	}

	// Proses gambar
	for _, img := range images {
		imageMap[img.Name] = img.ImageURL
		// Catat juga nama elemen dari data gambar (jika ada yg belum tercatat)
		allElementNames[img.Name] = true
	}

	// Tambahkan elemen dasar secara eksplisit jika belum ada dari scraping
	baseElements := []string{"Air", "Earth", "Fire", "Water"}
	for _, base := range baseElements {
		if _, exists := imageMap[base]; !exists {
			// Jika gambar elemen dasar tidak ada di JSON, URL akan kosong
			// Anda bisa tambahkan placeholder jika diperlukan frontend
			// imageMap[base] = "/placeholder.svg"
			fmt.Printf("Info: URL gambar untuk elemen dasar '%s' tidak ditemukan di JSON.\n", base)
		}
		allElementNames[base] = true
	}

	fmt.Printf("Total elemen unik yang teridentifikasi: %d\n", len(allElementNames))
}

// --- (Opsional) Fungsi Getter untuk Mengakses Data ---
// Membantu mengontrol akses dan bisa menyembunyikan variabel global jika diinginkan

func GetRecipeMap() map[string][]Recipe {
	// Tambahkan pengecekan jika InitData belum dipanggil / error? (tergantung kebutuhan)
	return recipeMap
}

func GetImageMap() map[string]string {
	return imageMap
}

func GetAllElementNames() map[string]bool {
	return allElementNames
}

// IsElementExists memeriksa apakah nama elemen ada dalam daftar elemen yang diketahui.
// Bisa dibuat case-insensitive jika perlu.
func IsElementExists(name string) bool {
	_, exists := allElementNames[name]
	return exists
}
