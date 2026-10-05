const apiBase = (import.meta.env.VITE_API_URL ?? '').replace(/\/+$/, '');

export function apiUrl(path: string): string {
  return `${apiBase}${path}`;
}

/** WebSocket endpoint for one RTSP stream. Works for same-origin (dev proxy) and cross-origin (prod). */
export function wsUrl(rtspUrl: string): string {
  const u = new URL('/ws', apiBase || window.location.origin);
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
  u.searchParams.set('url', rtspUrl);
  return u.toString();
}

export const demoStreams: string[] = (import.meta.env.VITE_DEMO_STREAMS ?? '')
  .split(',')
  .map((s) => s.trim())
  .filter(Boolean);

/** Close codes sent by the backend (see backend/internal/api). */
export const CLOSE_FATAL = 4001; // bad URL, auth failure, not found, at capacity: retrying will not help
export const CLOSE_INTERRUPTED = 4002; // stream dropped after it was working: safe to retry
