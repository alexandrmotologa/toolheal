import React from 'react';
import { ParameterDiff as IParameterDiff } from '../types';
import { ArrowRight, CheckCircle2, Sliders } from 'lucide-react';

interface ParameterDiffProps {
  diffs: IParameterDiff[];
}

export const ParameterDiff: React.FC<ParameterDiffProps> = ({ diffs }) => {
  const getBadgeStyle = (fixType: string) => {
    switch (fixType) {
      case 'KEY_ALIAS':
        return 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30';
      case 'KEY_CASE':
        return 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30';
      case 'KEY_FUZZY':
        return 'bg-violet-500/15 text-violet-400 border-violet-500/30';
      case 'TYPE_COERCE':
        return 'bg-sky-500/15 text-sky-400 border-sky-500/30';
      case 'ENUM_SNAP':
        return 'bg-amber-500/15 text-amber-400 border-amber-500/30';
      case 'DATE_NORMALIZE':
        return 'bg-purple-500/15 text-purple-400 border-purple-500/30';
      case 'DEFAULT_INJECT':
        return 'bg-slate-500/15 text-slate-400 border-slate-500/30';
      case 'ENVELOPE_UNWRAP':
        return 'bg-teal-500/15 text-teal-400 border-teal-500/30';
      default:
        return 'bg-slate-700 text-slate-300 border-slate-600';
    }
  };

  const formatValue = (v: any) => {
    if (v === null || v === undefined) return <span className="text-slate-500 italic">null</span>;
    if (typeof v === 'boolean') return <span className="text-pink-400">{v.toString()}</span>;
    if (typeof v === 'number') return <span className="text-amber-300">{v}</span>;
    if (typeof v === 'string') return <span className="text-emerald-300">"{v}"</span>;
    if (Array.isArray(v)) return <span className="text-indigo-300">[{v.map(formatValue).reduce((prev, curr) => [prev, ', ', curr] as any)}]</span>;
    return <span className="text-slate-300">{JSON.stringify(v)}</span>;
  };

  if (!diffs || diffs.length === 0) {
    return (
      <div className="flex items-center gap-2 p-3 text-xs text-slate-400 bg-slate-900/40 rounded-lg border border-slate-800">
        <CheckCircle2 className="w-4 h-4 text-emerald-500" />
        <span>Parameters matched official schema perfectly. No healing required.</span>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
        <span className="flex items-center gap-1.5">
          <Sliders className="w-3.5 h-3.5 text-sky-400" /> Applied Parameter Fixes ({diffs.length})
        </span>
      </div>

      <div className="divide-y divide-slate-800/80 border border-slate-800 rounded-lg overflow-hidden bg-slate-900/50">
        {diffs.map((diff, index) => (
          <div key={index} className="p-3 text-xs font-mono hover:bg-slate-800/30 transition-colors">
            <div className="flex items-center justify-between gap-2 mb-1.5">
              <span className={`px-2 py-0.5 rounded text-[10px] font-semibold tracking-wide border ${getBadgeStyle(diff.fix_type)}`}>
                {diff.fix_type}
              </span>
              <span className="text-[11px] text-slate-400">
                Confidence: <span className="text-slate-200">{(diff.confidence * 100).toFixed(0)}%</span>
              </span>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-[1fr,auto,1fr] items-center gap-2 bg-slate-950/60 p-2.5 rounded border border-slate-800/60">
              {/* Original */}
              <div className="truncate">
                <span className="text-rose-400/90 font-semibold">{diff.original_key}</span>
                <span className="text-slate-500 mx-1">:</span>
                <span className="text-slate-400">{formatValue(diff.original_value)}</span>
              </div>

              {/* Arrow */}
              <div className="flex justify-center text-slate-500">
                <ArrowRight className="w-3.5 h-3.5 text-sky-400 hidden md:block" />
              </div>

              {/* Healed */}
              <div className="truncate">
                <span className="text-emerald-400 font-semibold">{diff.healed_key}</span>
                <span className="text-slate-500 mx-1">:</span>
                <span className="text-slate-200">{formatValue(diff.healed_value)}</span>
              </div>
            </div>

            <p className="mt-1.5 text-[11px] text-slate-400 font-sans">
              {diff.description}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
};
