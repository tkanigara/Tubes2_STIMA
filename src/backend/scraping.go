package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath" // <- Tambahkan import ini
	"strings"
	// Hanya perlu goquery dan library standar ini
	"github.com/PuerkitoBio/goquery"
)

// --- Struct Definitions ---

// Struct untuk menyimpan satu resep kombinasi (hanya teks)
// type Recipe struct {
// 	Result      string `json:"result"`
// 	Ingredient1 string `json:"ingredient1"`
// 	Ingredient2 string `json:"ingredient2"`
// }

// // Struct BARU untuk menyimpan pemetaan nama elemen ke URL gambar
// type ElementImage struct {
// 	Name     string `json:"name"`     // Nama elemen
// 	ImageURL string `json:"imageURL"` // URL Gambar (disimpan apa adanya)
// }

// --- Konfigurasi ---

// !!! GANTI URL INI DENGAN URL WEBSITE YANG ANDA SCRAPE !!!
// Contoh: "https://hints.littlealchemy2.com/all"
const targetURL = "https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)#Tier_15_elements"
// Dihapus sesuai permintaan, tapi diperlukan jika ada URL relatif pada gambar/link lain
// const baseURL = "URL_DASAR_WEBSITE_TARGET_ANDA"

// --- Fungsi Helper ---

// Fungsi untuk mencoba mendapatkan URL gambar yang valid (handle data: dan data-src)
func getValidImageURL(imgTag *goquery.Selection) (string, bool) {
	// 1. Coba atribut 'data-src' (atau 'data-original', dll. - SESUAIKAN JIKA PERLU)
	imgSrc, exists := imgTag.Attr("data-src")
	if exists && !strings.HasPrefix(imgSrc, "data:") {
		return imgSrc, true // Dapatkan URL asli dari data-src
	}
	// 2. Jika tidak ada data-src, coba atribut 'src'
	imgSrc, exists = imgTag.Attr("src")
	if exists && !strings.HasPrefix(imgSrc, "data:") {
		return imgSrc, true // Dapat URL normal dari src
	}
	// 3. Abaikan jika src adalah data: URL atau tidak ada src
	return "", false // Tidak ditemukan URL gambar yang valid
}


// --- Fungsi Utama ---
func RunScraping() {
	// Validasi URL Target
	if targetURL == "URL_WEBSITE_TARGET_ANDA_DI_SINI" {
		log.Fatal("Error: Anda belum mengganti placeholder targetURL di dalam kode!")
	}

	// --- MODIFIKASI: Tentukan direktori data dan buat jika belum ada ---
	dataDir := "data" // Nama subdirektori
	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		log.Fatalf("Error membuat direktori '%s': %v", dataDir, err)
	}
	fmt.Printf("Memastikan direktori '%s' ada.\n", dataDir)
	// -------------------------------------------------------------------

	fmt.Println("Memulai proses scraping dari:", targetURL)

	// 1. HTTP GET Request
	res, err := http.Get(targetURL)
	if err != nil { log.Fatalf("Error GET request: %v", err) }
	defer res.Body.Close()
	if res.StatusCode != 200 { log.Fatalf("Error status code: %d", res.StatusCode) }

	// 2. Load HTML
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil { log.Fatalf("Error membaca HTML: %v", err) }
	fmt.Println("Berhasil memuat dokumen HTML.")

	// 3. Proses Scraping
	var allRecipes []Recipe               // Slice untuk data resep
	var elementImages []ElementImage     // Slice BARU untuk data gambar
	processedElements := make(map[string]bool) // Set untuk melacak elemen yg gambarnya sudah diproses

	// Selector Tabel Utama
	tableSelector := "table.list-table.col-list.icon-hover" // !! VERIFIKASI !!
	fmt.Printf("Mencari tabel dengan selector: '%s'\n", tableSelector)

	doc.Find(tableSelector).Each(func(index int, table *goquery.Selection) {
		fmt.Printf("\nMemproses Tabel ke-%d\n", index+1)
		table.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
			if row.Find("th").Length() > 0 { return } // Skip header

			// --- Ekstrak Info Elemen Hasil (Kolom 1) ---
			resultCell := row.Find("td:nth-child(1)")
			resultNameLink := resultCell.Find("a") // !! VERIFIKASI !!
			resultName := strings.TrimSpace(resultNameLink.Text())
			if resultName == "" { return } // Skip baris tanpa nama

			fmt.Printf("  Memproses Elemen: %s\n", resultName)

			// Cari URL Gambar Elemen Hasil & simpan jika belum diproses
			if _, processed := processedElements[resultName]; !processed {
				imgResultSelector := "span > span > a > img" // !! VERIFIKASI !!
				imgURL := ""
				resultCell.Find(imgResultSelector).First().Each(func(_ int, imgTag *goquery.Selection) {
					validURL, isValid := getValidImageURL(imgTag)
					if isValid {
						// Simpan URL apa adanya (absolut atau relatif)
						imgURL = validURL
					}
				})
				if imgURL != "" {
					elementImages = append(elementImages, ElementImage{Name: resultName, ImageURL: imgURL})
					processedElements[resultName] = true // Tandai sudah diproses
					fmt.Printf("    -> URL Gambar Hasil ditemukan: %s\n", imgURL)
				} else {
					fmt.Printf("    -> Peringatan: Tidak ditemukan URL gambar valid untuk hasil '%s'.\n", resultName)
					processedElements[resultName] = true // Tetap tandai agar tidak dicari lagi
				}
			}

			// --- Ekstrak Info Resep (Kolom 2) ---
			recipesCell := row.Find("td:nth-child(2)")
			if recipesCell.Find("ul").Length() == 0 || strings.Contains(strings.ToLower(recipesCell.Text()), "available from the start") || strings.Contains(strings.ToLower(recipesCell.Text()), "does not have any recipes") {
				return // Skip resep
			}

			// Loop setiap item resep (li)
			liSelector := "ul > li" // !! VERIFIKASI !!
			recipesCell.Find(liSelector).Each(func(j int, li *goquery.Selection) {
				var ingredientNames []string
				var ingredientImageURLs []string // Tampung URL valid yg ditemukan

				// Cari Nama Bahan
				nameIngredientSelector := "a" // !! VERIFIKASI !!
				li.Find(nameIngredientSelector).Each(func(k int, nameLink *goquery.Selection) {
					ingName := strings.TrimSpace(nameLink.Text())
					if ingName != "" && ingName != "+" && len(ingName) > 1 {
						ingredientNames = append(ingredientNames, ingName)
					}
				})

				// Cari Gambar Bahan
				imgIngredientSelector := "span > span > a > img" // !! VERIFIKASI !!
				li.Find(imgIngredientSelector).Each(func(k int, imgTag *goquery.Selection) {
					imgURL, isValid := getValidImageURL(imgTag)
					if isValid {
						ingredientImageURLs = append(ingredientImageURLs, imgURL) // Simpan URL apa adanya
					}
				})

				// Verifikasi jumlah & buat struct
				if len(ingredientNames) == 2 {
					bahan1 := ingredientNames[0]
					bahan2 := ingredientNames[1]

					// Buat resep teks
					fmt.Printf("    -> Resep ke-%d: %s + %s\n", j+1, bahan1, bahan2)
					recipe := Recipe{ Result: resultName, Ingredient1: bahan1, Ingredient2: bahan2 }
					allRecipes = append(allRecipes, recipe)

					// Cocokkan & Simpan URL Gambar Bahan jika belum diproses
					var imgURL1, imgURL2 string
					if len(ingredientImageURLs) >= 1 { imgURL1 = ingredientImageURLs[0] }
					if len(ingredientImageURLs) == 2 { imgURL2 = ingredientImageURLs[1] }

					if _, processed := processedElements[bahan1]; !processed && imgURL1 != "" {
						elementImages = append(elementImages, ElementImage{Name: bahan1, ImageURL: imgURL1})
						processedElements[bahan1] = true
						fmt.Printf("      -> URL Gambar Bahan 1 ditemukan: %s\n", imgURL1)
					}
					if _, processed := processedElements[bahan2]; !processed && imgURL2 != "" {
						elementImages = append(elementImages, ElementImage{Name: bahan2, ImageURL: imgURL2})
						processedElements[bahan2] = true
						fmt.Printf("      -> URL Gambar Bahan 2 ditemukan: %s\n", imgURL2)
					}

				} else {
					fmt.Printf("    -> Peringatan: Gagal memproses resep ke-%d untuk %s. Jumlah Nama Bahan: %d (%v)\n",
						j+1, resultName, len(ingredientNames), ingredientNames)
				}
			}) // Akhir loop li
		}) // Akhir loop tr
	}) // Akhir loop table

	fmt.Printf("\nTotal resep tekstual yang berhasil di-scrape: %d\n", len(allRecipes))
	fmt.Printf("Total pemetaan gambar elemen unik yang ditemukan: %d\n", len(elementImages))

	// --- MODIFIKASI: Simpan file ke dalam direktori 'data' ---
	// 4. Marshal & Tulis JSON untuk Resep
	if len(allRecipes) > 0 {
		recipeData, err := json.MarshalIndent(allRecipes, "", "  ")
		if err != nil { log.Fatalf("Error marshal JSON Resep: %v", err) }
		recipeFileName := filepath.Join(dataDir, "recipes_scraped.json") // Gunakan filepath.Join
		err = os.WriteFile(recipeFileName, recipeData, 0644)
		if err != nil { log.Fatalf("Error menulis JSON Resep ke file '%s': %v", recipeFileName, err) }
		fmt.Printf("Sukses! Data resep tekstual telah disimpan ke %s\n", recipeFileName)
	} else {
		fmt.Println("Tidak ada resep tekstual yang di-scrape untuk disimpan.")
	}

	// 5. Marshal & Tulis JSON untuk Gambar Elemen
	if len(elementImages) > 0 {
		imageData, err := json.MarshalIndent(elementImages, "", "  ")
		if err != nil { log.Fatalf("Error marshal JSON Gambar: %v", err) }
		imageFileName := filepath.Join(dataDir, "element_images_urls.json") // Gunakan filepath.Join
		err = os.WriteFile(imageFileName, imageData, 0644)
		if err != nil { log.Fatalf("Error menulis JSON Gambar ke file '%s': %v", imageFileName, err) }
		fmt.Printf("Sukses! Data URL gambar elemen telah disimpan ke %s\n", imageFileName)
	} else {
		fmt.Println("Tidak ada data URL gambar elemen yang di-scrape untuk disimpan.")
	}
	// ------------------------------------------------------------
}