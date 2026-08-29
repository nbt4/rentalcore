import { useEffect, useRef, useState } from 'react';
import { FileText, Download, Trash2, RefreshCw, Upload, Eye, X, Check } from 'lucide-react';
import { api } from '../lib/api';
import { toast } from '../lib/toast';
import { appPath } from '../lib/app-paths';

interface Document {
  documentID: number;
  filename: string;
  originalFilename: string;
  fileSize: number;
  mimeType: string;
  documentType: string;
  entityType: string;
  entityID: string;
  description: string;
  uploadedAt: string;
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function mimeIcon(mime: string) {
  if (mime.startsWith('image/')) return '🖼️';
  if (mime === 'application/pdf') return '📄';
  if (mime.includes('word') || mime.includes('document')) return '📝';
  if (mime.includes('sheet') || mime.includes('excel')) return '📊';
  return '📎';
}

const DOC_TYPES = ['contract', 'manual', 'photo', 'invoice', 'receipt', 'signature', 'other'] as const;
const DOC_TYPE_LABELS: Record<string, string> = {
  contract: 'Vertrag', manual: 'Handbuch', photo: 'Foto',
  invoice: 'Rechnung', receipt: 'Quittung', signature: 'Unterschrift', other: 'Sonstiges',
};

function UploadModal({ onClose, onUploaded }: { onClose: () => void; onUploaded: () => void }) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [docType, setDocType] = useState<string>('other');
  const [description, setDescription] = useState('');
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const file = fileRef.current?.files?.[0];
    if (!file) { setError('Keine Datei ausgewählt'); return; }
    setError('');
    setUploading(true);
    try {
      const fd = new FormData();
      fd.append('file', file);
      fd.append('entityType', 'system');
      fd.append('entityID', '0');
      fd.append('documentType', docType);
      fd.append('description', description);
      await fetch(appPath('/documents/upload'), {
        method: 'POST',
        credentials: 'include',
        body: fd,
      }).then(async (r) => {
        if (!r.ok) {
          const j = await r.json().catch(() => ({}));
          throw new Error(j.error || 'Upload fehlgeschlagen');
        }
      });
      onUploaded();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload fehlgeschlagen');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="glass-dark rounded-xl border border-white/10 w-full max-w-md p-6">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold flex items-center gap-2"><Upload className="w-5 h-5 text-accent-red" />Dokument hochladen</h2>
          <button onClick={onClose} className="p-1.5 hover:bg-white/10 rounded-lg transition-colors"><X className="w-4 h-4" /></button>
        </div>
        {error && <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">{error}</div>}
        <form onSubmit={submit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Datei *</label>
            <input ref={fileRef} type="file" required className="w-full text-sm text-gray-300 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:bg-accent-red/20 file:text-accent-red file:text-sm file:cursor-pointer hover:file:bg-accent-red/30" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Dokumenttyp</label>
            <select value={docType} onChange={(e) => setDocType(e.target.value)} className="w-full px-3 py-2 rounded-lg text-sm">
              {DOC_TYPES.map((t) => <option key={t} value={t}>{DOC_TYPE_LABELS[t]}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Beschreibung</label>
            <input type="text" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Optional" className="w-full px-3 py-2 rounded-lg text-sm" />
          </div>
          <div className="flex gap-3 pt-1">
            <button type="submit" disabled={uploading} className="flex items-center gap-2 px-5 py-2 bg-accent-red hover:bg-accent-red/80 disabled:opacity-50 text-white rounded-lg text-sm font-medium transition-colors">
              <Check className="w-4 h-4" />{uploading ? 'Wird hochgeladen...' : 'Hochladen'}
            </button>
            <button type="button" onClick={onClose} className="px-4 py-2 bg-white/10 hover:bg-white/15 rounded-lg text-sm transition-colors">Abbrechen</button>
          </div>
        </form>
      </div>
    </div>
  );
}

export function DocumentsPage() {
  const [docs, setDocs] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showUpload, setShowUpload] = useState(false);

  const load = () => {
    setLoading(true);
    setError('');
    api.get<{ documents: Document[] }>('/documents')
      .then((r) => setDocs(r.data.documents || []))
      .catch((e) => setError(e.message || 'Fehler beim Laden'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const handleDelete = async (id: number) => {
    if (!confirm('Dokument wirklich löschen?')) return;
    await api.delete(`/documents/${id}`).catch((e: any) => toast.error(e));
    load();
  };

  return (
    <div className="space-y-6">
      {showUpload && <UploadModal onClose={() => setShowUpload(false)} onUploaded={load} />}

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><FileText className="w-6 h-6 text-accent-red" /> Dokumente</h1>
          <p className="text-gray-400 text-sm mt-1">Hochgeladene Dokumente und Dateien</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={load} className="p-2 hover:bg-white/10 rounded-lg transition-colors text-gray-400">
            <RefreshCw className="w-4 h-4" />
          </button>
          <button onClick={() => setShowUpload(true)} className="flex items-center gap-2 px-4 py-2.5 bg-accent-red hover:bg-accent-red/80 text-white rounded-lg text-sm font-medium transition-colors">
            <Upload className="w-4 h-4" /> Hochladen
          </button>
        </div>
      </div>

      <div className="glass-dark rounded-xl border border-white/10">
        {loading ? (
          <div className="flex justify-center py-16"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>
        ) : error ? (
          <div className="p-8 text-center text-red-400 text-sm">{error}</div>
        ) : docs.length === 0 ? (
          <div className="text-center py-16 text-gray-500">
            <FileText className="w-10 h-10 mx-auto mb-3 opacity-30" />
            <p>Keine Dokumente vorhanden</p>
            <button onClick={() => setShowUpload(true)} className="mt-4 text-accent-red text-sm hover:underline">Erstes Dokument hochladen</button>
          </div>
        ) : (
          <div className="divide-y divide-white/5">
            {docs.map((doc) => (
              <div key={doc.documentID} className="flex items-center justify-between px-6 py-4 hover:bg-white/5 transition-colors">
                <div className="flex items-center gap-3 min-w-0">
                  <span className="text-xl flex-shrink-0">{mimeIcon(doc.mimeType)}</span>
                  <div className="min-w-0">
                    <div className="font-medium text-white text-sm truncate">{doc.originalFilename || doc.filename}</div>
                    <div className="text-xs text-gray-400 mt-0.5 flex items-center gap-2 flex-wrap">
                      <span>{formatSize(doc.fileSize)}</span>
                      <span>·</span>
                      <span className="capitalize">{doc.documentType}</span>
                      <span>·</span>
                      <span className="capitalize">{doc.entityType} #{doc.entityID}</span>
                      <span>·</span>
                      <span>{new Date(doc.uploadedAt).toLocaleDateString('de-DE')}</span>
                      {doc.description && <><span>·</span><span className="italic">{doc.description}</span></>}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-1 flex-shrink-0 ml-4">
                  <a
                    href={appPath(`/documents/${doc.documentID}/download`)}
                    target="_blank"
                    rel="noreferrer"
                    className="p-2 hover:bg-white/10 rounded-lg transition-colors text-gray-400 hover:text-white"
                    title="Ansehen"
                  >
                    <Eye className="w-4 h-4" />
                  </a>
                  <a
                    href={appPath(`/documents/${doc.documentID}/download`)}
                    download
                    className="p-2 hover:bg-white/10 rounded-lg transition-colors text-gray-400 hover:text-white"
                    title="Herunterladen"
                  >
                    <Download className="w-4 h-4" />
                  </a>
                  <button
                    onClick={() => handleDelete(doc.documentID)}
                    className="p-2 hover:bg-red-500/10 rounded-lg transition-colors text-gray-400 hover:text-red-400"
                    title="Löschen"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
