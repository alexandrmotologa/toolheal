import React, { useState } from 'react';
import { ToolDefinition, ParameterDiff as IParameterDiff } from '../types';
import { ParameterDiff } from './ParameterDiff';
import { Play, Sparkles, Clock, CheckCircle2, AlertTriangle, Code2 } from 'lucide-react';

interface ToolSandboxProps {
  tools: ToolDefinition[];
  onHealComplete?: () => void;
}

export const ToolSandbox: React.FC<ToolSandboxProps> = ({ tools, onHealComplete }) => {
  const [selectedTool, setSelectedTool] = useState<string>(tools[0]?.name || 'create_user');
  const [inputPayload, setInputPayload] = useState<string>(
    JSON.stringify(
      {
        user_id: 'usr_8829',
        email: 'engineer@deepmind.ai',
        role: 'admin',
        is_active: 'true',
        created_at: '15/11/2026',
        tags: 'ai, agents, tooling',
      },
      null,
      2
    )
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [healedResult, setHealedResult] = useState<any | null>(null);
  const [diffs, setDiffs] = useState<IParameterDiff[]>([]);
  const [latencyUs, setLatencyUs] = useState<number | null>(null);

  const presets: Record<string, any> = {
    'create_user': {
      user_id: 'usr_991',
      email: 'alex@agent.com',
      role: 'administrator',
      active: '1',
      created_at: 'today',
      tags: 'backend',
    },
    'process_payment': {
      customer: 'cus_5511',
      amount: '7500',
      currency: 'usd',
      capture: 'true',
      memo: 'API Gateway License',
    },
    'search_database': {
      q: 'schema reconciliation',
      limit: '50',
      offset: 2,
      status: 'active',
      since: 'yesterday',
    },
    'create_github_issue': {
      repo: 'alexandrmotologa/toolheal',
      title: 'Fix parameter casing mismatch',
      labels: 'bug',
      priority: 'urgent',
    },
  };

  const loadPreset = (toolName: string) => {
    setSelectedTool(toolName);
    if (presets[toolName]) {
      setInputPayload(JSON.stringify(presets[toolName], null, 2));
    }
  };

  const handleHeal = async () => {
    setLoading(true);
    setError(null);
    setHealedResult(null);

    let parsedArgs: any;
    try {
      parsedArgs = JSON.parse(inputPayload);
    } catch (e: any) {
      setError(`Invalid JSON syntax: ${e.message}`);
      setLoading(false);
      return;
    }

    try {
      const resp = await fetch('/api/heal', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tool: selectedTool,
          arguments: parsedArgs,
        }),
      });

      const data = await resp.json();
      if (!resp.ok && !data.arguments) {
        setError(data.error || 'Failed to heal tool call');
      } else {
        setHealedResult(data.arguments || data.result);
        setDiffs(data.diffs || []);
        setLatencyUs(data.latency_us || 0);
        if (onHealComplete) onHealComplete();
      }
    } catch (err: any) {
      setError(err.message || 'Network error reaching toolheal daemon');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-[#101726] border border-slate-800 rounded-xl p-6 shadow-xl space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-4">
        <div>
          <h2 className="text-base font-semibold text-white flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-sky-400" />
            Interactive Healing Sandbox
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Test parameter reconciliation against registered schemas with sub-millisecond execution.
          </p>
        </div>

        {/* Preset quick buttons */}
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-slate-400 font-medium">Presets:</span>
          {Object.keys(presets).map((name) => (
            <button
              key={name}
              onClick={() => loadPreset(name)}
              className={`px-2.5 py-1 text-xs rounded-md font-mono transition-colors border ${
                selectedTool === name
                  ? 'bg-sky-500/20 text-sky-300 border-sky-500/40'
                  : 'bg-slate-900 text-slate-400 border-slate-800 hover:text-slate-200'
              }`}
            >
              {name}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Input Pane */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <label className="text-xs font-semibold uppercase tracking-wider text-slate-400">
              Target Tool Schema
            </label>
            <select
              value={selectedTool}
              onChange={(e) => setSelectedTool(e.target.value)}
              className="bg-slate-900 border border-slate-800 rounded px-2.5 py-1 text-xs font-mono text-slate-200 focus:outline-none focus:border-sky-500"
            >
              {tools.map((t) => (
                <option key={t.name} value={t.name}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>

          <div className="relative">
            <textarea
              value={inputPayload}
              onChange={(e) => setInputPayload(e.target.value)}
              rows={12}
              className="w-full bg-slate-950 font-mono text-xs text-slate-200 p-4 rounded-lg border border-slate-800 focus:outline-none focus:border-sky-500 leading-relaxed resize-none"
              placeholder="Paste raw or hallucinated JSON tool arguments here..."
            />
          </div>

          <button
            onClick={handleHeal}
            disabled={loading}
            className="w-full py-2.5 px-4 bg-sky-500 hover:bg-sky-400 disabled:bg-slate-800 text-white font-medium text-xs rounded-lg shadow-md transition-all flex items-center justify-center gap-2"
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            {loading ? 'Reconciling Parameters...' : 'Heal Parameters (Zero Token LLM Bypass)'}
          </button>

          {error && (
            <div className="p-3 bg-rose-500/10 border border-rose-500/30 rounded-lg text-rose-400 text-xs flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>

        {/* Output & Diffs Pane */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
              <Code2 className="w-4 h-4 text-emerald-400" /> Reconciled Output
            </span>
            {latencyUs !== null && (
              <span className="text-xs font-mono px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {latencyUs < 1000 ? `${latencyUs} µs` : `${(latencyUs / 1000).toFixed(2)} ms`}
              </span>
            )}
          </div>

          {healedResult ? (
            <div className="space-y-4">
              <div className="relative">
                <pre className="bg-slate-950 font-mono text-xs text-emerald-400 p-4 rounded-lg border border-slate-800 overflow-x-auto max-h-[160px] leading-relaxed">
                  {JSON.stringify(healedResult, null, 2)}
                </pre>
              </div>

              {/* Diffs Breakdown */}
              <ParameterDiff diffs={diffs} />
            </div>
          ) : (
            <div className="h-[280px] border border-dashed border-slate-800 rounded-lg flex flex-col items-center justify-center text-slate-500 text-xs p-6 text-center">
              <CheckCircle2 className="w-8 h-8 mb-2 stroke-slate-700" />
              <span>Click "Heal Parameters" to run the fuzzy reconciliation pipeline.</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
