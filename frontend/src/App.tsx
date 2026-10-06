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
        <div className="top__identity">
          <div className="top__icon" aria-hidden="true">◉</div>
          <div>
            <p className="top__eyebrow">VIDEO MONITORING</p>
            <h1>RTSP Stream Viewer</h1>
            <p className="top__sub">Add camera streams and watch them live in your browser.</p>
          </div>
        </div>
        <div className="top__stats" aria-label="Stream status summary">
          <div className="top__summary">
            <span className="top__summary-icon" aria-hidden="true">◉</span>
            <span className="top__summary-value">{streams.length}</span>
            <span className="top__summary-label">Active streams</span>
          </div>
        </div>
      </header>

      <AddStreamForm onAdd={add} />

      <section className="streams-panel" aria-label="Streams">
        <div className="toolbar">
          <p className="toolbar__count" aria-live="polite">
            {streams.length === 0 ? 'No streams yet' : `${streams.length} ${streams.length === 1 ? 'Stream' : 'Streams'}`}
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
            <button type="button" className="btn btn--danger" onClick={clear}>
              <span aria-hidden="true">⌫</span> Remove all
            </button>
          )}
        </div>

        <main className="streams-scroll">
          <StreamGrid streams={streams} columns={columns} onRemove={remove} />
        </main>
      </section>
    </div>
  );
}
