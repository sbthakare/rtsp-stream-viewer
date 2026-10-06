import type { CSSProperties } from 'react';
import type { GridColumns, StreamItem, TileStatus } from '../types';
import { StreamTile } from './StreamTile';

interface Props {
  streams: StreamItem[];
  columns: GridColumns;
  onRemove: (id: string) => void;
  onStatusChange: (id: string, status: TileStatus) => void;
}

export function StreamGrid({ streams, columns, onRemove, onStatusChange }: Props) {
  if (streams.length === 0) {
    return (
      <div className="empty">
        <h2>No streams yet</h2>
        <p>Paste an RTSP URL above and press Add stream. Add several to watch them side by side.</p>
      </div>
    );
  }
  const style = { '--cols': columns === 'auto' ? undefined : columns } as CSSProperties;
  return (
    <div className={`grid ${columns === 'auto' ? 'grid--auto' : `grid--fixed grid--cols-${columns}`}`} style={style}>
      {streams.map((s) => (
        <StreamTile key={s.id} item={s} onRemove={onRemove} onStatusChange={onStatusChange} />
      ))}
    </div>
  );
}
