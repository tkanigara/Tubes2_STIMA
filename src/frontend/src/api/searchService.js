// src/frontend/src/api/searchService.js

// Ini akan menjadi URL fallback jika VITE_API_BASE_URL tidak diset saat proses build.
// Sangat berguna untuk pengembangan lokal jika Anda tidak membuat file .env.development.
const FALLBACK_API_BASE_URL = "http://localhost:8080";

// Vite akan secara otomatis mengganti import.meta.env.VITE_API_BASE_URL
// dengan nilai variabel lingkungan yang sesuai saat proses 'npm run build'.
// Jika tidak ada, ia akan menggunakan FALLBACK_API_BASE_URL.
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

/**
 * Fungsi untuk memanggil endpoint /api/search
 * @param {string} target Nama elemen target
 * @param {string} algo Algoritma ('bfs' atau 'dfs' atau 'bds')
 * @param {string} mode Mode ('shortest' atau 'multiple')
 * @param {number} [maxRecipes] Jumlah maksimal resep (hanya untuk mode 'multiple')
 * @returns {Promise<object>} Promise yang resolve dengan data JSON dari API
 */
async function findRecipes(target, algo, mode, maxRecipes) {
  const params = new URLSearchParams({ target, algo, mode });

  if (mode === 'multiple' && maxRecipes && maxRecipes > 0) {
    params.append('max', maxRecipes.toString());
  }

  // Perhatikan di sini: kita menggabungkan API_BASE_URL dengan path spesifik '/api/search'
  const url = `${API_BASE_URL}/api/search?${params.toString()}`;

  console.log(`Frontend: Mengirim request ke: ${url}`);

  try {
    const response = await fetch(url);

    if (!response.ok) {
      //const errorData = await response.json().catch(() => ({ message: response.statusText }));
      const backendErrorMessage = 'Elemen tidak ditemukan';
      throw new Error(`API Error (${response.status}): ${backendErrorMessage}`);
    }

    const data = await response.json();
    console.log("Frontend: Menerima data:", data);
    return data;

  } catch (error) {
    console.error("Frontend: Gagal mengambil resep dari API:", error);
    throw error; // Lempar ulang error agar bisa ditangani lebih lanjut
  }
}

/**
 * Fungsi untuk mendapatkan URL gambar elemen yang akan diproxy oleh backend.
 * @param {string} elementName Nama elemen
 * @returns {string} URL lengkap ke endpoint proxy gambar backend
 */
function getElementImageURL(elementName) {
  // Path '/api/image' ditambahkan di sini:
  return `${API_BASE_URL}/api/image?elementName=${encodeURIComponent(elementName)}`;
}

// Ekspor fungsi agar bisa digunakan di komponen React atau JavaScript lain
export { findRecipes, getElementImageURL };