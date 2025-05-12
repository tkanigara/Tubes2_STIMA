// src/frontend/src/api/searchService.js

// Ini akan menjadi URL fallback jika VITE_API_BASE_URL tidak diset saat proses build.
// Sangat berguna untuk pengembangan lokal jika Anda tidak membuat file .env.development.
const FALLBACK_API_BASE_URL = "http://localhost:8080";

// Vite akan secara otomatis mengganti import.meta.env.VITE_API_BASE_URL
// dengan nilai variabel lingkungan yang sesuai saat proses 'npm run build'.
// Jika tidak ada, ia akan menggunakan FALLBACK_API_BASE_URL.
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || FALLBACK_API_BASE_URL;

// Deteksi mode development vs production untuk debugging
const IS_PRODUCTION = import.meta.env.PROD || (import.meta.env.VITE_API_BASE_URL || '').includes('railway.app') || API_BASE_URL === '/api';

console.log(`Frontend: Running in ${IS_PRODUCTION ? 'PRODUCTION' : 'DEVELOPMENT'} mode`);
console.log("Frontend: Using API_BASE_URL:", API_BASE_URL);

// Fungsi untuk membuat URL API yang benar
// Jika API_BASE_URL adalah URL relatif (dimulai dengan /), maka gunakan origin window saat ini
// Jika API_BASE_URL adalah URL absolut (termasuk http:// atau https://), gunakan apa adanya
function makeApiUrl(path) {
  // Jika API_BASE_URL adalah URL relatif (mulai dengan /)
  if (API_BASE_URL.startsWith('/')) {
    return `${window.location.origin}${API_BASE_URL}${path}`;
  }
  
  // Jika API_BASE_URL adalah URL absolut
  return `${API_BASE_URL}${path}`;
}

// First, try to ping the backend to ensure it's available
async function pingBackend() {
  try {
    const pingUrl = makeApiUrl('/ping');
    console.log(`Frontend: Pinging backend at: ${pingUrl}`);
    
    const response = await fetch(pingUrl, { 
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      mode: 'cors', // Explicitly set CORS mode
    });
    
    if (response.ok) {
      const data = await response.json();
      console.log("Frontend: Backend ping successful:", data);
      return true;
    } else {
      console.error("Frontend: Backend ping failed with status:", response.status);
      return false;
    }
  } catch (error) {
    console.error("Frontend: Backend ping error:", error);
    return false;
  }
}

// Try to ping backend when module loads
pingBackend().then(isOnline => {
  if (isOnline) {
    console.log("Backend connection confirmed!");
  } else {
    console.warn("Warning: Could not connect to backend. Some features may not work.");
  }
});

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

  // Use makeApiUrl function to handle both relative and absolute URLs
  const url = makeApiUrl('/search') + '?' + params.toString();

  console.log(`Frontend: Mengirim request ke: ${url}`);

  try {
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      mode: 'cors', // Explicitly set CORS mode
      credentials: 'omit', // Don't send cookies
    });

    console.log("Frontend: Response status:", response.status);
    console.log("Frontend: Response headers:", [...response.headers].map(h => `${h[0]}: ${h[1]}`).join(", "));

    if (!response.ok) {
      let errorMessage = 'Unknown error';
      try {
        const errorData = await response.json();
        errorMessage = errorData.message || errorData.error || `Error ${response.status}`;
      } catch (e) {
        errorMessage = response.statusText || 'Adiiiitt Elemen kamu gaadaaa';
      }
      throw new Error(`API Error (${response.status}): ${errorMessage}`);
    }

    const data = await response.json();
    console.log("Frontend: Menerima data:", data);
    return data;

  } catch (error) {
    console.error("Frontend: Gagal mengambil resep dari API:", error);
    console.error("Error stack:", error.stack);
    throw error; // Lempar ulang error agar bisa ditangani lebih lanjut
  }
}

/**
 * Fungsi untuk mendapatkan URL gambar elemen yang akan diproxy oleh backend.
 * @param {string} elementName Nama elemen
 * @returns {string} URL lengkap ke endpoint proxy gambar backend
 */
function getElementImageURL(elementName) {
  // Use makeApiUrl function to handle both relative and absolute URLs
  const imageUrl = makeApiUrl('/image') + `?elementName=${encodeURIComponent(elementName)}`;
  console.log(`Frontend: Generated image URL: ${imageUrl}`);
  return imageUrl;
}

// Ekspor fungsi agar bisa digunakan di komponen React atau JavaScript lain
export { findRecipes, getElementImageURL, pingBackend };