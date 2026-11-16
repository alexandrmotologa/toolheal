import React, { useState } from 'react';
import { InvocationRecord } from '../types';
import { ParameterDiff } from './ParameterDiff';
import { Activity, Clock, ShieldCheck, ChevronDown, ChevronUp, Radio } from 'lucide-react';

interface LiveInvocationLogProps {
  invocations: InvocationRecord[];
  isStreaming: boolean;
}

export const LiveInvocationLog: React.FC<LiveInvocationLogProps> = ({
  invocations,
  isStreaming,
}) => {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const toggleExpand = (id: string) => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  const filteredInvocations = invocations.filter((inv) => {
    if (filterStatus === 'all') return true;
    if (filterStatus === 'healed') return inv.healed;
    if (filterStatus === 'fallback') return inv.status === 'fallback';
    return true;
  });

  return (
    <div className="bg-[#101726] border border-slate-800 rounded-xl p-5 shadow-lg space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-800 pb-3">
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5 text-sky-400" />
          <h3 className="text-sm font-semibold tracking-wide text-slate-300 uppercase">
            Live Tool Interception Stream
          </h3>
          <span
            className={`flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-full font-mono font-medium border ${
              isStreaming
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                : 'bg-slate-800 text-slate-400 border-slate-700'
            }`}
          >
            <Radio className={`w-3 h-3 ${isStreaming ? 'animate-pulse text-emerald-400' : ''}`} />
            {isStreaming ? 'LIVE SSE' : 'DISCONNECTED'}
          </span>
        </div>

        {/* Filter buttons */}
        <div className="flex items-center gap-1.5 text-xs">
          {['all', 'healed', 'fallback'].map((s) => (
            <button
              key={s}
              onClick={() => setFilterStatus(s)}
              className={`px-2.5 py-1 rounded font-medium capitalize transition-colors ${
                filterStatus === s
                  ? 'bg-sky-500 text-white shadow-sm'
                  : 'bg-slate-900 text-slate-400 hover:text-slate-200 border border-slate-800'
              }`}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {filteredInvocations.length === 0 ? (
        <div className="text-center py-12 text-slate-500 text-xs">
          No invocations matching filter yet. Trigger a tool call or run benchmarks to stream data.
        </div>
      ) : (
        <div className="space-y-2">
          {filteredInvocations.map((inv) => {
            const isExpanded = expandedId === inv.id;

            return (
              <div
                key={inv.id}
                className="border border-slate-800 rounded-lg bg-slate-900/60 overflow-hidden transition-all hover:border-slate-700"
              >
                <div
                  onClick={() => toggleExpand(inv.id)}
                  className="p-3.5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 cursor-pointer select-none"
                >
                  <div className="flex items-center gap-3">
                    <span
                      className={`px-2 py-0.5 text-[10px] font-mono font-semibold uppercase rounded border ${
                        inv.status === 'healed_success'
                          ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                          : inv.status === 'fallback'
                          ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                          : 'bg-slate-800 text-slate-400 border-slate-700'
                      }`}
                    >
                      {inv.status}
                    </span>

                    <span className="font-mono text-xs font-semibold text-white">
                      {inv.tool_name}
                    </span>

                    {inv.healed && (
                      <span className="text-[11px] text-emerald-400 flex items-center gap-1 font-mono">
                        <ShieldCheck className="w-3.5 h-3.5" />
                        +{inv.tokens_saved.toLocaleString()} tokens
                      </span>
                    )}
                  </div>

                  <div className="flex items-center gap-4 text-xs font-mono text-slate-400">
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3 text-slate-500" />
                      {inv.latency_us < 1000
                        ? `${inv.latency_us} µs`
                        : `${(inv.latency_us / 1000).toFixed(2)} ms`}
                    </span>

                    <span className="text-slate-500 text-[11px]">
                      {new Date(inv.created_at).toLocaleTimeString()}
                    </span>

                    {isExpanded ? (
                      <ChevronUp className="w-4 h-4 text-slate-400" />
                    ) : (
                      <ChevronDown className="w-4 h-4 text-slate-400" />
                    )}
                  </div>
                </div>

                {/* Expanded Inspection */}
                {isExpanded && (
                  <div className="p-4 border-t border-slate-800/80 bg-slate-950/80 space-y-4">
                    <ParameterDiff diffs={inv.diffs} />

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs font-mono">
                      <div>
                        <span className="text-[11px] uppercase tracking-wider text-rose-400 block mb-1">
                          Original Hallucinated Payload
                        </span>
                        <pre className="p-3 bg-slate-900 border border-slate-800 rounded text-slate-300 max-h-[140px] overflow-y-auto">
                          {JSON.stringify(inv.original_args, null, 2)}
                        </pre>
                      </div>

                      <div>
                        <span className="text-[11px] uppercase tracking-wider text-emerald-400 block mb-1">
                          Healed Sanitized Arguments
                        </span>
                        <pre className="p-3 bg-slate-900 border border-slate-800 rounded text-emerald-300 max-h-[140px] overflow-y-auto">
                          {JSON.stringify(inv.healed_args, null, 2)}
                        </pre>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
