// src/backend/graph.go
package main // Package sama dengan main.go dan data.go

import (
	"fmt"
	"sync"
)

// --- Variabel Global untuk Graf ---
var (
	// alchemyGraph adalah representasi adjacency list kita.
	// Key: Nama Bahan (string)
	// Value: Slice dari semua resep ([]Recipe) di mana bahan tersebut digunakan.
	alchemyGraph map[string][]Recipe

	// buildGraphOnce memastikan graf hanya dibangun sekali.
	buildGraphOnce sync.Once
)

// --- Fungsi untuk Membangun Graf ---

// BuildGraph menginisialisasi dan mengisi alchemyGraph berdasarkan recipeMap.
// Fungsi ini harus dipanggil SETELAH InitData() berhasil.
func BuildGraph(inputRecipeMap map[string][]Recipe) {
	buildGraphOnce.Do(func() { // Hanya jalankan sekali
		fmt.Println("Membangun struktur graf dari data resep...")
		alchemyGraph = make(map[string][]Recipe)

		// Iterasi melalui semua resep yang sudah dikelompokkan berdasarkan hasil
		for _, recipes := range inputRecipeMap {
			// Iterasi melalui setiap resep individu
			for _, recipe := range recipes {
				// Tambahkan resep ini ke daftar untuk kedua bahannya
				// Jika key belum ada, append akan membuat slice baru
				alchemyGraph[recipe.Ingredient1] = append(alchemyGraph[recipe.Ingredient1], recipe)
				alchemyGraph[recipe.Ingredient2] = append(alchemyGraph[recipe.Ingredient2], recipe)
			}
		}
		fmt.Printf("Graf selesai dibangun. Jumlah node (elemen bahan) dalam graf: %d\n", len(alchemyGraph))
	})
}

// --- (Opsional) Fungsi Getter untuk Graf ---

func GetAlchemyGraph() map[string][]Recipe {
	// Perlu dipastikan BuildGraph sudah dipanggil sebelumnya
	// (Biasanya pemanggilan di main() sudah cukup)
	return alchemyGraph
}