import { apiUrl } from './config';

/** Asks the backend whether it will accept this URL. Resolves to an error message, or null when accepted. */
export async function checkStreamUrl(url: string): Promise<string | null> {
  try {
    const res = await fetch(apiUrl('/api/validate'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url }),
    });
    const body = (await res.json()) as { ok: boolean; error?: string };
    return body.ok ? null : (body.error ?? 'The server rejected this URL.');
  } catch {
    return 'Cannot reach the stream server. Check that the backend is running.';
  }
}
