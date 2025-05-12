package main

import (
	"container/list"
	"errors"
	"fmt"
	"sort" // Diperlukan untuk generatePathIdentifier dan sorting
	// Diperlukan untuk generatePathIdentifier
	"sync"
	"sync/atomic" // Digunakan untuk nodesVisitedCount di multiple
)

// --- Definisi yang Duplikat Dihapus ---
// ... (asumsi definisi ada di file lain) ...

// --- Implementasi Bidirectional Search (BDS) ---

// reconstructSingleSegmentPath: Membangun jalur dari parent maps setelah pertemuan.
// (Fungsi ini tetap sama seperti versi sebelumnya)
func reconstructSingleSegmentPath(parentMap map[string]Recipe, startNode string, stopCondition func(string) bool) []Recipe {
	pathList := list.New()
	processed := make(map[string]bool)
	curr := startNode
	for curr != "" && !stopCondition(curr) {
		recipe, exists := parentMap[curr]
		if !exists { break }
		recipeKey := getUniqueRecipeKey(recipe)
		if processed[recipeKey] { break }
		pathList.PushFront(recipe)
		processed[recipeKey] = true
		_, p1Exists := parentMap[recipe.Ingredient1]
		_, p2Exists := parentMap[recipe.Ingredient2]
		chosenParent := ""
		if p1Exists && p2Exists {
			if recipe.Ingredient1 < recipe.Ingredient2 { chosenParent = recipe.Ingredient1 } else { chosenParent = recipe.Ingredient2 }
		} else if p1Exists { chosenParent = recipe.Ingredient1
		} else if p2Exists { chosenParent = recipe.Ingredient2
		} else {
			if isBaseElement(recipe.Ingredient1) && stopCondition(recipe.Ingredient1) { chosenParent = recipe.Ingredient1
			} else if isBaseElement(recipe.Ingredient2) && stopCondition(recipe.Ingredient2) { chosenParent = recipe.Ingredient2
			} else if isBaseElement(recipe.Ingredient1) && !stopCondition(recipe.Ingredient1) { chosenParent = recipe.Ingredient1
			} else if isBaseElement(recipe.Ingredient2) && !stopCondition(recipe.Ingredient2) { chosenParent = recipe.Ingredient2
			} else { chosenParent = "" }
		}
		curr = chosenParent
	}
	finalPath := make([]Recipe, 0, pathList.Len())
	for e := pathList.Front(); e != nil; e = e.Next() {
		finalPath = append(finalPath, e.Value.(Recipe))
	}
	return finalPath
}

// buildSortedPathFromRecipes: Mengurutkan sekumpulan resep berdasarkan dependensi.
// Mirip dengan logika buildRecipePath di BFS.
func buildSortedPathFromRecipes(recipes map[string]Recipe, targetElement string) []Recipe {
	fmt.Println("  Mengurutkan resep gabungan berdasarkan dependensi...")
	if len(recipes) == 0 {
		return []Recipe{}
	}

	// 1. Identifikasi semua elemen yang terlibat (bahan dan hasil)
	elementsInvolved := make(map[string]bool)
	for _, r := range recipes {
		elementsInvolved[r.Ingredient1] = true
		elementsInvolved[r.Ingredient2] = true
		elementsInvolved[r.Result] = true
	}

	// 2. Inisialisasi elemen yang tersedia (awalnya hanya base elements)
	available := make(map[string]bool)
	for _, base := range baseElements {
		if elementsInvolved[base] { // Hanya tambahkan base element jika relevan dengan path ini
			available[base] = true
		}
	}

	// 3. Buat map resep yang belum digunakan
	remainingRecipes := make(map[string]Recipe)
	for k, v := range recipes {
		remainingRecipes[k] = v
	}

	// 4. Iteratif membangun jalur terurut
	sortedPath := make([]Recipe, 0, len(recipes))
	iterations := 0 // Safety break
	maxIterations := len(recipes) * 2 + 10 // Batas iterasi untuk mencegah infinite loop

	for !available[targetElement] && iterations < maxIterations {
		addedRecipeInIteration := false
		// Cari resep yang bisa dibuat (kedua bahan tersedia) dari resep yang tersisa
		candidates := make([]Recipe, 0)
		candidateKeys := make([]string, 0) // Simpan key untuk menghapus dari remainingRecipes

		for key, recipe := range remainingRecipes {
			if available[recipe.Ingredient1] && available[recipe.Ingredient2] {
				candidates = append(candidates, recipe)
				candidateKeys = append(candidateKeys, key)
			}
		}

		if len(candidates) == 0 {
			// Tidak ada lagi resep yang bisa dibuat, tapi target belum tercapai
			fmt.Printf("  ERROR (Sort): Tidak ada kandidat resep yang bisa dibuat, target '%s' belum tersedia. Elemen tersedia: %v\n", targetElement, available)
			// Kembalikan apa yang sudah diurutkan sejauh ini, mungkin tidak lengkap
			return sortedPath
		}

		// Pilih kandidat terbaik (misalnya, urutkan untuk determinisme)
		sort.SliceStable(candidates, func(i, j int) bool {
			// Strategi sorting bisa bervariasi, misal berdasarkan nama hasil
			return candidates[i].Result < candidates[j].Result
		})

		// Tambahkan resep terpilih ke jalur terurut
		// Kita bisa tambahkan semua kandidat yang valid di iterasi ini
		for i, recipe := range candidates {
			key := candidateKeys[i]
			sortedPath = append(sortedPath, recipe)
			available[recipe.Result] = true // Tandai hasil sebagai tersedia
			delete(remainingRecipes, key)  // Hapus dari resep tersisa
			addedRecipeInIteration = true
			// fmt.Printf("    Sort Step %d: Menambahkan %s + %s => %s\n", len(sortedPath), recipe.Ingredient1, recipe.Ingredient2, recipe.Result)
		}

		if !addedRecipeInIteration && !available[targetElement] {
			// Jika tidak ada resep yang ditambahkan tapi target belum ada, berarti ada masalah
			fmt.Printf("  ERROR (Sort): Tidak ada resep baru ditambahkan di iterasi %d, target '%s' belum tersedia.\n", iterations+1, targetElement)
			return sortedPath // Kembalikan path parsial
		}
		iterations++
	}

	if iterations >= maxIterations {
		fmt.Printf("  ERROR (Sort): Melebihi batas iterasi maksimum (%d), target '%s' mungkin tidak tercapai atau ada loop dependensi.\n", maxIterations, targetElement)
	} else if !available[targetElement] {
        fmt.Printf("  PERINGATAN (Sort): Loop selesai, tapi target '%s' tidak tersedia di akhir.\n", targetElement)
    } else {
        fmt.Printf("  Pengurutan resep selesai. Total langkah terurut: %d\n", len(sortedPath))
    }


	return sortedPath
}


// FindPathBDS: Mencari jalur menggunakan hybrid BDS + BFS.
func FindPathBDS(targetElement string) ([]Recipe, int, error) {
	fmt.Printf("Hybrid BDS+BFS: Mencari jalur ke: %s\n", targetElement)
	recipeMap := GetRecipeMap()
	alchemyGraph := GetAlchemyGraph()
	if recipeMap == nil || alchemyGraph == nil {
		return nil, 0, errors.New("data resep/graf belum diinisialisasi")
	}
	if isBaseElement(targetElement) {
		return []Recipe{}, 0, nil
	}

	nodesVisitedCount := 0
	queueForward := list.New()
	visitedForward := make(map[string]int)
	parentForward := make(map[string]Recipe)
	queueBackward := list.New()
	visitedBackward := make(map[string]int)
	parentBackward := make(map[string]Recipe)

	// Inisialisasi
	for _, base := range baseElements {
		if visitedForward[base] == 0 {
			queueForward.PushBack(base)
			visitedForward[base] = 1
		}
	}
	if visitedBackward[targetElement] == 0 {
		queueBackward.PushBack(targetElement)
		visitedBackward[targetElement] = 1
	}

	currentLevelForward := 1
	currentLevelBackward := 1
	var meetingNode string = ""

	// --- Loop BDS Utama ---
	for queueForward.Len() > 0 && queueBackward.Len() > 0 && meetingNode == "" {

		// --- Langkah Maju ---
		lenF := queueForward.Len()
		for i := 0; i < lenF && meetingNode == ""; i++ {
			if queueForward.Len() == 0 { break }
			currF := queueForward.Remove(queueForward.Front()).(string)
			nodesVisitedCount++

			if visitedBackward[currF] > 0 {
				// fmt.Printf("BDS: Pertemuan DARI FWD di '%s'\n", currF)
				meetingNode = currF
			}

			recipesUsingCurrF := alchemyGraph[currF]
			for _, recipe := range recipesUsingCurrF {
				otherIng := ""
				if recipe.Ingredient1 == currF { otherIng = recipe.Ingredient2 } else
				if recipe.Ingredient2 == currF { otherIng = recipe.Ingredient1 } else { continue }

				if visitedForward[otherIng] > 0 && visitedForward[otherIng] <= currentLevelForward {
					result := recipe.Result
					if visitedForward[result] == 0 {
						visitedForward[result] = currentLevelForward + 1
						parentForward[result] = recipe
						queueForward.PushBack(result)

						if visitedBackward[result] > 0 && meetingNode == "" {
							// fmt.Printf("BDS: Pertemuan SETELAH FWD ekspansi di '%s'\n", result)
							meetingNode = result
						}
					}
				}
			}
		}
		if meetingNode != "" { break }
		currentLevelForward++

		// --- Langkah Mundur ---
		lenB := queueBackward.Len()
		for i := 0; i < lenB && meetingNode == ""; i++ {
			if queueBackward.Len() == 0 { break }
			currB := queueBackward.Remove(queueBackward.Front()).(string)
			nodesVisitedCount++

			if visitedForward[currB] > 0 {
				// fmt.Printf("BDS: Pertemuan DARI BWD di '%s'\n", currB)
				meetingNode = currB
			}

			recipesMakingCurrB := recipeMap[currB]
			if _, exists := parentBackward[currB]; !exists && len(recipesMakingCurrB) > 0 {
				parentBackward[currB] = recipesMakingCurrB[0]
			}

			for _, recipe := range recipesMakingCurrB {
				ingredients := []string{recipe.Ingredient1, recipe.Ingredient2}
				for _, ing := range ingredients {
					if visitedBackward[ing] == 0 {
						visitedBackward[ing] = currentLevelBackward + 1
						queueBackward.PushBack(ing)

						if visitedForward[ing] > 0 && meetingNode == "" {
							// fmt.Printf("BDS: Pertemuan SETELAH BWD ekspansi di '%s'\n", ing)
							meetingNode = ing
						}
					}
				}
			}
		}
		if meetingNode != "" { break }
		currentLevelBackward++

		if queueForward.Len() == 0 || queueBackward.Len() == 0 { break }
	}
	// --- Akhir Loop BDS Utama ---

	if meetingNode == "" {
		fmt.Printf("Hybrid BDS+BFS: Tidak ada pertemuan ditemukan untuk '%s'.\n", targetElement)
		return nil, nodesVisitedCount, fmt.Errorf("jalur (BDS meeting) ke '%s' tidak ditemukan", targetElement)
	}

	fmt.Printf("Hybrid BDS+BFS: Pertemuan di '%s'. Memulai rekonstruksi dan pencarian BFS tambahan...\n", meetingNode)

	// --- Rekonstruksi dan Pencarian BFS Tambahan ---
	finalRecipe, finalRecipeExists := parentBackward[targetElement]
	if !finalRecipeExists {
		recipesForTarget := recipeMap[targetElement]
		if len(recipesForTarget) > 0 {
			finalRecipe = recipesForTarget[0]
			finalRecipeExists = true
			// fmt.Printf("  INFO: Menggunakan resep fallback untuk target '%s': %v\n", targetElement, finalRecipe)
		} else {
			fmt.Printf("  ERROR: Tidak dapat menemukan resep final untuk '%s'.\n", targetElement)
			return nil, nodesVisitedCount, fmt.Errorf("resep final untuk '%s' tidak ditemukan", targetElement)
		}
	}
	// fmt.Printf("  Resep Final: %s + %s => %s\n", finalRecipe.Ingredient1, finalRecipe.Ingredient2, finalRecipe.Result)

	ing1 := finalRecipe.Ingredient1
	ing2 := finalRecipe.Ingredient2

	combinedRecipes := make(map[string]Recipe) // Pindahkan deklarasi ke sini

	if meetingNode == ing1 || meetingNode == ing2 {
		// Kasus 1: Meeting node adalah salah satu bahan final
		var ingredientToSearchBFS string
		var pathForMeetingNodeSegment []Recipe

		if meetingNode == ing1 { ingredientToSearchBFS = ing2 } else { ingredientToSearchBFS = ing1 }

		fmt.Printf("  Merekonstruksi jalur FWD untuk meeting node '%s'...\n", meetingNode)
		stopAtBase := func(node string) bool { return isBaseElement(node) }
		pathForMeetingNodeSegment = reconstructSingleSegmentPath(parentForward, meetingNode, stopAtBase)
		fmt.Printf("  Jalur FWD untuk '%s' ditemukan (panjang: %d)\n", meetingNode, len(pathForMeetingNodeSegment))

		fmt.Printf("  Mencari jalur BFS untuk bahan '%s'\n", ingredientToSearchBFS)
		pathOtherIngredient, bfsNodes, errBFS := FindPathBFS(ingredientToSearchBFS)
		if errBFS != nil {
			fmt.Printf("  ERROR: Gagal mencari jalur BFS untuk '%s': %v\n", ingredientToSearchBFS, errBFS)
			return nil, nodesVisitedCount + bfsNodes, fmt.Errorf("gagal mencari jalur BFS untuk bahan '%s': %v", ingredientToSearchBFS, errBFS)
		}
		nodesVisitedCount += bfsNodes
		fmt.Printf("  Jalur BFS untuk '%s' ditemukan (panjang: %d)\n", ingredientToSearchBFS, len(pathOtherIngredient))

		// Gabungkan resep
		for _, r := range pathForMeetingNodeSegment { combinedRecipes[getUniqueRecipeKey(r)] = r }
		for _, r := range pathOtherIngredient { combinedRecipes[getUniqueRecipeKey(r)] = r }

	} else {
		// Kasus 2: Meeting node bukan bahan final (perlu BFS untuk keduanya)
		fmt.Printf("  PERINGATAN: Meeting node '%s' bukan bahan langsung. Mencari BFS untuk KEDUA bahan '%s' dan '%s'.\n", meetingNode, ing1, ing2)

		fmt.Printf("  Mencari jalur BFS untuk bahan 1: '%s'\n", ing1)
		pathIng1, bfsNodes1, err1 := FindPathBFS(ing1)
		if err1 != nil {
			fmt.Printf("  ERROR: Gagal mencari jalur BFS untuk '%s': %v\n", ing1, err1)
			return nil, nodesVisitedCount + bfsNodes1, fmt.Errorf("gagal mencari jalur BFS untuk bahan '%s': %v", ing1, err1)
		}
		nodesVisitedCount += bfsNodes1
		fmt.Printf("  Jalur BFS untuk '%s' ditemukan (panjang: %d)\n", ing1, len(pathIng1))
		for _, r := range pathIng1 { combinedRecipes[getUniqueRecipeKey(r)] = r }


		fmt.Printf("  Mencari jalur BFS untuk bahan 2: '%s'\n", ing2)
		pathIng2, bfsNodes2, err2 := FindPathBFS(ing2)
		if err2 != nil {
			fmt.Printf("  ERROR: Gagal mencari jalur BFS untuk '%s': %v\n", ing2, err2)
			return nil, nodesVisitedCount + bfsNodes2, fmt.Errorf("gagal mencari jalur BFS untuk bahan '%s': %v", ing2, err2)
		}
		nodesVisitedCount += bfsNodes2
		fmt.Printf("  Jalur BFS untuk '%s' ditemukan (panjang: %d)\n", ing2, len(pathIng2))
		for _, r := range pathIng2 { combinedRecipes[getUniqueRecipeKey(r)] = r }

		// Kita juga perlu jalur dari meeting node ke base dalam kasus ini
		fmt.Printf("  Merekonstruksi jalur FWD untuk meeting node '%s' (kasus 2)...\n", meetingNode)
		stopAtBase := func(node string) bool { return isBaseElement(node) }
		pathMeetingToBase := reconstructSingleSegmentPath(parentForward, meetingNode, stopAtBase)
		fmt.Printf("  Jalur FWD untuk '%s' ditemukan (panjang: %d)\n", meetingNode, len(pathMeetingToBase))
		for _, r := range pathMeetingToBase { combinedRecipes[getUniqueRecipeKey(r)] = r }
	}

	// Selalu tambahkan resep final
	combinedRecipes[getUniqueRecipeKey(finalRecipe)] = finalRecipe

	// --- Urutkan Resep Gabungan ---
	finalPathSorted := buildSortedPathFromRecipes(combinedRecipes, targetElement)

	// Validasi akhir (opsional)
	if len(finalPathSorted) == 0 && !isBaseElement(targetElement) {
		fmt.Printf("  PERINGATAN AKHIR: Jalur terurut kosong untuk target non-dasar '%s'.\n", targetElement)
		// Mungkin ada masalah dalam pengurutan atau resep yang hilang
	} else if len(finalPathSorted) > 0 && finalPathSorted[len(finalPathSorted)-1].Result != targetElement {
		fmt.Printf("  PERINGATAN AKHIR: Jalur terurut TIDAK menghasilkan target '%s'. Resep terakhir: %v\n", targetElement, finalPathSorted[len(finalPathSorted)-1].Result)
	}

	fmt.Printf("Hybrid BDS+BFS: Penggabungan dan pengurutan selesai. Total resep unik terurut: %d\n", len(finalPathSorted))
	return finalPathSorted, nodesVisitedCount, nil
}


// FindMultiplePathsBDS: Mencari beberapa jalur unik menggunakan konkurensi.
// (Fungsi ini tetap sama, hanya memanggil FindPathBDS yang sudah diubah)
func FindMultiplePathsBDS(targetElement string, maxRecipes int) ([][]Recipe, int, error) {
	fmt.Printf("BDS Multiple (Hybrid): Mencari %d jalur ke: %s (Multithreaded)\n", maxRecipes, targetElement)
	// fmt.Println("CATATAN: Implementasi BDS Multiple saat ini cenderung menemukan jalur terpendek yang sama.")

	if maxRecipes <= 0 {
		return nil, 0, errors.New("jumlah resep minimal harus 1")
	}
	if isBaseElement(targetElement) {
		return [][]Recipe{{}}, 0, nil
	}

	var allFoundPaths [][]Recipe
	addedPathIdentifiers := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup
	nodesVisitedTotal := atomic.Int32{}
	foundCount := atomic.Int32{}
	quitChan := make(chan struct{})
	var quitOnce sync.Once
	closeQuitChan := func() {
		quitOnce.Do(func() { close(quitChan) })
	}
	defer closeQuitChan()

	numGoroutines := maxRecipes
	if numGoroutines < 1 { numGoroutines = 1 }
	maxGo := 10
	if numGoroutines > maxGo { numGoroutines = maxGo }

	fmt.Printf("BDS Multiple (Hybrid): Meluncurkan %d goroutine...\n", numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		if foundCount.Load() >= int32(maxRecipes) { break }
		wg.Add(1)
		go func(goroutineIndex int) {
			defer wg.Done()
			// Setiap goroutine sekarang menjalankan FindPathBDS (Hybrid)
			// Path yang dikembalikan sudah diurutkan oleh buildSortedPathFromRecipes
			path, nodesVisited, err := FindPathBDS(targetElement)
			nodesVisitedTotal.Add(int32(nodesVisited))
			mu.Lock()
			defer mu.Unlock()
			select {
			case <-quitChan: return
			default:
			}
			if err == nil && path != nil {
				// Path sudah diurutkan, bisa langsung generate ID
				if len(path) > 0 {
					pathID := generatePathIdentifier(path) // ID dari path terurut
					if !addedPathIdentifiers[pathID] {
						if currentFound := foundCount.Load(); currentFound < int32(maxRecipes) {
							// Salin path sebelum menambahkannya
							pathToAppend := make([]Recipe, len(path))
							copy(pathToAppend, path)
							allFoundPaths = append(allFoundPaths, pathToAppend)
							addedPathIdentifiers[pathID] = true
							newCount := foundCount.Add(1)
							fmt.Printf("Goroutine Hybrid-%d: Jalur UNIK ditemukan (Panjang: %d). Total Ditemukan: %d/%d\n", goroutineIndex, len(pathToAppend), newCount, maxRecipes)
							if newCount >= int32(maxRecipes) { closeQuitChan() }
						}
					}
				}
			}
		}(i)
	}

	wg.Wait()
	mu.Lock()
	finalPathsToReturn := make([][]Recipe, len(allFoundPaths))
	copy(finalPathsToReturn, allFoundPaths)
	currentFoundCount := len(finalPathsToReturn)
	mu.Unlock()

	if currentFoundCount == 0 && !isBaseElement(targetElement) {
		return nil, int(nodesVisitedTotal.Load()), fmt.Errorf("tidak ada jalur Hybrid BDS+BFS (multiple) yang valid ditemukan untuk '%s'", targetElement)
	}

	// Urutkan hasil akhir berdasarkan panjang path (opsional)
	sort.SliceStable(finalPathsToReturn, func(i, j int) bool {
		return len(finalPathsToReturn[i]) < len(finalPathsToReturn[j])
	})


	fmt.Printf("BDS Multiple (Hybrid): Selesai. Total jalur unik ditemukan: %d (diminta: %d). Total nodes visited (approx): %d\n", currentFoundCount, maxRecipes, nodesVisitedTotal.Load())
	return finalPathsToReturn, int(nodesVisitedTotal.Load()), nil
}