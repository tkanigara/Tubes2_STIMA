// src/backend/bfs.go
package main

import (
	"container/list"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

var baseElements = []string{"Air", "Earth", "Fire", "Water"}

var baseElementMap = map[string]bool{
	"Air":   true,
	"Earth": true,
	"Fire":  true,
	"Water": true,
}

func isBaseElement(name string) bool {
	return baseElementMap[name]
}

var (
	bfsPathCacheMutex sync.RWMutex
)

func FindPathBFS(targetElement string) ([]Recipe, int, error) {
	fmt.Printf("Finding BFS shortest path to: %s\n", targetElement)
	graph := GetAlchemyGraph()
	if graph == nil {
		return nil, 0, errors.New("alchemy graph not initialized")
	}

	bfsPathCacheMutex.RLock()
	if path, exists := bfsPathCache[targetElement]; exists {
		bfsPathCacheMutex.RUnlock()
		fmt.Printf("BFS Cache: Path to '%s' found in cache.\n", targetElement)
		return path, 0, nil
	}
	bfsPathCacheMutex.RUnlock()

	if isBaseElement(targetElement) {
		return []Recipe{}, 0, nil
	}

	queue := list.New()
	visited := make(map[string]bool, 1000)
	elementVisited := make(map[string]bool, 1000)
	recipeParent := make(map[string]Recipe)
	discovered := make(map[string]bool, 1000)
	nodesVisitedCount := 0

	depth := make(map[string]int)

	sortedBaseElements := make([]string, len(baseElements))
	copy(sortedBaseElements, baseElements)
	sort.Strings(sortedBaseElements)

	for _, base := range sortedBaseElements {
		elementVisited[base] = true
		discovered[base] = true
		queue.PushBack(base)
		depth[base] = 0

		fmt.Printf("Enqueue base element: %s\n", base)
	}

	for queue.Len() > 0 {
		currentElement := queue.Remove(queue.Front()).(string)
		currentDepth := depth[currentElement]
		fmt.Printf("Dequeue: %s at depth %d\n", currentElement, currentDepth)
		nodesVisitedCount++

		combinableRecipes := graph[currentElement]
		if len(combinableRecipes) == 0 {
			continue
		}
		discoveredElementsList := make([]string, 0, len(discovered))
		for element := range discovered {
			discoveredElementsList = append(discoveredElementsList, element)
		}
		sort.Strings(discoveredElementsList)

		for _, otherElement := range discoveredElementsList {
			pairKey := getPairKey(currentElement, otherElement)
			if visited[pairKey] {
				continue
			}
			visited[pairKey] = true
			recipes := getRecipes(currentElement, otherElement)

			for _, recipe := range recipes {
				result := recipe.Result

				if !discovered[result] {
					discovered[result] = true
					recipeParent[result] = recipe
					depth[result] = currentDepth + 1
					if result == targetElement {
						fmt.Printf("Target '%s' found!\n", targetElement)
						path := buildRecipePath(recipeParent, targetElement, depth)
						bfsPathCacheMutex.Lock()
						bfsPathCache[targetElement] = path
						bfsPathCacheMutex.Unlock()

						return path, nodesVisitedCount, nil
					}
					if !elementVisited[result] {
						elementVisited[result] = true
						queue.PushBack(result)
						fmt.Printf("Enqueue: %s (from %s + %s) at depth %d\n",
							result, currentElement, otherElement, depth[result])
					}
				}
			}
		}
	}
	fmt.Printf("Target '%s' cannot be found.\n", targetElement)
	return nil, nodesVisitedCount, fmt.Errorf("path to element '%s' not found", targetElement)
}

func getPairKey(a, b string) string {
	if a > b {
		return b + ":" + a
	}
	return a + ":" + b
}

func buildRecipePath(recipeParent map[string]Recipe, target string, depth map[string]int) []Recipe {
	dependencies := make(map[string][]string)
	elementsNeeded := make(map[string]bool)

	queue := list.New()
	queue.PushBack(target)
	elementsNeeded[target] = true

	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(string)

		if isBaseElement(current) {
			continue
		}

		recipe, exists := recipeParent[current]
		if !exists {
			continue
		}

		ing1, ing2 := recipe.Ingredient1, recipe.Ingredient2
		dependencies[ing1] = append(dependencies[ing1], current)
		dependencies[ing2] = append(dependencies[ing2], current)

		ingredients := []string{ing1, ing2}
		sort.Strings(ingredients)

		for _, ingredient := range ingredients {
			if !elementsNeeded[ingredient] && !isBaseElement(ingredient) {
				elementsNeeded[ingredient] = true
				queue.PushBack(ingredient)
			}
		}
	}

	var result []Recipe
	available := make(map[string]bool)

	sortedBaseElements := make([]string, len(baseElements))
	copy(sortedBaseElements, baseElements)
	sort.Strings(sortedBaseElements)
	for _, base := range sortedBaseElements {
		available[base] = true
	}

	remainingElements := len(elementsNeeded)
	for remainingElements > 0 {
		candidateElements := make([]string, 0, len(elementsNeeded))
		for element := range elementsNeeded {
			if !available[element] {
				recipe, exists := recipeParent[element]
				if !exists {
					continue
				}

				if available[recipe.Ingredient1] && available[recipe.Ingredient2] {
					candidateElements = append(candidateElements, element)
				}
			}
		}

		if len(candidateElements) == 0 {
			break
		}

		sort.SliceStable(candidateElements, func(i, j int) bool {
			if depth[candidateElements[i]] != depth[candidateElements[j]] {
				return depth[candidateElements[i]] < depth[candidateElements[j]]
			}

			depCountI := len(dependencies[candidateElements[i]])
			depCountJ := len(dependencies[candidateElements[j]])
			if depCountI != depCountJ {
				return depCountI > depCountJ
			}

			return candidateElements[i] < candidateElements[j]
		})

		bestElement := candidateElements[0]

		recipe := recipeParent[bestElement]
		result = append(result, recipe)
		available[bestElement] = true
		delete(elementsNeeded, bestElement)
		remainingElements--

		if available[target] {
			break
		}
	}

	return result
}

func getRecipes(a, b string) []Recipe {
	graph := GetAlchemyGraph()
	var result []Recipe

	aRecipes := graph[a]
	bRecipes := graph[b]

	sourceRecipes := aRecipes
	if len(bRecipes) < len(aRecipes) {
		sourceRecipes = bRecipes
	}

	for _, r := range sourceRecipes {
		if (r.Ingredient1 == a && r.Ingredient2 == b) || (r.Ingredient1 == b && r.Ingredient2 == a) {
			result = append(result, r)
		}
	}

	if len(result) > 1 {
		sort.SliceStable(result, func(i, j int) bool {
			return result[i].Result < result[j].Result
		})
	}
	return result
}

func generatePathIdentifier(path []Recipe) string {
	if len(path) == 0 {
		return ""
	}

	resultToIngredients := make(map[string]string)

	for _, r := range path {
		ing1, ing2 := r.Ingredient1, r.Ingredient2
		if ing1 > ing2 {
			ing1, ing2 = ing2, ing1
		}
		resultToIngredients[r.Result] = ing1 + "+" + ing2
	}

	parts := make([]string, 0, len(resultToIngredients))
	for result, ingredients := range resultToIngredients {
		parts = append(parts, fmt.Sprintf("%s=>%s", ingredients, result))
	}

	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func getUniqueRecipeKey(recipe Recipe) string {
	ing1, ing2 := recipe.Ingredient1, recipe.Ingredient2
	if ing1 > ing2 {
		ing1, ing2 = ing2, ing1
	}
	return fmt.Sprintf("%s+%s=>%s", ing1, ing2, recipe.Result)
}

func FindMultiplePathsBFS(targetElement string, maxRecipes int) ([][]Recipe, int, error) {
	fmt.Printf("Finding %d different BFS paths to: %s (Multithreaded)\n", maxRecipes, targetElement)

	graph := GetAlchemyGraph()
	if graph == nil {
		return nil, 0, errors.New("alchemy graph not initialized")
	}
	if maxRecipes <= 0 {
		return nil, 0, errors.New("minimum number of recipes must be 1")
	}
	if isBaseElement(targetElement) {
		return [][]Recipe{}, 0, nil
	}

	uniqueRecipeCombos, allCombinations := getAllUniqueRecipeCombinations(targetElement)
	if uniqueRecipeCombos == 0 {
		return nil, 0, fmt.Errorf("element '%s' not found in recipe database", targetElement)
	}

	fmt.Printf("Element '%s' can be created from %d unique ingredient combinations:\n",
		targetElement, uniqueRecipeCombos)
	for comboKey, recipe := range allCombinations {
		fmt.Printf("  - %s + %s => %s (key: %s)\n",
			recipe.Ingredient1, recipe.Ingredient2, recipe.Result, comboKey)
	}

	if uniqueRecipeCombos < maxRecipes {
		fmt.Printf("Adjusting max paths to %d to match available combinations\n", uniqueRecipeCombos)
		maxRecipes = uniqueRecipeCombos
	}

	if maxRecipes == 1 {
		firstPath, visitCount, err := FindPathBFS(targetElement)
		if err != nil {
			return nil, visitCount, err
		}
		return [][]Recipe{firstPath}, visitCount, nil
	}

	foundTargetCombinations := make(map[string]bool)
	remainingCombinations := make(map[string]Recipe, uniqueRecipeCombos)
	for comboKey, recipe := range allCombinations {
		remainingCombinations[comboKey] = recipe
	}

	var allFoundPaths [][]Recipe
	addedPathIdentifiers := make(map[string]bool)
	var mu sync.Mutex
	nodesVisitedCount := atomic.Int32{}

	var wg sync.WaitGroup
	pathChan := make(chan []Recipe, maxRecipes)
	done := atomic.Bool{}

	firstPath, _, firstErr := FindPathBFS(targetElement)
	if firstErr == nil && len(firstPath) > 0 {
		pathID := generatePathIdentifier(firstPath)

		var targetRecipe Recipe
		for _, r := range firstPath {
			if r.Result == targetElement {
				targetRecipe = r
				break
			}
		}

		comboKey := getUniqueRecipeKey(targetRecipe)

		mu.Lock()
		allFoundPaths = append(allFoundPaths, firstPath)
		addedPathIdentifiers[pathID] = true
		foundTargetCombinations[comboKey] = true
		delete(remainingCombinations, comboKey)
		mu.Unlock()

		fmt.Printf("Initial path found for %s via FindPathBFS using ingredients: %s + %s\n",
			targetElement, targetRecipe.Ingredient1, targetRecipe.Ingredient2)

		select {
		case pathChan <- firstPath:
		default:
		}
	}

	shouldStop := func() bool {
		mu.Lock()
		isDone := len(allFoundPaths) >= maxRecipes || len(foundTargetCombinations) >= uniqueRecipeCombos
		mu.Unlock()
		return isDone || done.Load()
	}

	if len(foundTargetCombinations) < uniqueRecipeCombos && len(allFoundPaths) < maxRecipes {
		numWorkersPerCombo := 3

		mu.Lock()
		combinationsToSearch := make([]Recipe, 0, len(remainingCombinations))
		for _, recipe := range remainingCombinations {
			combinationsToSearch = append(combinationsToSearch, recipe)
		}
		mu.Unlock()

		for comboIdx, targetRecipe := range combinationsToSearch {
			for w := 0; w < numWorkersPerCombo; w++ {
				if shouldStop() {
					break
				}

				wg.Add(1)
				go func(workerID int, comboIdx int, targetComboRecipe Recipe) {
					defer wg.Done()

					comboKey := getUniqueRecipeKey(targetComboRecipe)

					mu.Lock()
					alreadyFound := foundTargetCombinations[comboKey]
					mu.Unlock()

					if alreadyFound {
						return
					}

					strategyVariant := (workerID + comboIdx) % 5

					fmt.Printf("Worker %d searching for combo %d: %s + %s => %s (strategy: %d)\n",
						workerID, comboIdx,
						targetComboRecipe.Ingredient1, targetComboRecipe.Ingredient2,
						targetComboRecipe.Result, strategyVariant)

					currentPath := findPathForSpecificCombination(
						targetElement,
						targetComboRecipe,
						strategyVariant,
						&nodesVisitedCount,
						shouldStop,
					)

					if len(currentPath) > 0 {
						var foundTargetRecipe Recipe
						for _, r := range currentPath {
							if r.Result == targetElement {
								foundTargetRecipe = r
								break
							}
						}

						pathComboKey := getUniqueRecipeKey(foundTargetRecipe)
						if pathComboKey != comboKey {
							fmt.Printf("Warning: Worker %d found wrong combination %s instead of %s\n",
								workerID, pathComboKey, comboKey)
							return
						}

						pathID := generatePathIdentifier(currentPath)

						mu.Lock()
						isNewCombo := !foundTargetCombinations[pathComboKey]
						isNewPath := !addedPathIdentifiers[pathID]

						if isNewCombo && isNewPath && len(allFoundPaths) < maxRecipes {
							pathCopy := make([]Recipe, len(currentPath))
							copy(pathCopy, currentPath)

							allFoundPaths = append(allFoundPaths, pathCopy)
							addedPathIdentifiers[pathID] = true
							foundTargetCombinations[pathComboKey] = true
							delete(remainingCombinations, pathComboKey)

							fmt.Printf("Worker %d: Found path #%d for %s using ingredients: %s + %s (strategy: %d)\n",
								workerID, len(allFoundPaths), targetElement,
								foundTargetRecipe.Ingredient1, foundTargetRecipe.Ingredient2,
								strategyVariant)

							select {
							case pathChan <- pathCopy:
							default:
							}

							if len(allFoundPaths) >= maxRecipes || len(foundTargetCombinations) >= uniqueRecipeCombos {
								done.Store(true)
							}
						}
						mu.Unlock()
					}
				}(w, comboIdx, targetRecipe)
			}
		}

		if !shouldStop() {
			additionalWorkers := runtime.NumCPU() * 2

			for w := 0; w < additionalWorkers; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					strategyVariant := workerID % 5
					queue := list.New()
					localVisited := make(map[string]bool)
					parent := make(map[string]Recipe)
					discovered := make(map[string]bool)

					startOffset := (workerID * 17) % len(baseElements)
					for i := 0; i < len(baseElements); i++ {
						idx := (startOffset + i) % len(baseElements)
						base := baseElements[idx]
						queue.PushBack(base)
						localVisited[base] = true
						discovered[base] = true
					}

					depthMap := make(map[string]int)
					for _, base := range baseElements {
						depthMap[base] = 0
					}

					for queue.Len() > 0 && !shouldStop() {
						currentElement := queue.Remove(queue.Front()).(string)
						currentDepth := depthMap[currentElement]
						nodesVisitedCount.Add(1)

						if nodesVisitedCount.Load()%1000 == 0 {
							mu.Lock()
							if len(remainingCombinations) > 0 && len(remainingCombinations) <= 3 {
								var targetCombo Recipe
								for _, recipe := range remainingCombinations {
									targetCombo = recipe
									break
								}
								mu.Unlock()

								ing1 := targetCombo.Ingredient1
								ing2 := targetCombo.Ingredient2

								if !discovered[ing1] {
									queue.PushFront(ing1)
								}
								if !discovered[ing2] {
									queue.PushFront(ing2)
								}
							} else {
								mu.Unlock()
							}
						}

						combinableElements := make([]string, 0, len(discovered))
						for elem := range discovered {
							combinableElements = append(combinableElements, elem)
						}

						sortElements(combinableElements, strategyVariant, depthMap, workerID)

						for _, otherElement := range combinableElements {
							pairKey := getPairKey(currentElement, otherElement)

							if localVisited[pairKey] {
								continue
							}
							localVisited[pairKey] = true

							recipes := getRecipes(currentElement, otherElement)

							for _, recipe := range recipes {
								if shouldStop() {
									return
								}

								result := recipe.Result
								resultDepth := currentDepth + 1

								_, alreadyFound := parent[result]

								shouldOverride := false
								if alreadyFound {
									rnd := (int(nodesVisitedCount.Load()) + workerID + int(resultDepth)) % 100
									shouldOverride = rnd < 15 // 15% chance
								}

								if !alreadyFound || shouldOverride {
									parent[result] = recipe
									depthMap[result] = resultDepth
								}

								wasNewDiscovery := !discovered[result]
								discovered[result] = true

								queueIt := wasNewDiscovery
								if wasNewDiscovery || shouldOverride {
									if !localVisited[result] || shouldOverride {
										localVisited[result] = true
										if queueIt {
											queue.PushBack(result)
										}
									}
								}

								if result == targetElement {
									comboKey := getUniqueRecipeKey(recipe)

									mu.Lock()
									alreadyFoundThisCombo := foundTargetCombinations[comboKey]
									mu.Unlock()

									if alreadyFoundThisCombo {
										continue
									}

									currentPath := buildDiversePath(parent, targetElement, workerID)
									if len(currentPath) > 0 {
										var pathTargetRecipe Recipe
										for _, r := range currentPath {
											if r.Result == targetElement {
												pathTargetRecipe = r
												break
											}
										}

										pathComboKey := getUniqueRecipeKey(pathTargetRecipe)
										pathID := generatePathIdentifier(currentPath)

										mu.Lock()
										isNewCombo := !foundTargetCombinations[pathComboKey]
										isNewPath := !addedPathIdentifiers[pathID]

										if isNewCombo && isNewPath && len(allFoundPaths) < maxRecipes {
											pathCopy := make([]Recipe, len(currentPath))
											copy(pathCopy, currentPath)

											allFoundPaths = append(allFoundPaths, pathCopy)
											addedPathIdentifiers[pathID] = true
											foundTargetCombinations[pathComboKey] = true
											delete(remainingCombinations, pathComboKey)

											fmt.Printf("Worker %d: Found path #%d for %s using ingredients: %s + %s (strategy: %d)\n",
												workerID, len(allFoundPaths), targetElement,
												pathTargetRecipe.Ingredient1, pathTargetRecipe.Ingredient2,
												strategyVariant)
											select {
											case pathChan <- pathCopy:
											default:
											}

											if len(allFoundPaths) >= maxRecipes || len(foundTargetCombinations) >= uniqueRecipeCombos {
												done.Store(true)
											}
										}
										mu.Unlock()
									}
								}
							}
						}
					}
				}(w)
			}
		}
	}

	go func() {
		wg.Wait()
		close(pathChan)
	}()

	for range pathChan {
		// Paths are already saved to allFoundPaths
	}

	mu.Lock()
	result := make([][]Recipe, len(allFoundPaths))
	copy(result, allFoundPaths)

	missingCount := len(remainingCombinations)
	if missingCount > 0 {
		fmt.Printf("Warning: %d combinations were never found:\n", missingCount)
		for comboKey, recipe := range remainingCombinations {
			fmt.Printf("  - Missing: %s + %s => %s (key: %s)\n",
				recipe.Ingredient1, recipe.Ingredient2, recipe.Result, comboKey)
		}
	}

	foundCount := len(result)
	foundCombinations := len(foundTargetCombinations)
	mu.Unlock()

	if foundCount == 0 && !isBaseElement(targetElement) {
		fmt.Printf("BFS Multiple: No paths found for '%s'.\n", targetElement)
		return nil, int(nodesVisitedCount.Load()), fmt.Errorf("path to element '%s' not found", targetElement)
	}

	fmt.Printf("BFS Multiple: Found %d unique paths (using %d/%d unique ingredient combinations) for '%s' (requested %d).\n",
		foundCount, foundCombinations, uniqueRecipeCombos, targetElement, maxRecipes)
	return result, int(nodesVisitedCount.Load()), nil
}

func getAllUniqueRecipeCombinations(element string) (int, map[string]Recipe) {
	uniqueCombos := make(map[string]Recipe)

	if isBaseElement(element) {
		return 0, uniqueCombos
	}

	graph := GetAlchemyGraph()
	if graph == nil {
		return 0, uniqueCombos
	}

	for _, recipes := range graph {
		for _, recipe := range recipes {
			if recipe.Result == element {
				comboKey := getUniqueRecipeKey(recipe)
				uniqueCombos[comboKey] = recipe
			}
		}
	}

	return len(uniqueCombos), uniqueCombos
}

func findPathForSpecificCombination(targetElement string, targetRecipe Recipe,
	strategyVariant int, nodesVisitedCount *atomic.Int32, shouldStop func() bool) []Recipe {

	ing1 := targetRecipe.Ingredient1
	ing2 := targetRecipe.Ingredient2

	queue := list.New()
	localVisited := make(map[string]bool)
	parent := make(map[string]Recipe)
	discovered := make(map[string]bool)
	depthMap := make(map[string]int)

	for _, base := range baseElements {
		queue.PushBack(base)
		localVisited[base] = true
		discovered[base] = true
		depthMap[base] = 0
	}

	for queue.Len() > 0 && !shouldStop() {
		currentElement := queue.Remove(queue.Front()).(string)
		currentDepth := depthMap[currentElement]
		nodesVisitedCount.Add(1)

		if discovered[ing1] && discovered[ing2] {
			if !discovered[targetElement] {
				parent[targetElement] = targetRecipe
				depthMap[targetElement] = max(depthMap[ing1], depthMap[ing2]) + 1
				discovered[targetElement] = true

				return buildDiversePath(parent, targetElement, strategyVariant)
			}
		}

		combinableElements := make([]string, 0, len(discovered))
		for elem := range discovered {
			combinableElements = append(combinableElements, elem)
		}

		sortElements(combinableElements, strategyVariant, depthMap, int(nodesVisitedCount.Load()))

		for _, otherElement := range combinableElements {
			pairKey := getPairKey(currentElement, otherElement)

			if localVisited[pairKey] {
				continue
			}
			localVisited[pairKey] = true

			recipes := getRecipes(currentElement, otherElement)

			for _, recipe := range recipes {
				if shouldStop() {
					return []Recipe{}
				}

				result := recipe.Result
				resultDepth := currentDepth + 1

				_, alreadyFound := parent[result]

				isPriorityElement := result == ing1 || result == ing2

				if !alreadyFound || isPriorityElement {
					parent[result] = recipe
					depthMap[result] = resultDepth
				}

				wasNewDiscovery := !discovered[result]
				discovered[result] = true

				if wasNewDiscovery || isPriorityElement {
					if !localVisited[result] || isPriorityElement {
						localVisited[result] = true

						if isPriorityElement {
							queue.PushFront(result)
						} else {
							queue.PushBack(result)
						}
					}
				}
			}
		}
	}

	return []Recipe{}
}

func sortElements(elements []string, strategyVariant int, depthMap map[string]int, seed int) {
	switch strategyVariant {
	case 0:
		// Alphabetical
		sort.Strings(elements)
	case 1:
		// Reverse alphabetical
		sort.Slice(elements, func(i, j int) bool {
			return elements[i] > elements[j]
		})
	case 2:
		// By depth (shallow first)
		sort.Slice(elements, func(i, j int) bool {
			depthI := depthMap[elements[i]]
			depthJ := depthMap[elements[j]]
			if depthI != depthJ {
				return depthI < depthJ
			}
			return elements[i] < elements[j]
		})
	case 3:
		// By depth (deep first)
		sort.Slice(elements, func(i, j int) bool {
			depthI := depthMap[elements[i]]
			depthJ := depthMap[elements[j]]
			if depthI != depthJ {
				return depthI > depthJ
			}
			return elements[i] < elements[j]
		})
	case 4:
		// Pseudo-random but deterministic ordering
		sort.Slice(elements, func(i, j int) bool {
			hashI := (seed*31 + len(elements[i])*43 + int(elements[i][0])) % 100
			hashJ := (seed*31 + len(elements[j])*43 + int(elements[j][0])) % 100
			if hashI != hashJ {
				return hashI < hashJ
			}
			return elements[i] < elements[j]
		})
	}
}

func buildDiversePath(parent map[string]Recipe, target string, workerID int) []Recipe {
	elementsNeeded := make(map[string]bool)
	queue := list.New()
	queue.PushBack(target)
	processed := make(map[string]bool)
	processed[target] = true
	elementsNeeded[target] = true

	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(string)

		if isBaseElement(current) {
			continue
		}

		recipe, exists := parent[current]
		if !exists {
			return []Recipe{}
		}

		for _, ingredient := range []string{recipe.Ingredient1, recipe.Ingredient2} {
			if processed[ingredient] || isBaseElement(ingredient) {
				continue
			}

			processed[ingredient] = true
			elementsNeeded[ingredient] = true
			queue.PushBack(ingredient)
		}
	}

	var result []Recipe
	available := make(map[string]bool)

	for _, base := range baseElements {
		available[base] = true
	}

	for !available[target] {
		candidates := make([]Recipe, 0)

		for element := range elementsNeeded {
			if available[element] {
				continue
			}

			recipe, exists := parent[element]
			if !exists {
				continue
			}

			if available[recipe.Ingredient1] && available[recipe.Ingredient2] {
				candidates = append(candidates, recipe)
			}
		}

		if len(candidates) == 0 {
			return []Recipe{}
		}

		strategyVariant := workerID % 3
		sort.SliceStable(candidates, func(i, j int) bool {
			switch strategyVariant {
			case 0:
				return candidates[i].Result < candidates[j].Result
			case 1:
				return candidates[i].Result > candidates[j].Result
			default:
				ing1i := candidates[i].Ingredient1 + candidates[i].Ingredient2
				ing1j := candidates[j].Ingredient1 + candidates[j].Ingredient2
				return ing1i < ing1j
			}
		})

		recipe := candidates[0]
		result = append(result, recipe)
		available[recipe.Result] = true

		if available[target] {
			break
		}
	}

	return result
}

func ResetCaches() {
	bfsPathCacheMutex.Lock()
	bfsPathCache = make(map[string][]Recipe)
	bfsPathCacheMutex.Unlock()
}
