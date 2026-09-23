import { useCallback, useEffect, useState } from 'react';
import { FileText, Upload, X } from 'lucide-react';
import { api } from '../lib/api';
import { appPath } from '../lib/app-paths';

interface Attachment {
  attachmentID: number;
  documentID?: number;
  originalFilename: string;
  fileSizeFormatted: string;
  source: 'attachment' | 'document';
  uploadedAt: string;
}

interface HistoryEntry {
  history_id: number;
  changed_at: string;
  description: string;
  user_name: string;
}

export default function JobActivityPanel({ jobId }: { jobId: number }) {
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [files, changes] = await Promise.all([
        api.get<Attachment[]>(`/jobs/${jobId}/attachments`),
        api.get<{ history: HistoryEntry[] }>(`/jobs/${jobId}/history`),
      ]);
      setAttachments(files.data || []);
      setHistory(changes.data.history || []);
      setError('');
    } catch {
      setError('Dokumente und Verlauf konnten nicht geladen werden.');
    } finally {
      setLoading(false);
    }
  }, [jobId]);

  useEffect(() => { load(); }, [load]);

  const upload = async (file?: File) => {
    if (!file) return;
    setBusy(true);
    setError('');
    const data = new FormData();
    data.append('entityType', 'job');
    data.append('entityID', String(jobId));
    data.append('documentType', 'job_file');
    data.append('file', file);
    try {
      const response = await fetch(appPath('/documents/upload'), { method: 'POST', credentials: 'include', body: data });
      if (!response.ok) throw new Error('Upload failed');
      await load();
    } catch {
      setError('Datei konnte nicht hochgeladen werden.');
    } finally {
      setBusy(false);
    }
  };

  const remove = async (file: Attachment) => {
    if (!confirm('Datei aus diesem Job entfernen?')) return;
    setBusy(true);
    setError('');
    try {
      if (file.source === 'document' && file.documentID) {
        const response = await fetch(appPath(`/documents/${file.documentID}`), { method: 'DELETE', credentials: 'include' });
        if (!response.ok) throw new Error('Delete failed');
      } else {
        await api.delete(`/jobs/attachments/${file.attachmentID}`);
      }
      await load();
    } catch {
      setError('Datei konnte nicht entfernt werden.');
    } finally {
      setBusy(false);
    }
  };

  return <>
    <section className="jobs-card">
      <div className="jobs-card-heading"><h2>Dokumente</h2></div>
      <div className="jobs-card-body">
        <label className="suite-button inline-flex cursor-pointer items-center gap-2">
          <Upload className="w-4 h-4" /> Datei hinzufügen
          <input type="file" className="sr-only" disabled={busy} onChange={(event) => { upload(event.target.files?.[0]); event.target.value = ''; }} />
        </label>
        {loading ? <p className="jobs-form-note mt-3">Lädt…</p> : attachments.length === 0 ? <p className="jobs-form-note mt-3">Noch keine Dokumente vorhanden.</p> :
          <ul className="mt-3 space-y-2">
            {attachments.map((file) => <li key={`${file.source}-${file.attachmentID}`} className="flex items-center gap-2 text-sm">
              <FileText className="w-4 h-4 shrink-0 text-[var(--text-secondary)]" />
              {file.source === 'attachment'
                ? <a className="min-w-0 flex-1 truncate underline" href={appPath(`/api/v1/jobs/attachments/${file.attachmentID}/download`)}>{file.originalFilename}</a>
                : <a className="min-w-0 flex-1 truncate underline" href={appPath(`/documents/${file.documentID}/download`)}>{file.originalFilename}</a>}
              <span className="text-xs text-[var(--text-muted)]">{file.fileSizeFormatted}</span>
              <button className="suite-button" disabled={busy} onClick={() => remove(file)} aria-label={`${file.originalFilename} entfernen`}><X className="w-4 h-4" /></button>
            </li>)}
          </ul>}
        {error && <div className="jobs-inline-alert mt-3" role="alert">{error} <button className="underline" onClick={load}>Erneut versuchen</button></div>}
      </div>
    </section>
    <section className="jobs-card">
      <div className="jobs-card-heading"><h2>Verlauf</h2></div>
      <div className="jobs-card-body">
        {loading ? <p className="jobs-form-note">Lädt…</p> : history.length === 0 ? <p className="jobs-form-note">Noch keine Änderungen protokolliert.</p> :
          <ol className="space-y-3">
            {history.slice(0, 8).map((entry) => <li key={entry.history_id} className="border-b border-[var(--border-divider)] pb-3 last:border-0 last:pb-0">
              <p className="text-sm">{entry.description}</p>
              <p className="mt-1 text-xs text-[var(--text-muted)]">{new Date(entry.changed_at).toLocaleString('de-DE')} · {entry.user_name || 'System'}</p>
            </li>)}
          </ol>}
      </div>
    </section>
  </>;
}
