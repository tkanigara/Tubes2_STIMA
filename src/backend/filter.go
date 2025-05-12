// src/backend/filter.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Recipe struct (sama)
// type Recipe struct {
// 	Result      string `json:"result"`
// 	Ingredient1 string `json:"ingredient1"`
// 	Ingredient2 string `json:"ingredient2"`
// }

func getRecipeID(r Recipe) string {
	ings := []string{r.Ingredient1, r.Ingredient2}
	sort.Strings(ings)
	return fmt.Sprintf("%s+%s=>%s", ings[0], ings[1], r.Result)
}

func runFilter() {
	baseDir := "data"
	rawRecipeFile := filepath.Join(baseDir, "recipes_scraped.json")
	filteredRecipeFile := filepath.Join(baseDir, "recipes_final_filtered.json")

	baseElements := []string{"Air", "Earth", "Fire", "Water"}

	fmt.Println("Memulai skrip filter resep lanjutan...")

	rawBytes, err := os.ReadFile(rawRecipeFile)
	if err != nil {
		fmt.Printf("Error membaca file resep mentah '%s': %v\n", rawRecipeFile, err)
		return
	}
	var initialRecipes []Recipe
	err = json.Unmarshal(rawBytes, &initialRecipes)
	if err != nil {
		fmt.Printf("Error unmarshal JSON resep mentah: %v\n", err)
		return
	}
	fmt.Printf("Berhasil memuat %d resep mentah.\n", len(initialRecipes))

	// Kumpulkan semua elemen unik dari data mentah
	initialElementsSet := make(map[string]bool)
	for _, base := range baseElements {
		initialElementsSet[base] = true
	}
	for _, recipe := range initialRecipes {
		initialElementsSet[recipe.Result] = true
		initialElementsSet[recipe.Ingredient1] = true
		initialElementsSet[recipe.Ingredient2] = true
	}
	fmt.Printf("Jumlah elemen unik awal (termasuk dasar): %d\n", len(initialElementsSet))


	allRemovedRecipesTracker := make(map[string]string)

	fmt.Println("\n--- TAHAP 1: Memfilter resep berdasarkan ketercapaian dari elemen dasar ---")
	recipesAfterStage1, removedInStage1 := filterUnmakeablePaths(initialRecipes, baseElements)
	fmt.Printf("Tahap 1: %d resep dihapus karena tidak tercapai dari dasar. Sisa: %d resep.\n", len(removedInStage1), len(recipesAfterStage1))
	for _, r := range removedInStage1 {
		allRemovedRecipesTracker[getRecipeID(r)] = "Tidak tercapai dari elemen dasar"
	}

	fmt.Println("\n--- TAHAP 2: Menghitung tier elemen ---")
	elementTiersStage2, _ := calculateElementTiers(recipesAfterStage1, baseElements)
	fmt.Printf("Tier dihitung untuk %d elemen (yang memiliki tier).\n", len(elementTiersStage2))

	fmt.Println("\n--- TAHAP 3: Memfilter resep berdasarkan validitas tier ---")
	recipesAfterStage3, removedInStage2 := filterByTierLogic(recipesAfterStage1, elementTiersStage2)
	fmt.Printf("Tahap 3: %d resep dihapus karena logika tier. Sisa: %d resep.\n", len(removedInStage2), len(recipesAfterStage3))
	for _, r := range removedInStage2 {
		tierR, _ := elementTiersStage2[r.Result]
		tierI1, _ := elementTiersStage2[r.Ingredient1]
		tierI2, _ := elementTiersStage2[r.Ingredient2]
		reason := fmt.Sprintf("Tier tidak valid (H:%d, B1:%d, B2:%d)", tierR, tierI1, tierI2)
		if _, exists := allRemovedRecipesTracker[getRecipeID(r)]; !exists {
			allRemovedRecipesTracker[getRecipeID(r)] = reason
		}
	}

	fmt.Println("\n--- TAHAP 4: Iterasi ulang filter ketercapaian dan tier ---")
	previousRecipeCount := -1
	currentIterationRecipes := recipesAfterStage3
	finalIteration := 0
	maxFinalIterations := 5

	for len(currentIterationRecipes) != previousRecipeCount && finalIteration < maxFinalIterations {
		finalIteration++
		fmt.Printf("Iterasi Finalisasi Filter - Putaran %d\n", finalIteration)
		previousRecipeCount = len(currentIterationRecipes)

		makeableInLoop, removedInUnmakeablePass := filterUnmakeablePaths(currentIterationRecipes, baseElements)
		for _, r := range removedInUnmakeablePass {
			if _, exists := allRemovedRecipesTracker[getRecipeID(r)]; !exists {
				allRemovedRecipesTracker[getRecipeID(r)] = fmt.Sprintf("Tidak tercapai (Putaran Finalisasi %d)", finalIteration)
			}
		}
		
		tiersInLoop, _ := calculateElementTiers(makeableInLoop, baseElements)
		
		currentIterationRecipes, removedInTierPass := filterByTierLogic(makeableInLoop, tiersInLoop)
		for _, r := range removedInTierPass {
			tierR, _ := tiersInLoop[r.Result]
			tierI1, _ := tiersInLoop[r.Ingredient1]
			tierI2, _ := tiersInLoop[r.Ingredient2]
			reason := fmt.Sprintf("Tier tidak valid (Putaran Finalisasi %d - H:%d, B1:%d, B2:%d)",
				 finalIteration, tierR, tierI1, tierI2)
			if _, exists := allRemovedRecipesTracker[getRecipeID(r)]; !exists {
				allRemovedRecipesTracker[getRecipeID(r)] = reason
			}
		}

		if len(currentIterationRecipes) == previousRecipeCount {
			fmt.Println("Finalisasi filter konvergen.")
			break
		}
		fmt.Printf("  Setelah putaran %d finalisasi, tersisa %d resep.\n", finalIteration, len(currentIterationRecipes))
	}
	if finalIteration >= maxFinalIterations && len(currentIterationRecipes) != previousRecipeCount {
	    fmt.Println("Peringatan: Finalisasi filter mencapai batas iterasi maksimum sebelum konvergen.")
    }
	finalValidRecipes := currentIterationRecipes

	if len(allRemovedRecipesTracker) > 0 {
		fmt.Printf("\n--- Daftar Resep yang Dihapus (%d total dari semua tahap) ---\n", len(allRemovedRecipesTracker))
		var sortedRemovedIDs []string
		for id := range allRemovedRecipesTracker {
			sortedRemovedIDs = append(sortedRemovedIDs, id)
		}
		sort.Strings(sortedRemovedIDs)
		for _, id := range sortedRemovedIDs {
			fmt.Printf("  %s (Alasan: %s)\n", id, allRemovedRecipesTracker[id])
		}
	} else {
		fmt.Println("\nTidak ada resep yang dihapus selama proses filter.")
	}

	finalValidElementsSet := make(map[string]bool)
	for _, base := range baseElements {
		finalValidElementsSet[base] = true
	}
	for _, recipe := range finalValidRecipes {
		finalValidElementsSet[recipe.Result] = true
		finalValidElementsSet[recipe.Ingredient1] = true
		finalValidElementsSet[recipe.Ingredient2] = true
	}
	fmt.Printf("\nJumlah elemen unik yang valid setelah semua filter: %d\n", len(finalValidElementsSet))

	// --- TAMBAHAN: Identifikasi dan cetak elemen yang dihilangkan ---
	var removedElementsList []string
	for initialEl := range initialElementsSet {
		if !finalValidElementsSet[initialEl] {
			removedElementsList = append(removedElementsList, initialEl)
		}
	}
	sort.Strings(removedElementsList) // Urutkan untuk output yang konsisten

	if len(removedElementsList) > 0 {
		fmt.Printf("\n--- Daftar Elemen yang Dihilangkan (%d total) ---\n", len(removedElementsList))
		for i, el := range removedElementsList {
			fmt.Printf("  %d. %s\n", i+1, el)
		}
	} else {
		fmt.Println("\nTidak ada elemen yang dihilangkan (semua elemen awal masih valid atau merupakan bagian dari resep valid).")
	}
	// --- AKHIR TAMBAHAN ---


	filteredBytes, err := json.MarshalIndent(finalValidRecipes, "", "  ")
	if err != nil {
		fmt.Printf("Error marshal JSON resep terfilter akhir: %v\n", err)
		return
	}
	err = os.WriteFile(filteredRecipeFile, filteredBytes, 0644)
	if err != nil {
		fmt.Printf("Error menulis JSON resep terfilter akhir ke file '%s': %v\n", filteredRecipeFile, err)
		return
	}

	fmt.Printf("\nProses filter keseluruhan selesai. %d resep valid disimpan ke '%s'.\n", len(finalValidRecipes), filteredRecipeFile)
}

// filterUnmakeablePaths (sama seperti versi sebelumnya)
func filterUnmakeablePaths(recipesToFilter []Recipe, baseElements []string) ([]Recipe, []Recipe) {
	currentRecipes := make([]Recipe, len(recipesToFilter))
	copy(currentRecipes, recipesToFilter)
	var removedInThisCall []Recipe

	initialRecipeIDs := make(map[string]Recipe) 
    for _, r := range recipesToFilter {
        initialRecipeIDs[getRecipeID(r)] = r
    }

	iteration := 0
	for {
		iteration++
		previousRecipeCount := len(currentRecipes)
		makeableElements := make(map[string]bool)
		for _, base := range baseElements { makeableElements[base] = true }

		propagationPass := 0
		for {
			propagationPass++
			madeChangeThisPass := false
			if len(currentRecipes) == 0 && propagationPass > 1 { break }

			for _, recipe := range currentRecipes {
				if makeableElements[recipe.Ingredient1] && makeableElements[recipe.Ingredient2] {
					if !makeableElements[recipe.Result] {
						makeableElements[recipe.Result] = true
						madeChangeThisPass = true
					}
				}
			}
			if !madeChangeThisPass || propagationPass > len(recipesToFilter)+len(baseElements)+10 { break }
		}

		var nextValidRecipes []Recipe
		for _, recipe := range currentRecipes {
			if makeableElements[recipe.Ingredient1] && makeableElements[recipe.Ingredient2] {
				nextValidRecipes = append(nextValidRecipes, recipe)
			}
		}
		currentRecipes = nextValidRecipes

		if len(currentRecipes) == previousRecipeCount { break }
		if iteration > 30 { fmt.Println("    Peringatan: Filter Ketercapaian melebihi batas iterasi (30)."); break; }
	}
    
    finalValidRecipeIDs := make(map[string]bool)
    for _, r := range currentRecipes {
        finalValidRecipeIDs[getRecipeID(r)] = true
    }
    for id, originalRecipe := range initialRecipeIDs {
        if !finalValidRecipeIDs[id] {
            removedInThisCall = append(removedInThisCall, originalRecipe)
        }
    }
	return currentRecipes, removedInThisCall
}

// calculateElementTiers (sama seperti versi sebelumnya)
func calculateElementTiers(recipesForTierCalc []Recipe, baseElements []string) (map[string]int, map[string]bool) {
	elementTiers := make(map[string]int)
	allInvolvedElements := make(map[string]bool) 

	for _, base := range baseElements {
		elementTiers[base] = 0
		allInvolvedElements[base] = true
	}

	recipesByResult := make(map[string][]Recipe)
	for _, r := range recipesForTierCalc {
		recipesByResult[r.Result] = append(recipesByResult[r.Result], r)
		allInvolvedElements[r.Result] = true
		allInvolvedElements[r.Ingredient1] = true
		allInvolvedElements[r.Ingredient2] = true
	}

	if len(recipesForTierCalc) == 0 {
		return elementTiers, allInvolvedElements
	}
	
	maxTierIterations := len(allInvolvedElements) + 10 
	if maxTierIterations < 50 { maxTierIterations = 50 }

	for iter := 0; iter < maxTierIterations; iter++ {
		changedInThisIteration := false
		for el := range allInvolvedElements {
			if isBase(el, baseElements) {
				continue
			}
			minTierForEl := -1 
			if recipes, ok := recipesByResult[el]; ok {
				for _, recipe := range recipes {
					tierIng1, ing1HasTier := elementTiers[recipe.Ingredient1]
					tierIng2, ing2HasTier := elementTiers[recipe.Ingredient2]
					if ing1HasTier && ing2HasTier {
						currentRecipeProducesTier := 1 + max(tierIng1, tierIng2)
						if minTierForEl == -1 || currentRecipeProducesTier < minTierForEl {
							minTierForEl = currentRecipeProducesTier
						}
					}
				}
			}
			currentStoredTier, elHadTier := elementTiers[el]
			if minTierForEl != -1 {
				if !elHadTier || minTierForEl < currentStoredTier {
					elementTiers[el] = minTierForEl
					changedInThisIteration = true
				}
			}
		}
		if !changedInThisIteration {
			break
		}
        if iter == maxTierIterations -1 {
             fmt.Printf("  Peringatan: Perhitungan tier mungkin mencapai batas iterasi kalkulasi (%d).\n", maxTierIterations)
        }
	}
    
	defaultHighTier := len(recipesForTierCalc) + 2
	for el := range allInvolvedElements {
		if _, hasTier := elementTiers[el]; !hasTier {
			elementTiers[el] = defaultHighTier
		}
	}
	return elementTiers, allInvolvedElements
}

// filterByTierLogic (sama seperti versi sebelumnya)
func filterByTierLogic(recipes []Recipe, tiers map[string]int) ([]Recipe, []Recipe) {
	var validRecipes []Recipe
	var removedRecipes []Recipe
	
	maxKnownTier := 0
    for _, t := range tiers { if t > maxKnownTier { maxKnownTier = t } }
    defaultTierForUnknown := maxKnownTier + 2

	for _, recipe := range recipes {
		tierResult, okR := tiers[recipe.Result]
		if !okR { tierResult = defaultTierForUnknown }
		tierIng1, okI1 := tiers[recipe.Ingredient1]
		if !okI1 { tierIng1 = defaultTierForUnknown }
		tierIng2, okI2 := tiers[recipe.Ingredient2]
		if !okI2 { tierIng2 = defaultTierForUnknown }

		if tierIng1 > tierResult || tierIng2 > tierResult {
			removedRecipes = append(removedRecipes, recipe)
		} else {
			validRecipes = append(validRecipes, recipe)
		}
	}
	return validRecipes, removedRecipes
}

// isBase dan max (sama seperti versi sebelumnya)
func isBase(elementName string, baseElements []string) bool {
    for _, b := range baseElements { if b == elementName { return true } }
    return false
}

func max(a, b int) int {
	if a > b { return a }
	return b
}