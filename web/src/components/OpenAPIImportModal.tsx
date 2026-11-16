import React, { useState } from 'react';
import { X, Download, Globe, FileCode, CheckCircle2, AlertCircle } from 'lucide-react';

interface OpenAPIImportModalProps {
  isOpen: boolean;
  onClose: () => void;
  onImportSuccess: () => void;
}

export const OpenAPIImportModal: React.FC<OpenAPIImportModalProps> = ({
  isOpen,
  onClose,
  onImportSuccess,
}) => {
  const [tab, setTab] = useState<'url' | 'raw'>('url');
  const [url, setUrl] = useState('');
  const [rawSpec, setRawSpec] = useState('');
  const [loading, setLoading] = useState(false);
  const [resultMsg, setResultMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  if (!isOpen) return null;

  const handleImport = async () => {
    setLoading(true);
    setResultMsg(null);

    const payload = tab === 'url' ? { url: url.trim() } : { spec: rawSpec.trim() };

    try {
      const resp = await fetch('/api/tools/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      const data = await resp.json();
      if (!resp.ok) {
        setResultMsg({ type: 'error', text: data.error || 'Import failed' });
      } else {
        setResultMsg({
          type: 'success',
          text: `Successfully imported ${data.imported_count} tools! (Total catalog: ${data.total_tools})`,
        });
        onImportSuccess();
        setTimeout(() => {
          onClose();
        }, 1500);
      }
    } catch (err: any) {
      setResultMsg({ type: 'error', text: err.message || 'Network error during import' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
      <div className="bg-[#101726] border border-slate-800 rounded-xl max-w-xl w-full p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95">
        <div className="flex items-center justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-2">
            <Download className="w-5 h-5 text-sky-400" />
            <h3 className="text-base font-semibold text-white">Import OpenAPI / Swagger Tools</h3>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tab selection */}
        <div className="flex gap-2 p-1 bg-slate-900 rounded-lg border border-slate-800 text-xs font-medium">
          <button
            onClick={() => setTab('url')}
            className={`flex-1 py-1.5 rounded-md flex items-center justify-center gap-1.5 transition-colors ${
              tab === 'url' ? 'bg-sky-500 text-white' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Globe className="w-3.5 h-3.5" /> From URL
          </button>
          <button
            onClick={() => setTab('raw')}
            className={`flex-1 py-1.5 rounded-md flex items-center justify-center gap-1.5 transition-colors ${
              tab === 'raw' ? 'bg-sky-500 text-white' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <FileCode className="w-3.5 h-3.5" /> Paste JSON / YAML
          </button>
        </div>

        {tab === 'url' ? (
          <div className="space-y-2">
            <label className="text-xs text-slate-300 font-medium">OpenAPI Specification URL</label>
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://api.example.com/openapi.json"
              className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs font-mono text-slate-200 focus:outline-none focus:border-sky-500"
            />
            <p className="text-[11px] text-slate-500">
              Accepts public or local endpoints exposing OpenAPI 3.0, 3.1, or Swagger 2.0 specs.
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            <label className="text-xs text-slate-300 font-medium">OpenAPI JSON or YAML Payload</label>
            <textarea
              value={rawSpec}
              onChange={(e) => setRawSpec(e.target.value)}
              rows={8}
              placeholder="Paste raw OpenAPI 3.0 / 3.1 specification text..."
              className="w-full p-3 bg-slate-950 border border-slate-800 rounded-lg text-xs font-mono text-slate-200 focus:outline-none focus:border-sky-500 resize-none"
            />
          </div>
        )}

        {resultMsg && (
          <div
            className={`p-3 rounded-lg text-xs flex items-center gap-2 border ${
              resultMsg.type === 'success'
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                : 'bg-rose-500/10 text-rose-400 border-rose-500/30'
            }`}
          >
            {resultMsg.type === 'success' ? (
              <CheckCircle2 className="w-4 h-4 shrink-0" />
            ) : (
              <AlertCircle className="w-4 h-4 shrink-0" />
            )}
            <span>{resultMsg.text}</span>
          </div>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleImport}
            disabled={loading || (tab === 'url' ? !url.trim() : !rawSpec.trim())}
            className="px-4 py-2 bg-sky-500 hover:bg-sky-400 disabled:bg-slate-800 text-white text-xs font-medium rounded-lg shadow-md transition-colors flex items-center gap-2"
          >
            <Download className="w-3.5 h-3.5" />
            {loading ? 'Importing Tools...' : 'Import Tools'}
          </button>
        </div>
      </div>
    </div>
  );
};
