import React, { useState } from 'react';
import './SearchForm.css';

function SearchForm({ onSearchSubmit, isLoading }) {
  const [target, setTarget] = useState('');
  const [algo, setAlgo] = useState('bfs');
  const [mode, setMode] = useState('shortest');
  const [maxRecipes, setMaxRecipes] = useState(1);

  const handleSubmit = (event) => {
    event.preventDefault();
    if (!target) {
      alert('Masukkan elemen target!');
      return;
    }
    if (mode === 'multiple' && (!maxRecipes || maxRecipes <= 0)) {
        alert('Masukkan jumlah resep minimal 1 untuk mode multiple!');
        return;
    }

    const searchParams = { target, algo, mode };
    if (mode === 'multiple') {
        searchParams.max = maxRecipes;
    }
    onSearchSubmit(searchParams);
  };

  return (
    <form onSubmit={handleSubmit} className="search-form-container">
      <div className="search-and-algo-group">


        <div className="form-options-group algo-group">
          <div className="radio-group">
                    <div className="target-input-group">
          <input
            type="text"
            id="targetElement"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder="Contoh: Mud, Human, ..."
            required
            className="form-input"
          />
        </div>
            {/* Opsi BFS */}
            <input
              type="radio"
              id="algo-bfs" // ID unik
              value="bfs"
              checked={algo === 'bfs'}
              onChange={(e) => setAlgo(e.target.value)}
              className="radio-input"
            />
            <label htmlFor="algo-bfs" className="radio-label"> {/* htmlFor merujuk ke ID input */}
              BFS
            </label>

            {/* Opsi DFS */}
            <input
              type="radio"
              id="algo-dfs" // ID unik
              value="dfs"
              checked={algo === 'dfs'}
              onChange={(e) => setAlgo(e.target.value)}
              className="radio-input"
            />
            <label htmlFor="algo-dfs" className="radio-label"> {/* htmlFor merujuk ke ID input */}
              DFS
            </label>

            {/* Opsi Bidirectional */}
            <input
              type="radio"
              id="algo-bds" // ID unik
              value="bds"
              checked={algo === 'bds'}
              onChange={(e) => setAlgo(e.target.value)}
              className="radio-input"
            />
            <label htmlFor="algo-bds" className="radio-label"> {/* htmlFor merujuk ke ID input */}
              Bidirectional
            </label>
          </div>
        </div>
      </div>

      <div className="form-options-group mode-group">
        <div className="radio-group">
          {/* Opsi Shortest */}
          <input
            type="radio"
            id="mode-shortest" // ID unik
            value="shortest"
            checked={mode === 'shortest'}
            onChange={(e) => setMode(e.target.value)}
            className="radio-input"
          />
          <label htmlFor="mode-shortest" className="radio-label"> {/* htmlFor merujuk ke ID input */}
            Shortest
          </label>

          {/* Opsi Multiple */}
          <input
            type="radio"
            id="mode-multiple" // ID unik
            value="multiple"
            checked={mode === 'multiple'}
            onChange={(e) => setMode(e.target.value)}
            className="radio-input"
          />
          
          <label htmlFor="mode-multiple" className="radio-label"> {/* htmlFor merujuk ke ID input */}
            Multiple
          </label>

          {mode === 'multiple' &&(<div className="max-recipes-group">
            <input
              type="number"
              id="maxRecipes"
              value={maxRecipes}
              onChange={(e) => setMaxRecipes(parseInt(e.target.value, 10) || 1)}
              min="1"
              className="max-recipes-input"
            />
          </div>
          )}

                          <button
        type="submit"
        disabled={isLoading}
        className="submit-button"
      >
        

        {isLoading ? 'Mencari...' : 'Cari Resep'}
      </button>


        </div>
      </div>
    </form>
  );
}

export default SearchForm;
