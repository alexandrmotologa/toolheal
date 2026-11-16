import React, { useState, useEffect } from 'react';
import { SuggestedPatch } from '../types';
import { Lightbulb, Check, ArrowRight, Zap, RefreshCw } from 'lucide-react';

interface SuggestedPatchesWidgetProps {
  onPatchApplied?: () => void;
}

export const SuggestedPatchesWidget: React.FC<SuggestedPatchesWidgetProps> = ({ onPatchApplied }) => {
  const [patches, setPatches] = useState<SuggestedPatch[]>([]);
  const [applying, setApplying] = useState<string | null>(null);
  const [appliedSet, setAppliedSet] = useState<Record<string, boolean>>({});

  const fetchPatches = async () => {
    try {
      const resp = await fetch('/api/patches/suggested');
      if (resp.ok) {
        const data = await resp.json();
        setPatches(data);
      }
    } catch (e) {
      console.warn('Failed to fetch suggested patches:', e);
    }
  };

  useEffect(() => {
    fetchPatches();
    const interval = setInterval(fetchPatches, 8000);
    return () => clearInterval(interval);
  }, []);

  const handleApply = async (patch: SuggestedPatch) => {
    const patchId = `${patch.tool_name}:${patch.original_key}`;
    setApplying(patchId);

    try {
      const resp = await fetch('/api/patches/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tool_name: patch.tool_name,
          canonical_key: patch.canonical_key,
          alias: patch.original_key,
        }),
      });

      if (resp.ok) {
        setAppliedSet((prev) => ({ ...prev, [patchId]: true }));
        if (onPatchApplied) onPatchApplied();
      }
    } catch (err) {
      console.error('Failed to apply patch:', err);
    } finally {
      setApplying(null);
    }
  };

  if (patches.length === 0) {
    return null;
  }

  return (
    <div className="bg-[#101726] border border-amber-500/30 rounded-xl p-5 shadow-lg space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Lightbulb className="w-5 h-5 text-amber-400" />
          <h3 className="text-sm font-semibold tracking-wide text-white uppercase">
            Auto-Learned Schema Patches ({patches.length})
          </h3>
        </div>
        <button
          onClick={fetchPatches}
          className="text-slate-400 hover:text-white text-xs flex items-center gap-1 font-mono"
        >
          <RefreshCw className="w-3 h-3" /> Refresh
        </button>
      </div>

      <p className="text-xs text-slate-400">
        Repeated hallucinations detected across agent runs. Click to add them as official aliases to eliminate fuzzy calculation overhead.
      </p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {patches.map((patch) => {
          const patchId = `${patch.tool_name}:${patch.original_key}`;
          const isApplied = appliedSet[patchId];

          return (
            <div
              key={patchId}
              className="p-3 rounded-lg bg-slate-900/80 border border-slate-800 flex items-center justify-between gap-3 text-xs font-mono"
            >
              <div className="space-y-1 truncate">
                <span className="text-[10px] text-slate-400 font-sans bg-slate-800 px-1.5 py-0.5 rounded">
                  {patch.tool_name}
                </span>
                <div className="flex items-center gap-1.5 truncate">
                  <span className="text-rose-400 font-semibold truncate">{patch.original_key}</span>
                  <ArrowRight className="w-3 h-3 text-slate-500 shrink-0" />
                  <span className="text-emerald-400 font-semibold truncate">{patch.canonical_key}</span>
                </div>
                <span className="text-[10px] text-slate-400 font-sans block">
                  {patch.hits} occurrences captured
                </span>
              </div>

              <button
                onClick={() => handleApply(patch)}
                disabled={isApplied || applying === patchId}
                className={`px-3 py-1.5 rounded-lg text-xs font-sans font-semibold shrink-0 transition-colors flex items-center gap-1 border ${
                  isApplied
                    ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40'
                    : 'bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border-amber-500/40'
                }`}
              >
                {isApplied ? (
                  <>
                    <Check className="w-3.5 h-3.5 text-emerald-400" /> Patched
                  </>
                ) : applying === patchId ? (
                  'Patching...'
                ) : (
                  <>
                    <Zap className="w-3.5 h-3.5" /> Add Alias
                  </>
                )}
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
};
