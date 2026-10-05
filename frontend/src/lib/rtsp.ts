export type ParseResult = { ok: true; url: string } | { ok: false; error: string };

export function parseRtspUrl(raw: string): ParseResult {
  const value = raw.trim();
  if (!value) return { ok: false, error: 'Enter an RTSP URL, for example rtsp://host:8554/cam1.' };
  let u: URL;
  try {
    u = new URL(value);
  } catch {
    return { ok: false, error: 'That does not look like a valid URL.' };
  }
  if (u.protocol !== 'rtsp:' && u.protocol !== 'rtsps:') {
    return { ok: false, error: 'The URL must start with rtsp:// or rtsps://.' };
  }
  if (!u.hostname) return { ok: false, error: 'The URL is missing a host name.' };
  return { ok: true, url: value };
}

/** Human readable label and a URL safe to show on screen (no password). */
export function describeUrl(raw: string): { name: string; masked: string } {
  try {
    const u = new URL(raw);
    const hadCreds = Boolean(u.username || u.password);
    u.username = '';
    u.password = '';
    const masked = `${hadCreds ? '***@' : ''}${u.host}${u.pathname}${u.search}`;
    const path = u.pathname.replace(/^\/+|\/+$/g, '');
    return { name: path ? `${u.hostname}/${path}` : u.hostname, masked: `${u.protocol}//${masked}` };
  } catch {
    return { name: raw, masked: raw };
  }
}
