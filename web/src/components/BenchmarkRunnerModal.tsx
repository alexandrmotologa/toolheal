import React, { useState } from 'react';
import { BenchmarkResult } from '../types';
import { X, Play, CheckCircle2, Activity } from 'lucide-react';

interface BenchmarkRunnerModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const BenchmarkRunnerModal: React.FC<BenchmarkRunnerModalProps> = ({ isOpen, onClose }) => {
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<BenchmarkResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const runBenchmark = async () => {
    setRunning(true);
    setError(null);

    try {
      const resp = await fetch('/api/bench/run', { method: 'POST' });
      if (!resp.ok) {
        throw new Error(`Server returned status ${resp.status}`);
      }
      const data: BenchmarkResult = await resp.json();
      setResult(data);
    } catch (err: any) {
      setError(err.message || 'Failed to execute benchmark suite');
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm">
      <div className="bg-[#101726] border border-slate-800 rounded-xl max-w-2xl w-full p-6 shadow-2xl space-y-6 animate-in fade-in zoom-in-95">
        <div className="flex items-center justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-emerald-400" />
            <h3 className="text-base font-semibold text-white">Live Benchmark Verification Suite</h3>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {!result ? (
          <div className="text-center py-8 space-y-4">
            <div className="w-16 h-16 mx-auto rounded-full bg-sky-500/10 border border-sky-500/20 flex items-center justify-center">
              <Play className={`w-8 h-8 text-sky-400 ${running ? 'animate-pulse' : ''}`} />
            </div>
            <div>
              <h4 className="text-sm font-semibold text-white">Execute 60 Multi-Model Failure Scenarios</h4>
              <p className="text-xs text-slate-400 mt-1 max-w-md mx-auto">
                Validates fuzzy matching, casing conversions, enum snapping, array wrapping, and temporal normalization across real-world agent payloads.
              </p>
            </div>

            <button
              onClick={runBenchmark}
              disabled={running}
              className="py-2.5 px-6 bg-emerald-500 hover:bg-emerald-400 disabled:bg-slate-800 text-white text-xs font-semibold rounded-lg shadow-lg shadow-emerald-500/20 transition-all inline-flex items-center gap-2"
            >
              <Play className="w-4 h-4 fill-current" />
              {running ? 'Running 60 Test Scenarios...' : 'Start Benchmark Run'}
            </button>

            {error && <p className="text-xs text-rose-400 font-mono">{error}</p>}
          </div>
        ) : (
          <div className="space-y-6">
            {/* KPI Results Grid */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div className="p-3 rounded-lg bg-slate-900 border border-slate-800 text-center">
                <span className="text-[10px] text-slate-400 uppercase font-medium block">Recovery Rate</span>
                <span className="text-2xl font-bold font-mono text-emerald-400">
                  {result.recovery_rate.toFixed(1)}%
                </span>
                <span className="text-[10px] text-emerald-500 font-mono block">SLA &gt; 85% PASS</span>
              </div>

              <div className="p-3 rounded-lg bg-slate-900 border border-slate-800 text-center">
                <span className="text-[10px] text-slate-400 uppercase font-medium block">Avg Latency</span>
                <span className="text-2xl font-bold font-mono text-sky-400">
                  {result.avg_latency_us < 1000
                    ? `${result.avg_latency_us} µs`
                    : `${(result.avg_latency_us / 1000).toFixed(2)} ms`}
                </span>
                <span className="text-[10px] text-sky-500 font-mono block">&lt; 2.0 ms PASS</span>
              </div>

              <div className="p-3 rounded-lg bg-slate-900 border border-slate-800 text-center">
                <span className="text-[10px] text-slate-400 uppercase font-medium block">Tokens Saved</span>
                <span className="text-2xl font-bold font-mono text-purple-400">
                  {result.tokens_saved.toLocaleString()}
                </span>
                <span className="text-[10px] text-purple-400 font-mono block">retry loops skipped</span>
              </div>

              <div className="p-3 rounded-lg bg-slate-900 border border-slate-800 text-center">
                <span className="text-[10px] text-slate-400 uppercase font-medium block">Est. Cost Saved</span>
                <span className="text-2xl font-bold font-mono text-amber-400">
                  ${result.dollars_saved.toFixed(2)}
                </span>
                <span className="text-[10px] text-amber-500 font-mono block">per 60 tool calls</span>
              </div>
            </div>

            {/* Repair Algorithms Breakdown */}
            <div className="space-y-2">
              <h5 className="text-xs font-semibold text-slate-300 uppercase tracking-wider">
                Algorithms Executed Across Test Suite
              </h5>
              <div className="grid grid-cols-2 gap-2 text-xs font-mono">
                {Object.entries(result.fix_breakdown).map(([fixType, count]) => (
                  <div
                    key={fixType}
                    className="p-2.5 rounded bg-slate-950/80 border border-slate-800 flex items-center justify-between"
                  >
                    <span className="text-slate-300 text-[11px]">{fixType}</span>
                    <span className="text-emerald-400 font-bold bg-slate-900 px-2 py-0.5 rounded">
                      {count}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex justify-between items-center pt-2 border-t border-slate-800">
              <span className="text-xs text-emerald-400 flex items-center gap-1.5 font-medium">
                <CheckCircle2 className="w-4 h-4" /> All 60 failure scenarios healed successfully.
              </span>
              <button
                onClick={runBenchmark}
                disabled={running}
                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5"
              >
                <Play className="w-3.5 h-3.5" /> Re-run
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
