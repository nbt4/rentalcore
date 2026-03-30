import { useEffect, useState } from 'react';
import { FileText, Download, Trash2, RefreshCw } from 'lucide-react';
import { api } from '../lib/api';

interface Document {
  id: number;
  filename: string;
  original_filename: string;
  file_size: number;
  content_type: string;
  created_at: string;
  job_id?: number;
  description?: string;
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function DocumentsPage() {
  const [docs, setDocs] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    api.get<{ documents: Document[] }>('/documents')
      .then((r) => setDocs(r.data.documents || []))
      .catch((e) => setError(e.message || 'Fehler beim Laden'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const handleDelete = async (id: number) => {
    if (!confirm('Dokument löschen?')) return;
    await api.delete(`/documents/${id}`);
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><FileText className="w-6 h-6 text-accent-red" /> Dokumente</h1>
          <p className="text-gray-400 text-sm mt-1">Hochgeladene Dokumente und Dateien</p>
        </div>
        <button onClick={load} className="p-2 hover:bg-white/10 rounded-lg transition-colors text-gray-400">
          <RefreshCw className="w-4 h-4" />
        </button>
      </div>

      <div className="glass-dark rounded-xl border border-white/10">
        {loading ? (
          <div className="flex justify-center py-16"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>
        ) : error ? (
          <div className="p-8 text-center text-gray-400 text-sm">{error}</div>
        ) : docs.length === 0 ? (
          <div className="text-center py-16 text-gray-500">
            <FileText className="w-10 h-10 mx-auto mb-3 opacity-30" />
            <p>Keine Dokumente vorhanden</p>
          </div>
        ) : (
          <div className="divide-y divide-white/5">
            {docs.map((doc) => (
              <div key={doc.id} className="flex items-center justify-between px-6 py-4 hover:bg-white/5 transition-colors">
                <div className="flex items-center gap-3">
                  <FileText className="w-5 h-5 text-accent-red flex-shrink-0" />
                  <div>
                    <div className="font-medium text-white text-sm">{doc.original_filename || doc.filename}</div>
                    <div className="text-xs text-gray-400 mt-0.5">
                      {formatSize(doc.file_size)} · {new Date(doc.created_at).toLocaleDateString('de-DE')}
                      {doc.description && ` · ${doc.description}`}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <a
                    href={`/api/v1/documents/${doc.id}`}
                    download
                    className="p-2 hover:bg-white/10 rounded-lg transition-colors text-gray-400 hover:text-white"
                  >
                    <Download className="w-4 h-4" />
                  </a>
                  <button
                    onClick={() => handleDelete(doc.id)}
                    className="p-2 hover:bg-red-500/10 rounded-lg transition-colors text-gray-400 hover:text-red-400"
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
