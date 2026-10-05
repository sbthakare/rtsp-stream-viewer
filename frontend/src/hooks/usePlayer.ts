import { useEffect, useRef, useState, type RefObject } from 'react';
import JSMpeg from '@cycjimmy/jsmpeg-player';
import { CLOSE_FATAL, wsUrl } from '../lib/config';
import { createSource } from '../lib/jsmpegSource';
import type { TileStatus } from '../types';

const MAX_AUTO_RETRIES = 5;

export interface PlayerState {
  status: TileStatus;
  message: string;
}

/**
 * Connects one canvas to one stream and keeps it alive.
 *
 * - `paused` tears the connection down completely (the server stops FFmpeg once nobody is watching).
 * - Interrupted streams reconnect automatically with exponential backoff.
 * - Fatal errors (bad credentials, missing path, ...) stop and show the server's explanation.
 *
 * `run` changes whenever a fresh connection is wanted. The caller keys the <canvas> on it, because a
 * canvas that has held a WebGL context cannot be reused by a new player.
 */
export function usePlayer(canvasRef: RefObject<HTMLCanvasElement>, url: string, paused: boolean, run: number, bump: () => void) {
  const [state, setState] = useState<PlayerState>({ status: 'connecting', message: '' });
  const attempts = useRef(0);
  const bumpRef = useRef(bump);
  bumpRef.current = bump;

  useEffect(() => {
    if (paused) {
      setState({ status: 'paused', message: '' });
      return;
    }
    const canvas = canvasRef.current;
    if (!canvas) return;

    let disposed = false;
    let gotFrame = false;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    setState({ status: attempts.current > 0 ? 'reconnecting' : 'connecting', message: '' });

    const Source = createSource({
      onClose: (code, reason) => {
        if (disposed) return;
        const fatal = code === CLOSE_FATAL;
        if (fatal || attempts.current >= MAX_AUTO_RETRIES) {
          setState({
            status: 'error',
            message: reason || (fatal ? 'The stream could not be opened.' : 'The connection was lost and could not be restored.'),
          });
          return;
        }
        attempts.current += 1;
        const delay = Math.min(1000 * 2 ** (attempts.current - 1), 10000);
        setState({ status: 'reconnecting', message: reason });
        retryTimer = setTimeout(() => bumpRef.current(), delay);
      },
    });

    const player = new JSMpeg.Player(wsUrl(url), {
      canvas,
      source: Source,
      autoplay: true,
      audio: false,
      loop: false,
      pauseWhenHidden: false,
      videoBufferSize: 1024 * 1024,
      onVideoDecode: () => {
        if (gotFrame || disposed) return;
        gotFrame = true;
        attempts.current = 0;
        setState({ status: 'live', message: '' });
      },
    });

    return () => {
      disposed = true;
      clearTimeout(retryTimer);
      try {
        player.destroy();
      } catch {
        /* player may already be torn down */
      }
    };
  }, [canvasRef, url, paused, run]);

  /** Manual retry after an error. */
  const retry = () => {
    attempts.current = 0;
    bumpRef.current();
  };

  return { ...state, retry };
}
