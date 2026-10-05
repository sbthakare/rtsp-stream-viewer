import { useState } from 'react';
import { AddStreamForm } from './components/AddStreamForm';
import { StreamGrid } from './components/StreamGrid';
import { useStreams } from './hooks/useStreams';
import type { GridColumns } from './types';

const LAYOUTS: { value: GridColumns; label: string }[] = [
  { value: 'auto', label: 'Auto' },
  { value: 1, label: '1' },
  { value: 2, label: '2' },
  { value: 3, label: '3' },
  { value: 4, label: '4' },
];

export default function App() {
  const { streams, add, remove, clear } = useStreams();
  const [columns, setColumns] = useState<GridColumns>('auto');

  return (
    <div className="page">
      <header className="top">
        <h1>RTSP Stream Viewer</h1>
        <p className="top__sub">Add camera streams and watch them live in your browser.</p>
      </header>

      <AddStreamForm onAdd={add} />

      <div className="toolbar">
        <p className="toolbar__count" aria-live="polite">
          {streams.length === 0 ? 'No streams' : `${streams.length} ${streams.length === 1 ? 'stream' : 'streams'}`}
        </p>
        <div className="toolbar__group" role="group" aria-label="Grid columns">
          <span className="toolbar__label">Columns</span>
          {LAYOUTS.map((l) => (
            <button
              key={String(l.value)}
              type="button"
              className="seg"
              aria-pressed={columns === l.value}
              onClick={() => setColumns(l.value)}
            >
              {l.label}
            </button>
          ))}
        </div>
        {streams.length > 0 && (
          <button type="button" className="btn" onClick={clear}>
            Remove all
          </button>
        )}
      </div>

      <main>
        <StreamGrid streams={streams} columns={columns} onRemove={remove} />
      </main>
    </div>
  );
}
