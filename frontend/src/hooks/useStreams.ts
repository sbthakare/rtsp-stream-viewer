import { useCallback, useEffect, useState } from 'react';
import type { StreamItem } from '../types';
import { describeUrl } from '../lib/rtsp';

const STORAGE_KEY = 'rtsp-viewer:streams:v1';

function load(): StreamItem[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const parsed: unknown = raw ? JSON.parse(raw) : [];
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (s): s is StreamItem => s && typeof s.id === 'string' && typeof s.url === 'string' && typeof s.name === 'string',
    );
  } catch {
    return [];
  }
}

export function useStreams() {
  const [streams, setStreams] = useState<StreamItem[]>(load);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(streams));
    } catch {
      /* storage unavailable (private mode, quota): the list just will not persist */
    }
  }, [streams]);

  /** Returns an error message when the stream cannot be added, otherwise null. */
  const add = useCallback(
    (url: string, name: string): string | null => {
      if (streams.some((s) => s.url === url)) return 'That stream is already on the wall.';
      const item: StreamItem = {
        id: crypto.randomUUID(),
        url,
        name: name.trim() || describeUrl(url).name,
      };
      setStreams((prev) => [...prev, item]);
      return null;
    },
    [streams],
  );

  const remove = useCallback((id: string) => setStreams((prev) => prev.filter((s) => s.id !== id)), []);
  const clear = useCallback(() => setStreams([]), []);

  return { streams, add, remove, clear };
}
