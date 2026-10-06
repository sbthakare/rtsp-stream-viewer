import { useRef, useState } from 'react';
import { usePlayer } from '../hooks/usePlayer';
import { describeUrl } from '../lib/rtsp';
import type { StreamItem, TileStatus } from '../types';

const STATUS_LABEL: Record<TileStatus, string> = {
  connecting: 'Connecting',
  live: 'Live',
  reconnecting: 'Reconnecting',
  paused: 'Paused',
  error: 'Error',
};

interface Props {
  item: StreamItem;
  onRemove: (id: string) => void;
}

export function StreamTile({ item, onRemove }: Props) {
  const tileRef = useRef<HTMLElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [paused, setPaused] = useState(false);
  const [run, setRun] = useState(0);
  const { status, message, retry } = usePlayer(canvasRef, item.url, paused, run, () => setRun((r) => r + 1));

  const togglePause = () => {
    if (paused) setRun((r) => r + 1); // fresh canvas for the new connection
    setPaused(!paused);
  };

  const toggleFullscreen = () => {
    if (document.fullscreenElement) void document.exitFullscreen();
    else void tileRef.current?.requestFullscreen?.();
  };

  const { masked } = describeUrl(item.url);
  const showOverlay = status !== 'live';

  return (
    <article className="tile" ref={tileRef} aria-label={item.name}>
      <div className="tile__screen">
        <canvas key={run} ref={canvasRef} className="tile__canvas" />
        <span className={`tile__screen-status tile__screen-status--${status}`}>{STATUS_LABEL[status]}</span>
        <button type="button" className="tile__screen-fullscreen" onClick={toggleFullscreen} aria-label={`Fullscreen ${item.name}`}>
          ⛶
        </button>
        {showOverlay && (
          <div className="tile__overlay" role={status === 'error' ? 'alert' : 'status'}>
            <p className="tile__overlay-title">
              {status === 'connecting' && 'Connecting to stream…'}
              {status === 'reconnecting' && 'Connection dropped. Reconnecting…'}
              {status === 'paused' && 'Paused'}
              {status === 'error' && 'Cannot play this stream'}
            </p>
            {message && <p className="tile__overlay-detail">{message}</p>}
            {status === 'error' && (
              <button type="button" className="btn btn--light" onClick={retry}>
                Try again
              </button>
            )}
          </div>
        )}
      </div>
      <div className="tile__bar">
        <div className="tile__meta">
          <h3 className="tile__name" title={item.name}>
            <span aria-hidden="true">◉</span>{item.name}
          </h3>
          <p className="tile__url" title={masked}>
            {masked}
          </p>
        </div>
        <span className={`status status--${status}`}>{STATUS_LABEL[status]}</span>
        <div className="tile__actions">
          <button type="button" className="btn" onClick={togglePause} aria-pressed={paused}>
            <span aria-hidden="true">{paused ? '▶' : 'Ⅱ'}</span>{paused ? 'Play' : 'Pause'}
          </button>
          <button type="button" className="btn" onClick={toggleFullscreen}>
            <span aria-hidden="true">⛶</span>Fullscreen
          </button>
          <button type="button" className="btn btn--danger" onClick={() => onRemove(item.id)}>
            <span aria-hidden="true">⌫</span>Remove
          </button>
        </div>
      </div>
    </article>
  );
}
