import { useState, type FormEvent } from 'react';
import { checkStreamUrl } from '../lib/api';
import { demoStreams } from '../lib/config';
import { parseRtspUrl } from '../lib/rtsp';

interface Props {
  onAdd: (url: string, name: string) => string | null;
}

export function AddStreamForm({ onAdd }: Props) {
  const [url, setUrl] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(rawUrl: string, label: string) {
    const parsed = parseRtspUrl(rawUrl);
    if (!parsed.ok) return setError(parsed.error);
    setBusy(true);
    setError(null);
    const serverError = await checkStreamUrl(parsed.url);
    setBusy(false);
    if (serverError) return setError(serverError);
    const addError = onAdd(parsed.url, label);
    if (addError) return setError(addError);
    setUrl('');
    setName('');
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    void submit(url, name);
  }

  return (
    <section className="add" aria-labelledby="add-heading">
      <h2 id="add-heading" className="visually-hidden">Add a stream</h2>
      <form className="add__form" onSubmit={handleSubmit} noValidate>
        <div className="field field--grow">
          <label htmlFor="stream-url">Stream URL</label>
          <input
            id="stream-url"
            type="text"
            inputMode="url"
            autoComplete="off"
            spellCheck={false}
            placeholder="rtsp://user:password@host:8554/path"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? 'add-error' : undefined}
          />
        </div>
        <div className="field">
          <label htmlFor="stream-name">Stream name <span>(optional)</span></label>
          <input id="stream-name" type="text" placeholder="Front door" value={name} onChange={(e) => setName(e.target.value)} maxLength={60} />
        </div>
        <button type="submit" className="btn btn--primary btn--add" disabled={busy}>
          <span aria-hidden="true">+</span>{busy ? 'Checking...' : 'Add stream'}
        </button>
      </form>
      {error && <p id="add-error" className="add__error" role="alert">{error}</p>}
      {demoStreams.length > 0 && (
        <p className="add__demo">
          Test streams:
          {demoStreams.map((d) => (
            <button key={d} type="button" className="chip" onClick={() => void submit(d, '')} disabled={busy}>
              {d.split('/').pop()}
            </button>
          ))}
        </p>
      )}
    </section>
  );
}
