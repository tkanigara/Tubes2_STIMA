// src/backend/dfs.go
package main

import (
	//"container/list" // Digunakan oleh reconstructPathRevised (jika FindPathDFS masih pakai)
	"errors"
	"fmt"
	"sort" // Diperlukan untuk generatePathIdentifier jika dipindah ke sini
	"strings" // Diperlukan untuk generatePathIdentifier jika dipindah ke sini
	"sync" // Import sync untuk Mutex dan WaitGroup
)

func FindPathDFS(targetElement string) ([]Recipe, int, error) {
    fmt.Printf("Mencari jalur DFS (single) ke: %s\n", targetElement)

    // Persiapan
    recipeMap := GetRecipeMap()
    if recipeMap == nil {
        return nil, 0, errors.New("map resep belum diinisialisasi")
    }
    
    if isBaseElementDFS(targetElement) {
        return []Recipe{}, 0, nil // Target adalah elemen dasar
    }
    
    nodesVisitedCount := 0
    
    // Cache untuk elemen yang bisa dibuat
    knownCreatableElements := make(map[string]bool)
    for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
        knownCreatableElements[base] = true
    }
    
    // Cache untuk jalur ke setiap elemen
    pathCache := make(map[string][]Recipe)
    
    // Fungsi untuk memeriksa apakah elemen bisa dibuat
    var isCreatable func(element string, visited map[string]bool, depth int) bool
    isCreatable = func(element string, visited map[string]bool, depth int) bool {
        nodesVisitedCount++
        
        // Batasi kedalaman rekursi untuk mencegah stack overflow
        if depth > 500 {
            return false
        }
        
        // Base case 1: Jika elemen dasar
        if isBaseElementDFS(element) {
            return true
        }
        
        // Base case 2: Sudah kita ketahui bisa dibuat
        if known, exists := knownCreatableElements[element]; exists {
            return known
        }
        
        // Base case 3: Loop deteksi - dengan pengecualian untuk kedalaman rendah
        if depth > 30 && visited[element] {
            return false
        }
        
        // Tandai sudah dikunjungi untuk cabang ini
        newVisited := make(map[string]bool)
        for k, v := range visited {
            newVisited[k] = v
        }
        newVisited[element] = true
        
        // Cek setiap resep yang menghasilkan elemen ini
        recipes := recipeMap[element]
        if len(recipes) == 0 {
            // Simpan hasil: tidak bisa dibuat
            knownCreatableElements[element] = false
            return false
        }
        
        // Cek apakah ada resep yang valid (kedua bahannya bisa dibuat)
        for _, recipe := range recipes {
            ing1Creatable := isCreatable(recipe.Ingredient1, newVisited, depth+1)
            ing2Creatable := isCreatable(recipe.Ingredient2, newVisited, depth+1)
            
            if ing1Creatable && ing2Creatable {
                // Simpan hasil: bisa dibuat
                knownCreatableElements[element] = true
                return true
            }
        }
        
        // Simpan hasil: tidak bisa dibuat
        knownCreatableElements[element] = false
        return false
    }
    
    // Fungsi untuk membuat jalur berurutan dengan semua prasyarat (bottom-up DFS)
    var buildOrderedPath func(target string, availableElements map[string]bool, visited map[string]bool) []Recipe
    buildOrderedPath = func(target string, availableElements map[string]bool, visited map[string]bool) []Recipe {
        nodesVisitedCount++
        // Jika elemen dasar atau sudah tersedia, tidak perlu membuat
        if isBaseElementDFS(target) || availableElements[target] {
            return []Recipe{}
        }
        
        // Cek apakah ada di cache
        if path, exists := pathCache[target]; exists {
            // Verifikasi semua elemen dalam jalur cache
            clonedAvailable := make(map[string]bool)
            for k, v := range availableElements {
                clonedAvailable[k] = v
            }
            
            // Cek apakah jalur dari cache valid dengan elemen yang tersedia saat ini
            valid := true
            for _, recipe := range path {
                if !isBaseElementDFS(recipe.Ingredient1) && !clonedAvailable[recipe.Ingredient1] {
                    valid = false
                    break
                }
                if !isBaseElementDFS(recipe.Ingredient2) && !clonedAvailable[recipe.Ingredient2] {
                    valid = false
                    break
                }
                clonedAvailable[recipe.Result] = true
            }
            
            if valid {
                // Jalur cache valid, gunakan
                pathCopy := make([]Recipe, len(path))
                copy(pathCopy, path)
                return pathCopy
            }
            // Jalur cache tidak valid dengan elemen yang tersedia saat ini, lanjutkan cari
        }
        
        // Deteksi loop
        if visited[target] {
            return nil
        }
        
        // Tandai sebagai dikunjungi untuk mencegah loop
        newVisited := make(map[string]bool)
        for k, v := range visited {
            newVisited[k] = v
        }
        newVisited[target] = true
        
        // Cari resep yang bisa membuat target
        recipes := recipeMap[target]
        if len(recipes) == 0 {
            return nil
        }
        
        // Urutkan resep (prioritaskan yang bisa langsung dibuat)
        sort.Slice(recipes, func(i, j int) bool {
            iCanMake := (isBaseElementDFS(recipes[i].Ingredient1) || availableElements[recipes[i].Ingredient1]) &&
                        (isBaseElementDFS(recipes[i].Ingredient2) || availableElements[recipes[i].Ingredient2])
            jCanMake := (isBaseElementDFS(recipes[j].Ingredient1) || availableElements[recipes[j].Ingredient1]) &&
                        (isBaseElementDFS(recipes[j].Ingredient2) || availableElements[recipes[j].Ingredient2])
            
            if iCanMake && !jCanMake {
                return true
            }
            if !iCanMake && jCanMake {
                return false
            }
            
            // Jika keduanya sama, prioritas ke yang menggunakan elemen dasar
            iBaseCount := 0
            jBaseCount := 0
            
            if isBaseElementDFS(recipes[i].Ingredient1) {
                iBaseCount++
            }
            if isBaseElementDFS(recipes[i].Ingredient2) {
                iBaseCount++
            }
            if isBaseElementDFS(recipes[j].Ingredient1) {
                jBaseCount++
            }
            if isBaseElementDFS(recipes[j].Ingredient2) {
                jBaseCount++
            }
            
            if iBaseCount != jBaseCount {
                return iBaseCount > jBaseCount
            }
            
            return recipes[i].Result < recipes[j].Result // Stabil sort
        })
        
        // Cari jalur terbaik
        var bestPath []Recipe
        
        for _, recipe := range recipes {
            // Buat salinan daftar elemen yang tersedia
            elementsAvailable := make(map[string]bool)
            for k, v := range availableElements {
                elementsAvailable[k] = v
            }
            
            // Rangkai jalur untuk membuat kedua bahan secara berurutan
            // Kita akan memastikan bahan pertama dibuat, lalu bahan kedua
            
            // 1. Cek dan buat bahan pertama jika perlu
            var path1 []Recipe
            if !isBaseElementDFS(recipe.Ingredient1) && !elementsAvailable[recipe.Ingredient1] {
                path1 = buildOrderedPath(recipe.Ingredient1, elementsAvailable, newVisited)
                if path1 == nil {
                    continue // Tidak bisa membuat bahan pertama, coba resep lain
                }
                
                // Perbarui daftar elemen tersedia setelah membuat bahan pertama
                for _, p := range path1 {
                    elementsAvailable[p.Result] = true
                }
            }
            
            // 2. Cek dan buat bahan kedua jika perlu
            var path2 []Recipe
            if !isBaseElementDFS(recipe.Ingredient2) && !elementsAvailable[recipe.Ingredient2] {
                path2 = buildOrderedPath(recipe.Ingredient2, elementsAvailable, newVisited)
                if path2 == nil {
                    continue // Tidak bisa membuat bahan kedua, coba resep lain
                }
                
                // Perbarui daftar elemen tersedia setelah membuat bahan kedua
                for _, p := range path2 {
                    elementsAvailable[p.Result] = true
                }
            }
            
            // 3. Buat target dengan resep saat ini
            
            // Cek sekali lagi apakah kedua bahan tersedia (karena loop mungkin terjadi)
            if (!isBaseElementDFS(recipe.Ingredient1) && !elementsAvailable[recipe.Ingredient1]) ||
               (!isBaseElementDFS(recipe.Ingredient2) && !elementsAvailable[recipe.Ingredient2]) {
                continue // Ada masalah dengan ketersediaan bahan, coba resep lain
            }
            
            // Rangkai jalur lengkap: buat bahan 1, buat bahan 2, buat target
            completePath := make([]Recipe, 0)
            
            if path1 != nil {
                completePath = append(completePath, path1...)
            }
            
            if path2 != nil {
                completePath = append(completePath, path2...)
            }
            
            completePath = append(completePath, recipe)
            
            // Evaluasi jalur ini vs yang terbaik sejauh ini
            if bestPath == nil || len(completePath) < len(bestPath) {
                bestPath = completePath
            }
        }
        
        // Simpan hasil ke cache jika ditemukan jalur
        if bestPath != nil {
            pathCopy := make([]Recipe, len(bestPath))
            copy(pathCopy, bestPath)
            pathCache[target] = pathCopy
        }
        
        return bestPath
    }
    
    // Helper untuk menghapus resep duplikat dalam path
    //var removeDuplicateRecipes func(path []Recipe) []Recipe
    var removeDuplicateRecipes = func(path []Recipe) []Recipe {
        seen := make(map[string]bool)
        unique := make([]Recipe, 0, len(path))
        
        for _, recipe := range path {
            key := fmt.Sprintf("%s:%s+%s", recipe.Result, recipe.Ingredient1, recipe.Ingredient2)
            if !seen[key] {
                seen[key] = true
                unique = append(unique, recipe)
            }
        }
        
        return unique
    }
    
    // Tentukan elemen dasar yang tersedia
    availableElements := make(map[string]bool)
    for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
        availableElements[base] = true
    }
    
    // Cari jalur optimal
    fmt.Printf("Mencari jalur optimal untuk %s...\n", targetElement)
    optimalPath := buildOrderedPath(targetElement, availableElements, make(map[string]bool))
    
    if optimalPath == nil {
        return nil, nodesVisitedCount, fmt.Errorf("tidak ada jalur valid untuk membuat %s", targetElement)
    }
    
    // Hilangkan duplikat
    optimalPath = removeDuplicateRecipes(optimalPath)
    
    // Verifikasi jalur optimal
    available := make(map[string]bool)
    for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
        available[base] = true
    }
    
    for i, recipe := range optimalPath {
        // Debug: cek prasyarat tersedia
        if !isBaseElementDFS(recipe.Ingredient1) && !available[recipe.Ingredient1] {
            fmt.Printf("PERINGATAN: Jalur optimal - bahan %s tidak tersedia pada langkah %d\n", 
                       recipe.Ingredient1, i+1)
        }
        
        if !isBaseElementDFS(recipe.Ingredient2) && !available[recipe.Ingredient2] {
            fmt.Printf("PERINGATAN: Jalur optimal - bahan %s tidak tersedia pada langkah %d\n", 
                       recipe.Ingredient2, i+1)
        }
        
        // Tandai hasil sebagai tersedia
        available[recipe.Result] = true
    }
    
    // PERUBAHAN: Balik urutan jalur sebelum menampilkan & mengembalikan
    // Note: Kita tidak perlu membalik urutan karena buildOrderedPath sudah memberikan
    // jalur dengan urutan yang benar (dari bawah ke atas)
    
    // Debug - tampilkan jalur yang ditemukan
    fmt.Printf("Jalur DFS (single) (panjang: %d):\n", len(optimalPath))
    for i, recipe := range optimalPath {
        fmt.Printf("  Langkah %d: %s + %s => %s\n", 
                  i+1, recipe.Ingredient1, recipe.Ingredient2, recipe.Result)
    }
    
    return optimalPath, nodesVisitedCount, nil
}


func FindMultiplePathsDFS(targetElement string, maxRecipes int) ([][]Recipe, int, error) {
    fmt.Printf("Mencari %d jalur DFS BERBEDA ke: %s dengan multithreading (Super Robust)\n", maxRecipes, targetElement)

    // Akses data yang diperlukan
    recipeMap := GetRecipeMap()
    if recipeMap == nil {
        return nil, 0, errors.New("map resep belum diinisialisasi")
    }
    if maxRecipes <= 0 {
        return nil, 0, errors.New("jumlah resep minimal harus 1")
    }
    if isBaseElementDFS(targetElement) {
        return [][]Recipe{}, 0, nil
    }

    nodesVisitedCount := 0
    
    // Cache untuk elemen yang bisa dibuat
    knownCreatableElements := make(map[string]bool)
    for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
        knownCreatableElements[base] = true
    }
    var knownCreatableMutex sync.RWMutex

    // Cache untuk jalur ke setiap elemen
    pathCache := make(map[string][]Recipe)
    var pathCacheMutex sync.RWMutex
    
    // Fungsi untuk memeriksa apakah elemen bisa dibuat
    var isCreatable func(element string, visited map[string]bool, depth int) bool
    isCreatable = func(element string, visited map[string]bool, depth int) bool {
        nodesVisitedCount++
        
        // Batasi kedalaman rekursi untuk mencegah stack overflow
        if depth > 500 {
            return false
        }
        
        // Base case 1: Jika elemen dasar
        if isBaseElementDFS(element) {
            return true
        }
        
        // Base case 2: Sudah kita ketahui bisa dibuat
        knownCreatableMutex.RLock()
        if known, exists := knownCreatableElements[element]; exists {
            knownCreatableMutex.RUnlock()
            return known
        }
        knownCreatableMutex.RUnlock()
        
        // Base case 3: Loop deteksi - dengan pengecualian untuk kedalaman rendah
        if depth > 30 && visited[element] {
            return false
        }
        
        // Tandai sudah dikunjungi untuk cabang ini
        newVisited := make(map[string]bool)
        for k, v := range visited {
            newVisited[k] = v
        }
        newVisited[element] = true
        
        // Cek setiap resep yang menghasilkan elemen ini
        recipes := recipeMap[element]
        if len(recipes) == 0 {
            // Simpan hasil: tidak bisa dibuat
            knownCreatableMutex.Lock()
            knownCreatableElements[element] = false
            knownCreatableMutex.Unlock()
            return false
        }
        
        // Cek apakah ada resep yang valid (kedua bahannya bisa dibuat)
        for _, recipe := range recipes {
            ing1Creatable := isCreatable(recipe.Ingredient1, newVisited, depth+1)
            ing2Creatable := isCreatable(recipe.Ingredient2, newVisited, depth+1)
            
            if ing1Creatable && ing2Creatable {
                // Simpan hasil: bisa dibuat
                knownCreatableMutex.Lock()
                knownCreatableElements[element] = true
                knownCreatableMutex.Unlock()
                return true
            }
        }
        
        // Simpan hasil: tidak bisa dibuat
        knownCreatableMutex.Lock()
        knownCreatableElements[element] = false
        knownCreatableMutex.Unlock()
        return false
    }
    
    // Fungsi untuk membuat jalur berurutan dengan semua prasyarat (bottom-up DFS)
    var buildOrderedPath func(target string, availableElements map[string]bool, visited map[string]bool) []Recipe
    buildOrderedPath = func(target string, availableElements map[string]bool, visited map[string]bool) []Recipe {
        nodesVisitedCount++
        // Jika elemen dasar atau sudah tersedia, tidak perlu membuat
        if isBaseElementDFS(target) || availableElements[target] {
            return []Recipe{}
        }
        
        // Cek apakah ada di cache
        pathCacheMutex.RLock()
        if path, exists := pathCache[target]; exists {
            pathCacheMutex.RUnlock()
            
            // Verifikasi semua elemen dalam jalur cache
            clonedAvailable := make(map[string]bool)
            for k, v := range availableElements {
                clonedAvailable[k] = v
            }
            
            // Cek apakah jalur dari cache valid dengan elemen yang tersedia saat ini
            valid := true
            for _, recipe := range path {
                if !isBaseElementDFS(recipe.Ingredient1) && !clonedAvailable[recipe.Ingredient1] {
                    valid = false
                    break
                }
                if !isBaseElementDFS(recipe.Ingredient2) && !clonedAvailable[recipe.Ingredient2] {
                    valid = false
                    break
                }
                clonedAvailable[recipe.Result] = true
            }
            
            if valid {
                // Jalur cache valid, gunakan
                pathCopy := make([]Recipe, len(path))
                copy(pathCopy, path)
                return pathCopy
            }
            // Jalur cache tidak valid dengan elemen yang tersedia saat ini, lanjutkan cari
        } else {
            pathCacheMutex.RUnlock()
        }
        
        // Deteksi loop
        if visited[target] {
            return nil
        }
        
        // Tandai sebagai dikunjungi untuk mencegah loop
        newVisited := make(map[string]bool)
        for k, v := range visited {
            newVisited[k] = v
        }
        newVisited[target] = true
        
        // Cari resep yang bisa membuat target
        recipes := recipeMap[target]
        if len(recipes) == 0 {
            return nil
        }
        
        // Urutkan resep (prioritaskan yang bisa langsung dibuat)
        sort.Slice(recipes, func(i, j int) bool {
            iCanMake := (isBaseElementDFS(recipes[i].Ingredient1) || availableElements[recipes[i].Ingredient1]) &&
                        (isBaseElementDFS(recipes[i].Ingredient2) || availableElements[recipes[i].Ingredient2])
            jCanMake := (isBaseElementDFS(recipes[j].Ingredient1) || availableElements[recipes[j].Ingredient1]) &&
                        (isBaseElementDFS(recipes[j].Ingredient2) || availableElements[recipes[j].Ingredient2])
            
            if iCanMake && !jCanMake {
                return true
            }
            if !iCanMake && jCanMake {
                return false
            }
            
            // Jika keduanya sama, prioritas ke yang menggunakan elemen dasar
            iBaseCount := 0
            jBaseCount := 0
            
            if isBaseElementDFS(recipes[i].Ingredient1) {
                iBaseCount++
            }
            if isBaseElementDFS(recipes[i].Ingredient2) {
                iBaseCount++
            }
            if isBaseElementDFS(recipes[j].Ingredient1) {
                jBaseCount++
            }
            if isBaseElementDFS(recipes[j].Ingredient2) {
                jBaseCount++
            }
            
            if iBaseCount != jBaseCount {
                return iBaseCount > jBaseCount
            }
            
            return recipes[i].Result < recipes[j].Result // Stabil sort
        })
        
        // Cari jalur terbaik
        var bestPath []Recipe
        
        for _, recipe := range recipes {
            // Buat salinan daftar elemen yang tersedia
            elementsAvailable := make(map[string]bool)
            for k, v := range availableElements {
                elementsAvailable[k] = v
            }
            
            // Rangkai jalur untuk membuat kedua bahan secara berurutan
            // Kita akan memastikan bahan pertama dibuat, lalu bahan kedua
            
            // 1. Cek dan buat bahan pertama jika perlu
            var path1 []Recipe
            if !isBaseElementDFS(recipe.Ingredient1) && !elementsAvailable[recipe.Ingredient1] {
                path1 = buildOrderedPath(recipe.Ingredient1, elementsAvailable, newVisited)
                if path1 == nil {
                    continue // Tidak bisa membuat bahan pertama, coba resep lain
                }
                
                // Perbarui daftar elemen tersedia setelah membuat bahan pertama
                for _, p := range path1 {
                    elementsAvailable[p.Result] = true
                }
            }
            
            // 2. Cek dan buat bahan kedua jika perlu
            var path2 []Recipe
            if !isBaseElementDFS(recipe.Ingredient2) && !elementsAvailable[recipe.Ingredient2] {
                path2 = buildOrderedPath(recipe.Ingredient2, elementsAvailable, newVisited)
                if path2 == nil {
                    continue // Tidak bisa membuat bahan kedua, coba resep lain
                }
                
                // Perbarui daftar elemen tersedia setelah membuat bahan kedua
                for _, p := range path2 {
                    elementsAvailable[p.Result] = true
                }
            }
            
            // 3. Buat target dengan resep saat ini
            
            // Cek sekali lagi apakah kedua bahan tersedia (karena loop mungkin terjadi)
            if (!isBaseElementDFS(recipe.Ingredient1) && !elementsAvailable[recipe.Ingredient1]) ||
               (!isBaseElementDFS(recipe.Ingredient2) && !elementsAvailable[recipe.Ingredient2]) {
                continue // Ada masalah dengan ketersediaan bahan, coba resep lain
            }
            
            // Rangkai jalur lengkap: buat bahan 1, buat bahan 2, buat target
            completePath := make([]Recipe, 0)
            
            if path1 != nil {
                completePath = append(completePath, path1...)
            }
            
            if path2 != nil {
                completePath = append(completePath, path2...)
            }
            
            completePath = append(completePath, recipe)
            
            // Evaluasi jalur ini vs yang terbaik sejauh ini
            if bestPath == nil || len(completePath) < len(bestPath) {
                bestPath = completePath
            }
        }
        
        // Simpan hasil ke cache jika ditemukan jalur
        if bestPath != nil {
            pathCopy := make([]Recipe, len(bestPath))
            copy(pathCopy, bestPath)
            
            pathCacheMutex.Lock()
            pathCache[target] = pathCopy
            pathCacheMutex.Unlock()
        }
        
        return bestPath
    }
    
    // Helper untuk menghapus resep duplikat dalam path
    //var removeDuplicateRecipes func(path []Recipe) []Recipe
    var removeDuplicateRecipes = func(path []Recipe) []Recipe {
        seen := make(map[string]bool)
        unique := make([]Recipe, 0, len(path))
        
        for _, recipe := range path {
            key := fmt.Sprintf("%s:%s+%s", recipe.Result, recipe.Ingredient1, recipe.Ingredient2)
            if !seen[key] {
                seen[key] = true
                unique = append(unique, recipe)
            }
        }
        
        return unique
    }
    
    // Fungsi untuk mencari jalur alternatif
    //var findAlternativePaths func(target string, existingPath []Recipe, maxPaths int) [][]Recipe
    var findAlternativePaths = func(target string, existingPath []Recipe, maxPaths int) [][]Recipe {
        results := [][]Recipe{existingPath} // Mulai dengan jalur yang ada
        uniquePathMap := make(map[string]bool)
        
        // Tandai jalur yang ada sebagai sudah dilihat
        existingPathID := generatePathIdentifierDFS(existingPath)
        uniquePathMap[existingPathID] = true
        
        // Batasi jumlah goroutines berjalan simultan
        var wg sync.WaitGroup
        semaphore := make(chan struct{}, 8)
        var resultsMutex sync.Mutex
        
        // Cari jalur alternatif dengan mencoba semua resep untuk target
        recipesForTarget := recipeMap[target]
        
        for _, recipe := range recipesForTarget {
            // Skip jika resep sama dengan yang digunakan di jalur yang ada
            // Asumsi: resep untuk target ada di akhir jalur
            if len(existingPath) > 0 {
                lastRecipe := existingPath[len(existingPath)-1]
                if lastRecipe.Result == recipe.Result &&
                   lastRecipe.Ingredient1 == recipe.Ingredient1 &&
                   lastRecipe.Ingredient2 == recipe.Ingredient2 {
                    continue
                }
            }
            
            wg.Add(1)
            go func(r Recipe) {
                semaphore <- struct{}{} // Ambil token
                defer func() {
                    <-semaphore // Kembalikan token
                    wg.Done()
                }()
                
                // Inisialisasi dengan elemen dasar tersedia
                availableElements := make(map[string]bool)
                for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
                    availableElements[base] = true
                }
                
                // Verifikasi bahan-bahan dapat dibuat
                if !isCreatable(r.Ingredient1, make(map[string]bool), 0) || 
                   !isCreatable(r.Ingredient2, make(map[string]bool), 0) {
                    return // Resep tidak valid
                }
                
                // Buat jalur lengkap dari awal untuk resep alternatif ini
                var completePath []Recipe
                
                // Cari jalur untuk bahan pertama jika perlu
                if !isBaseElementDFS(r.Ingredient1) {
                    ing1Path := buildOrderedPath(r.Ingredient1, availableElements, make(map[string]bool))
                    if ing1Path == nil {
                        return // Tidak bisa membuat bahan pertama
                    }
                    
                    completePath = append(completePath, ing1Path...)
                    
                    // Perbarui elemen tersedia
                    for _, p := range ing1Path {
                        availableElements[p.Result] = true
                    }
                }
                
                // Cari jalur untuk bahan kedua jika perlu
                if !isBaseElementDFS(r.Ingredient2) && !availableElements[r.Ingredient2] {
                    ing2Path := buildOrderedPath(r.Ingredient2, availableElements, make(map[string]bool))
                    if ing2Path == nil {
                        return // Tidak bisa membuat bahan kedua
                    }
                    
                    completePath = append(completePath, ing2Path...)
                    
                    // Perbarui elemen tersedia
                    for _, p := range ing2Path {
                        availableElements[p.Result] = true
                    }
                }
                
                // Tambahkan resep target
                completePath = append(completePath, r)
                
                // Hilangkan duplikat
                finalPath := removeDuplicateRecipes(completePath)
                
                // Verifikasi jalur sudah benar (semua bahan tersedia saat digunakan)
                available := make(map[string]bool)
                for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
                    available[base] = true
                }
                
                valid := true
                for _, recipe := range finalPath {
                    // Cek bahan pertama tersedia
                    if !isBaseElementDFS(recipe.Ingredient1) && !available[recipe.Ingredient1] {
                        valid = false
                        break
                    }
                    
                    // Cek bahan kedua tersedia
                    if !isBaseElementDFS(recipe.Ingredient2) && !available[recipe.Ingredient2] {
                        valid = false
                        break
                    }
                    
                    // Tandai hasil resep sebagai tersedia
                    available[recipe.Result] = true
                }
                
                if !valid {
                    return // Jalur tidak valid, abaikan
                }
                
                // Cek keunikan jalur
                pathID := generatePathIdentifierDFS(finalPath)
                
                resultsMutex.Lock()
                defer resultsMutex.Unlock()
                
                if !uniquePathMap[pathID] && len(results) < maxPaths {
                    uniquePathMap[pathID] = true
                    results = append(results, finalPath)
                }
            }(recipe)
        }
        
        wg.Wait()
        
        return results
    }
    
    // ------------ Memulai pencarian jalur ------------
    
    // Tentukan elemen dasar yang tersedia
    availableElements := make(map[string]bool)
    for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
        availableElements[base] = true
    }
    
    // Cari jalur optimal
    fmt.Printf("Mencari jalur optimal untuk %s...\n", targetElement)
    optimalPath := buildOrderedPath(targetElement, availableElements, make(map[string]bool))
    
    if optimalPath == nil {
        return nil, nodesVisitedCount, fmt.Errorf("tidak ada jalur valid untuk membuat %s", targetElement)
    }
    
    // Hilangkan duplikat
    optimalPath = removeDuplicateRecipes(optimalPath)
    
    // Verifikasi jalur optimal
    available := make(map[string]bool)
    for _, base := range []string{"Air", "Earth", "Fire", "Water"} {
        available[base] = true
    }
    
    for i, recipe := range optimalPath {
        // Debug: cek prasyarat tersedia
        if !isBaseElementDFS(recipe.Ingredient1) && !available[recipe.Ingredient1] {
            fmt.Printf("PERINGATAN: Jalur optimal - bahan %s tidak tersedia pada langkah %d\n", 
                       recipe.Ingredient1, i+1)
        }
        
        if !isBaseElementDFS(recipe.Ingredient2) && !available[recipe.Ingredient2] {
            fmt.Printf("PERINGATAN: Jalur optimal - bahan %s tidak tersedia pada langkah %d\n", 
                       recipe.Ingredient2, i+1)
        }
        
        // Tandai hasil sebagai tersedia
        available[recipe.Result] = true
    }
    
    // Debug: tampilkan jalur optimal
    fmt.Printf("Jalur optimal (panjang: %d):\n", len(optimalPath))
    for i, recipe := range optimalPath {
        fmt.Printf("  Langkah %d: %s + %s => %s\n", 
                  i+1, recipe.Ingredient1, recipe.Ingredient2, recipe.Result)
    }
    
    // Jika hanya butuh 1 jalur, kembalikan sekarang
    if maxRecipes <= 1 {
        return [][]Recipe{optimalPath}, nodesVisitedCount, nil
    }
    
    // Cari jalur alternatif
    fmt.Printf("Mencari %d jalur alternatif...\n", maxRecipes-1)
    allPaths := findAlternativePaths(targetElement, optimalPath, maxRecipes)
    
    // Urutkan hasil berdasarkan panjang (pendek ke panjang)
    sort.Slice(allPaths, func(i, j int) bool {
        return len(allPaths[i]) < len(allPaths[j])
    })
    
    // Debug: tampilkan semua jalur
    for i, path := range allPaths {
        fmt.Printf("Jalur %d (panjang: %d):\n", i+1, len(path))
        for j, recipe := range path {
            fmt.Printf("  Langkah %d: %s + %s => %s\n", 
                      j+1, recipe.Ingredient1, recipe.Ingredient2, recipe.Result)
        }
    }
    
    return allPaths, nodesVisitedCount, nil
}

// Helper untuk menghapus resep duplikat dalam path
// func removeDuplicateRecipes(path []Recipe) []Recipe {
//     seen := make(map[string]bool)
//     unique := make([]Recipe, 0, len(path))
    
//     for _, recipe := range path {
//         key := fmt.Sprintf("%s:%s+%s", recipe.Result, recipe.Ingredient1, recipe.Ingredient2)
//         if !seen[key] {
//             seen[key] = true
//             unique = append(unique, recipe)
//         }
//     }
    
//     return unique
// }


// Fungsi isBaseElementDFS (pastikan ada dan bisa diakses)
func isBaseElementDFS(name string) bool {
	baseElements := []string{"Air", "Earth", "Fire", "Water"}
	for _, base := range baseElements {
		if name == base {
			return true
		}
	}
	return false
}

func generatePathIdentifierDFS(path []Recipe) string {
    recipesCopy := make([]Recipe, len(path))
    copy(recipesCopy, path)
    sort.Slice(recipesCopy, func(i, j int) bool {
        if recipesCopy[i].Result != recipesCopy[j].Result {
            return recipesCopy[i].Result < recipesCopy[j].Result
        }
        ing1i, ing2i := recipesCopy[i].Ingredient1, recipesCopy[i].Ingredient2
        if ing1i > ing2i { ing1i, ing2i = ing2i, ing1i }
        ing1j, ing2j := recipesCopy[j].Ingredient1, recipesCopy[j].Ingredient2
        if ing1j > ing2j { ing1j, ing2j = ing2j, ing1j }
        if ing1i != ing1j {
            return ing1i < ing1j
        }
        return ing2i < ing2j
    })
    var parts []string
    for _, r := range recipesCopy {
        ing1, ing2 := r.Ingredient1, r.Ingredient2
        if ing1 > ing2 { ing1, ing2 = ing2, ing1 }
        parts = append(parts, fmt.Sprintf("%s+%s=>%s", ing1, ing2, r.Result))
    }
    return strings.Join(parts, "|")
}
