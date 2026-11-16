import React from 'react';
import { DriftItem } from '../types';
import { Flame, ArrowRight, BarChart3 } from 'lucide-react';

interface DriftHeatmapProps {
  drifts: DriftItem[];
}

export const DriftHeatmap: React.FC<DriftHeatmapProps> = ({ drifts }) => {
  const maxOccurrences = drifts.length > 0 ? Math.max(...drifts.map((d) => d.occurrences)) : 1;

  if (!drifts || drifts.length === 0) {
    return (
      <div className="bg-[#101726] border border-slate-800 rounded-xl p-6 text-center text-slate-400">
        <Flame className="w-8 h-8 mx-auto mb-2 text-slate-600" />
        <p className="text-sm">No schema drift recorded yet. Intercept tool calls to build heatmap.</p>
      </div>
    );
  }

  return (
    <div className="bg-[#101726] border border-slate-800 rounded-xl p-5 shadow-lg">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Flame className="w-5 h-5 text-amber-500" />
          <h3 className="text-sm font-semibold tracking-wide text-slate-300 uppercase">
            Parameter Drift Heatmap
          </h3>
        </div>
        <span className="text-xs text-slate-400 flex items-center gap-1 font-mono">
          <BarChart3 className="w-3.5 h-3.5 text-sky-400" /> Top Drifting Keys
        </span>
      </div>

      <div className="space-y-3">
        {drifts.slice(0, 8).map((drift, idx) => {
          const widthPercent = Math.min(100, Math.max(10, (drift.occurrences / maxOccurrences) * 100));

          return (
            <div key={idx} className="space-y-1">
              <div className="flex items-center justify-between text-xs font-mono">
                <div className="flex items-center gap-1.5 truncate">
                  <span className="text-slate-400 font-sans text-[11px] bg-slate-800/80 px-1.5 py-0.5 rounded">
                    {drift.tool_name}
                  </span>
                  <span className="text-rose-400 truncate max-w-[120px]">{drift.original_key}</span>
                  <ArrowRight className="w-3 h-3 text-slate-500 shrink-0" />
                  <span className="text-emerald-400 truncate max-w-[120px]">{drift.healed_key}</span>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-[10px] text-slate-400 uppercase font-sans">
                    {drift.fix_type}
                  </span>
                  <span className="text-slate-200 font-bold bg-slate-800 px-1.5 py-0.5 rounded text-[11px]">
                    {drift.occurrences}x
                  </span>
                </div>
              </div>

              {/* Intensity Bar */}
              <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-amber-500 via-orange-500 to-rose-500 rounded-full transition-all duration-500"
                  style={{ width: `${widthPercent}%` }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
