import React, { useState, useEffect, useCallback } from 'react';
import Tree from 'react-d3-tree';
import './SearchResults.css'; // Pastikan file CSS ini ada dan sesuai
import notFoundImage from '../assets/notfound_dontol.jpg'; // Pastikan path gambar ini benar

const API_BASE_URL = "http://localhost:8080"; // Sesuaikan jika perlu
const LIVE_UPDATE_DELAY_MS = 800; // Kecepatan animasi live update

// Helper function to check if an element is a base element
const isBaseElement = (name) => {
    const baseElements = ["Air", "Earth", "Fire", "Water"]; // Daftar elemen dasar
    return baseElements.includes(name);
};

// Helper function to build the initial node structure for an element
const buildInitialElementNode = (elementName, imageURLs, depth = 0) => {
    const imageUrlPath = imageURLs && typeof imageURLs === 'object' ? (imageURLs[elementName] || '') : '';
    return {
      name: elementName || "Unknown",
      attributes: {
        type: isBaseElement(elementName) ? 'Base Element' : 'Element',
        imageUrl: imageUrlPath,
        depth: depth,
        originalName: elementName || "Unknown",
      },
      children: [],
    };
};

// Recursively builds the full static tree data from a path map
const buildElementNodeRecursive = (elementName, pathRecipesMap, imageURLs, currentDepth = 0, maxDepth = 20) => {
    // Batasi kedalaman rekursi untuk mencegah infinite loop atau pohon yang terlalu besar
    if (currentDepth > maxDepth) {
        return {
            name: `${elementName || "Unknown"} (Batas Kedalaman)`,
            attributes: { type: 'Error', originalName: elementName || "Unknown", depth: currentDepth, info: 'Max depth reached' },
            children: []
        };
    }

    // Buat node awal untuk elemen saat ini
    const node = buildInitialElementNode(elementName, imageURLs, currentDepth);
    const recipeMakingThis = pathRecipesMap && typeof pathRecipesMap === 'object' ? pathRecipesMap[elementName] : null;

    // Jika ada resep yang menghasilkan elemen ini, tambahkan node resep dan rekursif ke bawah
    if (recipeMakingThis && typeof recipeMakingThis === 'object' && recipeMakingThis.ingredient1 && recipeMakingThis.ingredient2) {
        const recipeNode = {
            name: `${recipeMakingThis.ingredient1} + ${recipeMakingThis.ingredient2}`, // Nama node resep
            attributes: {
                type: 'Recipe', // Tandai sebagai node resep
                result: recipeMakingThis.result,
                ingredient1: recipeMakingThis.ingredient1,
                ingredient2: recipeMakingThis.ingredient2,
                depth: currentDepth + 1,
                originalName: `${recipeMakingThis.ingredient1} + ${recipeMakingThis.ingredient2}`
            },
            children: []
        };
        // Rekursif untuk kedua ingredient
        recipeNode.children.push(buildElementNodeRecursive(recipeMakingThis.ingredient1, pathRecipesMap, imageURLs, currentDepth + 2, maxDepth));
        recipeNode.children.push(buildElementNodeRecursive(recipeMakingThis.ingredient2, pathRecipesMap, imageURLs, currentDepth + 2, maxDepth));
        // Tambahkan node resep sebagai anak dari node elemen
        node.children.push(recipeNode);
    }
    return node;
};

// Builds the complete static tree data structure for a given path
const buildFullTreeData = (path, targetElement, imageURLs) => {
  // Jika elemen dasar atau path tidak valid, hanya tampilkan node elemen dasar
  if (isBaseElement(targetElement) || !Array.isArray(path) || path.length === 0) {
    return [buildInitialElementNode(targetElement, imageURLs)];
  }

  // Buat map resep untuk pencarian cepat
  const pathRecipesMap = {};
  path.forEach(recipe => {
    if (recipe && typeof recipe === 'object' && recipe.result) {
        pathRecipesMap[recipe.result] = recipe;
    }
  });

  // Mulai bangun pohon secara rekursif dari elemen target
  const rootNode = buildElementNodeRecursive(targetElement, pathRecipesMap, imageURLs);
  return rootNode ? [rootNode] : [buildInitialElementNode(targetElement, imageURLs)]; // Pastikan selalu mengembalikan array
};

// Custom node rendering function for react-d3-tree
const renderNodeWithImage = ({ nodeDatum, toggleNode }) => {
    const isRecipeNode = nodeDatum?.attributes?.type === 'Recipe';
    const originalElementName = nodeDatum?.attributes?.originalName || nodeDatum?.name || "N/A";
    const displayName = isRecipeNode ? `+` : originalElementName; // Tampilkan '+' untuk node resep
    const partialImageUrl = nodeDatum?.attributes?.imageUrl;
    const fullImageUrl = partialImageUrl ? `${API_BASE_URL}${partialImageUrl}` : '';
    const imageSize = isRecipeNode ? 20 : (isBaseElement(originalElementName) ? 40 : 35); // Ukuran gambar berbeda
    const textYOffset = isRecipeNode ? 0 : imageSize / 2 + 8; // Posisi teks

    // Kelas CSS berbeda untuk styling
    let nodeClass = "node-element";
    if (isRecipeNode) nodeClass = "node-recipe";
    else if (nodeDatum?.attributes?.type === 'Base Element') nodeClass = "node-base-element";
    else if (originalElementName.includes("Batas Kedalaman") || nodeDatum?.attributes?.type === 'Error') nodeClass = "node-error";

    return (
      <g onClick={toggleNode} className={`tree-node ${nodeClass}`}>
        {/* Lingkaran latar belakang */}
        <circle r={isRecipeNode ? 12 : imageSize / 2 + 3} className="node-circle-bg" />
        {/* Tampilkan gambar jika bukan node resep dan URL valid */}
        {!isRecipeNode && fullImageUrl && (
            <image
              x={-imageSize / 2} y={-imageSize / 2}
              width={imageSize} height={imageSize}
              href={fullImageUrl} className="element-image"
              onError={(e) => { e.target.style.display = 'none'; }} // Sembunyikan jika gambar gagal dimuat
            />
        )}
        {/* Teks nama elemen atau '+' */}
        <text strokeWidth={isRecipeNode ? "0.4" : "0.5"} x={0} y={textYOffset}
          textAnchor="middle" alignmentBaseline={isRecipeNode ? "middle" : "hanging"}
          className="node-text">
          {displayName}
        </text>
      </g>
    );
};

// Builds the *next* state of the tree during live update by applying one recipe step
// NOTE: This function is designed to expand only *one* matching node per call.
// The logic to handle multiple expansions per recipe step is now in the useEffect hook.
const buildTreeForNextStep = (currentTreeDataArray, nextRecipeToProcess, allImageURLs) => {
    // Kasus awal: jika pohon belum ada, buat pohon dasar untuk resep pertama
    if (!Array.isArray(currentTreeDataArray) || currentTreeDataArray.length === 0 || !nextRecipeToProcess || typeof nextRecipeToProcess !== 'object') {
        // Ini seharusnya tidak terjadi jika live update dimulai dengan benar, tapi sebagai fallback
        const rootNode = buildInitialElementNode(nextRecipeToProcess.result, allImageURLs, 0);
        const recipeNode = {
            name: `${nextRecipeToProcess.ingredient1} + ${nextRecipeToProcess.ingredient2}`,
            attributes: { type: 'Recipe', result: nextRecipeToProcess.result, ingredient1: nextRecipeToProcess.ingredient1, ingredient2: nextRecipeToProcess.ingredient2, depth: 1, originalName: `${nextRecipeToProcess.ingredient1} + ${nextRecipeToProcess.ingredient2}` },
            children: [
                buildInitialElementNode(nextRecipeToProcess.ingredient1, allImageURLs, 2),
                buildInitialElementNode(nextRecipeToProcess.ingredient2, allImageURLs, 2)
            ]
        };
        rootNode.children = [recipeNode];
        return [rootNode];
    }

    // Buat salinan mendalam dari pohon saat ini untuk dimodifikasi (immutability)
    const newTreeData = JSON.parse(JSON.stringify(currentTreeDataArray));
    const rootNode = newTreeData[0]; // Asumsi selalu ada satu root node
    let expanded = false; // Flag untuk memastikan hanya satu node yang diperluas per panggilan

    // Helper untuk deep clone node (meskipun JSON.stringify sudah melakukan ini)
    const deepCloneNode = (node) => {
        if (!node) return null;
        return JSON.parse(JSON.stringify(node));
    };

    // Helper untuk mencari node berdasarkan nama (mungkin tidak efisien untuk pohon besar)
    // Ini digunakan untuk mencoba menggunakan kembali struktur node ingredient jika sudah ada di pohon
    const findNodeByName = (startNode, targetName) => {
        if (!startNode) return null;
        const queue = [startNode];
        while(queue.length > 0) {
            const currentNode = queue.shift();
            if (currentNode.name === targetName && currentNode.attributes?.type !== 'Recipe') {
                return currentNode; // Ditemukan
            }
            if (currentNode.children) {
                queue.push(...currentNode.children);
            }
        }
        return null; // Tidak ditemukan
    };

    // Fungsi rekursif untuk mencari dan memperluas node yang cocok dengan hasil resep
    const findAndExpandNode = (node) => {
        if (!node || expanded || !node.attributes) return false; // Berhenti jika sudah expand atau node tidak valid

        // Cek apakah node ini adalah target *elemen* (bukan resep) yang cocok dengan hasil resep
        if (node.name === nextRecipeToProcess.result && node.attributes.type !== 'Recipe') {
            // Cek apakah node ini *belum* memiliki anak resep (sudah diperluas sebelumnya di langkah ini)
            const alreadyHasThisRecipeAsChild = node.children?.some(child =>
                child.attributes?.type === 'Recipe' &&
                child.attributes?.ingredient1 === nextRecipeToProcess.ingredient1 &&
                child.attributes?.ingredient2 === nextRecipeToProcess.ingredient2);

            if (!alreadyHasThisRecipeAsChild) {
                // Simpan anak-anak yang sudah ada (jika ada, meskipun seharusnya tidak ada sebelum diperluas)
                const existingChildren = node.children || [];

                // Buat node resep baru
                const recipeNode = {
                    name: `${nextRecipeToProcess.ingredient1} + ${nextRecipeToProcess.ingredient2}`,
                    attributes: {
                        type: 'Recipe',
                        result: nextRecipeToProcess.result,
                        ingredient1: nextRecipeToProcess.ingredient1,
                        ingredient2: nextRecipeToProcess.ingredient2,
                        depth: (node.attributes.depth || 0) + 1,
                        originalName: `${nextRecipeToProcess.ingredient1} + ${nextRecipeToProcess.ingredient2}`
                    },
                    children: []
                };

                // Coba cari node penuh untuk ingredient (jika sudah muncul di pohon)
                // Ini opsional, bisa juga selalu buat node baru
                const ingredient1FullNode = findNodeByName(rootNode, nextRecipeToProcess.ingredient1);
                const ingredient2FullNode = findNodeByName(rootNode, nextRecipeToProcess.ingredient2);

                // Tambahkan anak ingredient ke node resep
                // Gunakan node yang ditemukan (clone) atau buat node awal baru
                recipeNode.children.push(
                    ingredient1FullNode && !isBaseElement(ingredient1FullNode.name) // Jangan clone jika base element, buat baru saja
                    ? deepCloneNode(ingredient1FullNode) // Clone struktur yang ada
                    : buildInitialElementNode(nextRecipeToProcess.ingredient1, allImageURLs, (node.attributes.depth || 0) + 2)
                );

                recipeNode.children.push(
                    ingredient2FullNode && !isBaseElement(ingredient2FullNode.name)
                    ? deepCloneNode(ingredient2FullNode)
                    : buildInitialElementNode(nextRecipeToProcess.ingredient2, allImageURLs, (node.attributes.depth || 0) + 2)
                );

                // Ganti anak node elemen target dengan node resep baru
                // (dan anak-anak yang sudah ada sebelumnya, jika ada)
                node.children = [recipeNode, ...existingChildren];
                expanded = true; // Tandai bahwa ekspansi telah terjadi di panggilan ini
                return true; // Berhasil memperluas
            }
        }

        // Jika node saat ini tidak cocok atau sudah punya anak resep, cari di anak-anaknya
        if (node.children && !expanded) { // Hanya lanjut jika belum ada yang diperluas
            for (let child of node.children) {
                const result = findAndExpandNode(child);
                if (result) {
                    // expanded sudah di set true di dalam rekursi
                    return true; // Hentikan pencarian lebih lanjut
                }
            }
        }

        return false; // Tidak ditemukan node untuk diperluas di cabang ini
    };

    // Mulai pencarian dan ekspansi dari root
    findAndExpandNode(rootNode);
    return newTreeData; // Kembalikan pohon yang (mungkin) sudah dimodifikasi
};

// --- Komponen Utama SearchResults ---
function SearchResults({ results, isLoading, error }) {
  // State untuk melacak status live update per pathKey
  const [liveUpdateStates, setLiveUpdateStates] = useState({});
  // State untuk menyimpan data pohon statis (hasil akhir) per pathKey
  const [treeDataForStaticView, setTreeDataForStaticView] = useState({});

  // Efek untuk membangun data pohon statis saat hasil pencarian berubah
  useEffect(() => {
    try {
        const newStaticData = {};
        if (results && results.pathFound) {
            const imageURLs = results.imageURLs || {}; // Ambil URL gambar
            // Mode shortest path
            if (results.mode === 'shortest' && Array.isArray(results.path)) {
                const pathKey = `path-block-shortest-0`;
                newStaticData[pathKey] = buildFullTreeData(results.path, results.searchTarget, imageURLs);
            // Mode multiple paths
            } else if (results.mode === 'multiple' && Array.isArray(results.paths)) {
                results.paths.forEach((p, index) => {
                    if (Array.isArray(p)) { // Pastikan path valid
                        const pathKey = `path-block-multiple-${index}`;
                        newStaticData[pathKey] = buildFullTreeData(p, results.searchTarget, imageURLs);
                    }
                });
            // Kasus elemen dasar (tidak ada path)
            } else if (isBaseElement(results.searchTarget)) {
                const pathKey = `path-block-${results.searchTarget}-base`;
                newStaticData[pathKey] = buildFullTreeData([], results.searchTarget, imageURLs);
            }
        }
        setTreeDataForStaticView(newStaticData); // Update state pohon statis
    } catch (e) {
        console.error("Error saat membangun treeDataForStaticView:", e);
        setTreeDataForStaticView({}); // Reset jika error
    }
  }, [results]); // Jalankan ulang saat `results` berubah

  // Fungsi untuk memulai atau me-restart live update untuk path tertentu
  const startLiveUpdate = useCallback((pathKey, originalPathData, targetElement, imageURLsInput) => {
    const imageURLs = imageURLsInput || {};
    // Jangan mulai live update jika elemen dasar atau path tidak valid
    if (isBaseElement(targetElement) || !Array.isArray(originalPathData) || originalPathData.length === 0) {
      // Set state untuk menampilkan pohon statis saja
      setLiveUpdateStates(prev => ({
          ...prev,
          [pathKey]: {
              isActive: true, // Tandai aktif untuk UI
              currentDisplayData: [buildInitialElementNode(targetElement, imageURLs)], // Tampilkan node awal
              fullPathRecipes: [],
              currentRecipeStep: 0,
              isBuilding: false, // Tidak sedang membangun
              isComplete: true, // Langsung selesai
              pathIdentifier: `live-${pathKey}-${new Date().getTime()}` // Key unik
          }
      }));
      return;
    }

    // Inisialisasi state live update
    const rootNode = buildInitialElementNode(targetElement, imageURLs); // Pohon awal hanya root
    setLiveUpdateStates(prev => ({
        ...prev,
        [pathKey]: {
            isActive: true,
            currentDisplayData: [rootNode], // Mulai dengan root node
            fullPathRecipes: [...originalPathData].reverse(), // Balik resep untuk proses dari dasar
            currentRecipeStep: 0, // Mulai dari resep pertama (setelah dibalik)
            isBuilding: true, // Mulai proses building
            isComplete: false, // Belum selesai
            pathIdentifier: `live-${pathKey}-${new Date().getTime()}` // Key unik baru untuk re-render
        }
    }));
  }, []); // useCallback dependencies kosong karena tidak bergantung state/props luar

  // --- Helper Function untuk Live Update (FIXED LOGIC) ---
  // Fungsi ini memeriksa apakah masih ada node di pohon `nodes`
  // yang namanya `resultName` dan belum diperluas (belum punya anak resep).
  const checkIfExpansionNeeded = useCallback((nodes, resultName, recipe) => {
      if (!Array.isArray(nodes) || !recipe) return false;
      const stack = [...nodes]; // Gunakan stack untuk traversal non-rekursif (lebih aman)

      while (stack.length > 0) {
          const node = stack.pop();
          if (!node) continue;

          // Cek node saat ini: apakah namanya cocok DAN bukan node resep itu sendiri
          if (node.name === resultName && node.attributes?.type !== 'Recipe') {
               // Cek apakah node ini *belum* memiliki anak resep *yang sesuai*
               const hasMatchingRecipeChild = node.children?.some(child =>
                   child.attributes?.type === 'Recipe' &&
                   child.attributes?.ingredient1 === recipe.ingredient1 &&
                   child.attributes?.ingredient2 === recipe.ingredient2
               );

               // Jika namanya cocok dan BELUM punya anak resep yang sesuai,
               // DAN bukan elemen dasar (tidak perlu diperluas), maka return true.
               if (!hasMatchingRecipeChild && !isBaseElement(node.name)) {
                   return true; // Ditemukan node yang perlu ekspansi dengan resep ini
               }
          }

          // Tambahkan anak ke stack untuk diperiksa (jika ada)
          if (node.children) {
              // Masukkan anak dalam urutan terbalik agar traversal mirip DFS
              for (let i = node.children.length - 1; i >= 0; i--) {
                  stack.push(node.children[i]);
              }
          }
      }
      return false; // Tidak ditemukan node yang perlu ekspansi dengan resep ini
  }, []); // useCallback karena ini fungsi murni


  // Efek untuk menjalankan timer live update (FIXED LOGIC)
  useEffect(() => {
    // Filter hanya live update yang aktif dan sedang berjalan
    const activeLiveUpdates = Object.entries(liveUpdateStates).filter(([, state]) => state.isActive && state.isBuilding);
    if (activeLiveUpdates.length === 0) return; // Keluar jika tidak ada yang perlu diproses

    const timers = activeLiveUpdates.map(([pathKey, state]) => {
      // Pastikan masih ada resep yang perlu diproses
      if (state.currentRecipeStep < state.fullPathRecipes.length) {
        // Set timer untuk langkah berikutnya
        const timerId = setTimeout(() => {
          setLiveUpdateStates(prev => {
            const currentPathState = prev[pathKey];
            // Safety check: pastikan state masih valid saat timer berjalan
            if (!currentPathState || !currentPathState.isActive || !currentPathState.isBuilding) return prev;

            // Ambil resep berikutnya berdasarkan step saat ini
            const nextRecipe = currentPathState.fullPathRecipes[currentPathState.currentRecipeStep];
            const imageURLs = results?.imageURLs || {};

            // Panggil buildTreeForNextStep untuk memperluas *satu* instance node hasil
            const partiallyUpdatedTree = buildTreeForNextStep(currentPathState.currentDisplayData, nextRecipe, imageURLs);

            // *** PEMERIKSAAN BARU (FIX) ***
            // Cek apakah *setelah* ekspansi tadi, *masih ada* node lain
            // di pohon yang cocok dengan hasil resep ini dan perlu diperluas.
            const needsMoreExpansionForThisRecipe = checkIfExpansionNeeded(partiallyUpdatedTree, nextRecipe.result, nextRecipe);

            // Tentukan indeks step berikutnya:
            // - Jika masih perlu ekspansi untuk resep *ini*, JANGAN increment step.
            // - Jika sudah tidak ada lagi node yang perlu diekspansi untuk resep *ini*, increment step.
            const nextStepIndex = needsMoreExpansionForThisRecipe
                ? currentPathState.currentRecipeStep // Tetap di step ini untuk memproses instance lain
                : currentPathState.currentRecipeStep + 1; // Lanjut ke resep berikutnya

            // Cek apakah seluruh proses live update selesai (sudah melewati resep terakhir)
            const isNowComplete = nextStepIndex >= currentPathState.fullPathRecipes.length;

            // Update state untuk pathKey ini
            return {
                ...prev,
                [pathKey]: {
                    ...currentPathState,
                    currentDisplayData: partiallyUpdatedTree, // Selalu update pohon dengan hasil ekspansi parsial
                    currentRecipeStep: nextStepIndex,      // Increment step secara kondisional
                    isBuilding: !isNowComplete,             // Berhenti building jika sudah selesai
                    isComplete: isNowComplete,
                    // Path identifier tidak perlu diubah di setiap sub-step resep yang sama
                    // pathIdentifier: `live-${pathKey}-${new Date().getTime()}` // Mungkin tidak perlu diupdate setiap saat?
                }
            };
          });
        }, LIVE_UPDATE_DELAY_MS); // Delay antar langkah
        return timerId; // Kembalikan ID timer untuk cleanup
      } else {
        // Jika step sudah melebihi jumlah resep, tandai selesai
        setLiveUpdateStates(prev => ({
            ...prev,
            [pathKey]: { ...prev[pathKey], isBuilding: false, isComplete: true }
        }));
        return null; // Tidak ada timer untuk langkah ini
      }
    });

    // Fungsi cleanup: batalkan semua timer jika komponen unmount atau state berubah
    return () => timers.forEach(id => { if (id) clearTimeout(id); });
  }, [liveUpdateStates, results?.imageURLs, checkIfExpansionNeeded]); // Tambahkan checkIfExpansionNeeded ke dependency array

  // Handler error untuk gambar yang gagal dimuat
  const handleImageError = (e) => { e.target.style.display = 'none'; };

  // Fungsi untuk merender satu langkah resep dalam daftar
  const renderStep = (step, stepIndex, pathIndex = null) => {
    // Validasi data langkah
    if (!step || typeof step !== 'object' || !step.result) return <li key={`invalid-step-${pathIndex}-${stepIndex}`} className="recipe-step-item invalid">Data langkah tidak valid</li>;

    const key = `${pathIndex !== null ? pathIndex + '-' : ''}${stepIndex}`;
    const imageURLs = results?.imageURLs || {};
    // Ambil URL gambar untuk ingredient dan hasil
    const imageUrl1 = imageURLs[step.ingredient1] ? `${API_BASE_URL}${imageURLs[step.ingredient1]}` : '';
    const imageUrl2 = imageURLs[step.ingredient2] ? `${API_BASE_URL}${imageURLs[step.ingredient2]}` : '';
    const imageUrlResult = imageURLs[step.result] ? `${API_BASE_URL}${imageURLs[step.result]}` : '';

    return (
      <li key={key} className="recipe-step-item">
        {/* Tampilkan gambar atau placeholder */}
        {imageUrl1 ? <img src={imageUrl1} alt={step.ingredient1 || ''} className="recipe-step-image" onError={handleImageError}/> : <span className="img-placeholder"></span>}
        {step.ingredient1 || '?'} {/* Tampilkan nama ingredient 1 */}
        <span className="recipe-step-separator">+</span>
        {imageUrl2 ? <img src={imageUrl2} alt={step.ingredient2 || ''} className="recipe-step-image" onError={handleImageError}/> : <span className="img-placeholder"></span>}
        {step.ingredient2 || '?'} {/* Tampilkan nama ingredient 2 */}
        <span className="recipe-step-separator">{' => '}</span>
        {imageUrlResult ? <img src={imageUrlResult} alt={step.result} className="recipe-step-image" onError={handleImageError}/> : <span className="img-placeholder"></span>}
        <strong className="recipe-step-result">{step.result}</strong> {/* Tampilkan nama hasil */}
      </li>);
  };

  // Fungsi untuk merender seluruh daftar langkah resep untuk satu path
  const renderPath = (path, pathIndex = null) => {
    // Validasi data path
    if (!Array.isArray(path)) return <p key={pathIndex !== null ? `invalid-path-data-${pathIndex}` : 'invalid-single-path-data'} className="invalid-path-message">Data jalur tidak valid.</p>;
    // Pesan jika path kosong (kecuali untuk elemen dasar)
    if (path.length === 0 && !isBaseElement(results?.searchTarget)) return <p key={pathIndex !== null ? `empty-path-${pathIndex}` : 'empty-single-path'} className="recipe-path-list-empty">(Tidak ada langkah resep)</p>;
    // Render daftar langkah menggunakan renderStep
    return <ol key={pathIndex !== null ? `path-${pathIndex}` : 'single-path-list'} className="recipe-path-list">{path.map((step, stepIndex) => renderStep(step, stepIndex, pathIndex))}</ol>;
  };

  // --- Render Utama Komponen ---

  // Tampilkan pesan loading
  if (isLoading) return <div className="loading-message">Memuat hasil pencarian...</div>;
  // Tampilkan pesan error jika ada
  if (error) return <div className="error-message">Error: {(typeof error === 'object' && error.message) ? error.message : String(error)} <img src={notFoundImage} alt="Path Not Found" style={{ marginTop: '20px', maxWidth: '900px', width: '100%', height: 'auto', objectFit: 'contain' }} /></div>;
  // Tampilkan pesan awal jika belum ada hasil
  if (!results || typeof results !== 'object') return <p className="initial-message">Silakan masukkan elemen yang ingin dicari.</p>;

  // Render hasil pencarian
  return (
    <div className="search-results-container">
      {/* Judul Hasil Pencarian */}
      <h2 className="results-title">
        Hasil Pencarian untuk: <strong className="target-element">{results.searchTarget || 'N/A'}</strong>
        <span className="search-info">
          ({results.algorithm?.toUpperCase() || 'N/A'} / {results.mode || 'N/A'}
          {results.mode === 'multiple' && ` - Maks: ${results.maxRecipes || 'N/A'}`})
        </span>
      </h2>
      {/* Statistik Pencarian (jika path ditemukan) */}
      {results.pathFound && (
         <div className="search-stats">
             <span>Node Diperiksa: <strong>{results.nodesVisited !== undefined && results.nodesVisited !== -1 ? results.nodesVisited.toLocaleString() : 'N/A'}</strong></span>
             <span>Durasi: <strong>{results.durationMillis ?? 'N/A'} ms</strong></span>
         </div> )}

       {/* Konten Hasil (jika path ditemukan) */}
       {results.pathFound ? (
           <>
            {/* Pesan khusus jika target adalah elemen dasar */}
            {isBaseElement(results.searchTarget) && (
                ((results.mode === 'shortest' && (!Array.isArray(results.path) || results.path.length === 0)) ||
                 (results.mode === 'multiple' && (!Array.isArray(results.paths) || results.paths.length === 0 || (Array.isArray(results.paths) && results.paths.every(p => !Array.isArray(p) || p.length === 0)))))
            ) && ( <div className="base-element-message">"{results.searchTarget}" adalah elemen dasar.</div> )}

           {/* --- Render untuk Mode Shortest --- */}
           {results.mode === 'shortest' && Array.isArray(results.path) && results.path.length > 0 && (() => {
                const pathKey = `path-block-shortest-0`;
                const currentLiveState = liveUpdateStates[pathKey];
                const staticTreeData = treeDataForStaticView[pathKey];
                // Pilih data pohon: live data jika aktif, jika tidak, data statis
                const treeToDisplay = currentLiveState?.isActive && currentLiveState.currentDisplayData ? currentLiveState.currentDisplayData : staticTreeData;
                return (
                   <div key={pathKey} className="path-block"> {/* Layout flex row */}
                       <div className="path-details-column"> {/* Kolom kiri: teks & tombol */}
                           <div className="path-text-section">
                               <h4 className="path-title">Jalur Terpendek (Langkah: {results.path.length})</h4>
                               {renderPath(results.path, `shortest-0`)} {/* Render daftar langkah */}
                           </div>
                           {/* Tombol Live Update (hanya jika bukan elemen dasar) */}
                           {!isBaseElement(results.searchTarget) && (
                               <button
                                   onClick={() => startLiveUpdate(pathKey, results.path, results.searchTarget, results.imageURLs)}
                                   disabled={currentLiveState?.isBuilding} // Nonaktifkan saat building
                                   className="live-update-button">
                                   {/* Teks tombol dinamis */}
                                   {currentLiveState?.isBuilding ? 'Memproses...' : (currentLiveState?.isComplete ? 'Putar Ulang Live' : 'Mulai Live Update')}
                               </button>
                           )}
                       </div>
                       {/* Kolom kanan: Visualisasi Pohon (jika data pohon valid) */}
                       {treeToDisplay && treeToDisplay.length > 0 && treeToDisplay[0] && (
                           <div className="path-visualization-column">
                                <h5 className="visualization-title-small">Visualisasi {currentLiveState?.isActive ? "(Live)" : ""}</h5>
                                <div id={`treeWrapper-shortest-0`} className="tree-wrapper">
                                    <Tree data={treeToDisplay} orientation="vertical" translate={{ x: 300, y: 50 }} // Posisi awal pohon
                                        renderCustomNodeElement={renderNodeWithImage} // Gunakan fungsi render custom
                                        zoomable={true} draggable={true} // Aktifkan interaksi
                                        nodeSize={{ x: 140, y: 120 }} // Ukuran node
                                        separation={{ siblings: 1.2, nonSiblings: 1.5 }} // Jarak antar node
                                        pathFunc="straight" // Jenis garis penghubung
                                        depthFactor={150} // Jarak antar level kedalaman
                                        key={currentLiveState?.pathIdentifier || `static-tree-${pathKey}`} // Key penting untuk re-render/transisi
                                    />
                                </div>
                           </div>
                       )}
                   </div>);
            })()}

           {/* --- Render untuk Mode Multiple --- */}
           {results.mode === 'multiple' && Array.isArray(results.paths) && results.paths.length > 0 &&
            results.paths.map((path, index) => {
                // Lewati path yang tidak valid atau path kosong untuk non-base element
                if (!Array.isArray(path) || (path.length === 0 && !isBaseElement(results.searchTarget))) return null;
                // Lewati path kosong jika targetnya base element (sudah ditangani di pesan base element)
                if (path.length === 0 && isBaseElement(results.searchTarget)) return null;

                const pathKey = `path-block-multiple-${index}`;
                const currentLiveState = liveUpdateStates[pathKey];
                const staticTreeData = treeDataForStaticView[pathKey];
                // Pilih data pohon: live data jika aktif, jika tidak, data statis
                const treeToDisplay = currentLiveState?.isActive && currentLiveState.currentDisplayData ? currentLiveState.currentDisplayData : staticTreeData;
                return (
                   <div key={pathKey} className="path-block"> {/* Layout flex row */}
                        <div className="path-details-column"> {/* Kolom kiri: teks & tombol */}
                           <div className="path-text-section">
                               <h4 className="path-title">Jalur {index + 1} (Langkah: {path.length})</h4>
                               {renderPath(path, `multiple-${index}`)} {/* Render daftar langkah */}
                           </div>
                           {/* Tombol Live Update (hanya jika bukan elemen dasar dan path tidak kosong) */}
                           {!isBaseElement(results.searchTarget) && path.length > 0 && (
                               <button
                                   onClick={() => startLiveUpdate(pathKey, path, results.searchTarget, results.imageURLs)}
                                   disabled={currentLiveState?.isBuilding} // Nonaktifkan saat building
                                   className="live-update-button">
                                   {/* Teks tombol dinamis */}
                                   {currentLiveState?.isBuilding ? 'Memproses...' : (currentLiveState?.isComplete ? 'Putar Ulang Live' : 'Mulai Live Update')}
                               </button>
                           )}
                        </div>
                         {/* Kolom kanan: Visualisasi Pohon (jika data pohon valid) */}
                         {treeToDisplay && treeToDisplay.length > 0 && treeToDisplay[0] && (
                           <div className="path-visualization-column">
                                <h5 className="visualization-title-small">Visualisasi Jalur {index + 1} {currentLiveState?.isActive ? "(Live)" : ""}</h5>
                                <div id={`treeWrapper-multiple-${index}`} className="tree-wrapper">
                                     <Tree data={treeToDisplay} orientation="vertical" translate={{ x: 300, y: 50 }} // Posisi awal pohon
                                         renderCustomNodeElement={renderNodeWithImage} // Gunakan fungsi render custom
                                         zoomable={true} draggable={true} // Aktifkan interaksi
                                         nodeSize={{ x: 140, y: 120 }} // Ukuran node
                                         separation={{ siblings: 1.2, nonSiblings: 1.5 }} // Jarak antar node
                                         pathFunc="straight" // Jenis garis penghubung
                                         depthFactor={150} // Jarak antar level kedalaman
                                         key={currentLiveState?.pathIdentifier || `static-tree-${pathKey}`} // Key penting untuk re-render/transisi
                                     />
                                 </div>
                           </div>
                         )}
                   </div>);
            })}
           </>
       ) : (
            // Pesan jika path tidak ditemukan (dan bukan karena loading/error)
            results && results.searchTarget &&
            <div className="path-not-found-message">
                 Jalur tidak ditemukan untuk elemen "{results.searchTarget}".
                 {results.error ? ` (Error: ${results.error})` : ''}
                 {/* Mungkin tampilkan gambar not found di sini juga jika diinginkan */}
            </div>
       )}
    </div>
  );
}

export default SearchResults;
