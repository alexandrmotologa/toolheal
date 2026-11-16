import React, { useState, useEffect } from 'react';
import {
  TelemetryMetrics,
  InvocationRecord,
  DriftItem,
  ToolDefinition,
} from '../types';
import { HealingGauge } from '../components/HealingGauge';
import { DriftHeatmap } from '../components/DriftHeatmap';
import { LiveInvocationLog } from '../components/LiveInvocationLog';
import { ToolSandbox } from '../components/ToolSandbox';
import { ToolRegistryView } from '../components/ToolRegistryView';
import { SuggestedPatchesWidget } from '../components/SuggestedPatchesWidget';
import { BenchmarkRunnerModal } from '../components/BenchmarkRunnerModal';
import {
  Zap,
  Activity,
  Layers,
  Sparkles,
  BookOpen,
  DollarSign,
  Clock,
  RefreshCw,
  Play,
  Lightbulb,
} from 'lucide-react';

export const Dashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'stream' | 'sandbox' | 'drifts' | 'catalog'>('stream');
  const [metrics, setMetrics] = useState<TelemetryMetrics>({
    total_invocations: 0,
    healed_invocations: 0,
    recovery_rate: 100,
    tokens_saved: 0,
    average_latency_us: 17,
  });
  const [invocations, setInvocations] = useState<InvocationRecord[]>([]);
  const [drifts, setDrifts] = useState<DriftItem[]>([]);
  const [tools, setTools] = useState<ToolDefinition[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [isBenchmarkModalOpen, setIsBenchmarkModalOpen] = useState(false);

  // Fetch initial telemetry and catalog
  const fetchData = async () => {
    try {
      const [mRes, iRes, dRes, tRes] = await Promise.all([
        fetch('/api/metrics'),
        fetch('/api/invocations?limit=50'),
        fetch('/api/drifts'),
        fetch('/api/tools'),
      ]);

      if (mRes.ok) setMetrics(await mRes.json());
      if (iRes.ok) setInvocations(await iRes.json());
      if (dRes.ok) setDrifts(await dRes.json());
      if (tRes.ok) setTools(await tRes.json());
    } catch (e) {
      console.warn('Initial data fetch error:', e);
    }
  };

  useEffect(() => {
    fetchData();

    // Setup SSE connection
    let eventSource: EventSource | null = null;
    try {
      eventSource = new EventSource('/api/stream');

      eventSource.onopen = () => {
        setIsStreaming(true);
      };

      eventSource.addEventListener('invocation', (event) => {
        try {
          const newRec: InvocationRecord = JSON.parse(event.data);
          setInvocations((prev) => [newRec, ...prev.slice(0, 49)]);

          // Update metrics optimistically
          setMetrics((prev) => {
            const newTotal = prev.total_invocations + 1;
            const newHealed = newRec.healed ? prev.healed_invocations + 1 : prev.healed_invocations;
            return {
              total_invocations: newTotal,
              healed_invocations: newHealed,
              recovery_rate: Number(((newHealed / newTotal) * 100).toFixed(1)),
              tokens_saved: prev.tokens_saved + newRec.tokens_saved,
              average_latency_us: Math.round((prev.average_latency_us + newRec.latency_us) / 2),
            };
          });
        } catch (err) {
          console.error('Failed to parse SSE invocation:', err);
        }
      });

      eventSource.onerror = () => {
        setIsStreaming(false);
      };
    } catch (err) {
      console.warn('SSE unsupported or network blocked:', err);
    }

    // Polling backup every 10 seconds
    const interval = setInterval(fetchData, 10000);

    return () => {
      clearInterval(interval);
      if (eventSource) eventSource.close();
    };
  }, []);

  // ROI calculations
  const dollarsSaved = (metrics.tokens_saved * 0.000003).toFixed(2);
  const secondsSaved = Math.round((metrics.tokens_saved / 2400) * 4.2);
  const minutesSaved = (secondsSaved / 60).toFixed(1);

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      {/* Top Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-800 pb-6">
        <div>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-sky-500 to-emerald-400 flex items-center justify-center shadow-lg shadow-sky-500/20">
              <Zap className="w-6 h-6 text-slate-950 fill-current" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight text-white flex items-center gap-2">
                toolheal
                <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-slate-800 text-sky-400 border border-slate-700">
                  v1.0.0
                </span>
              </h1>
              <p className="text-xs text-slate-400">
                Autonomous Self-Healing Tool Calling &amp; Schema Reconciliation Middleware
              </p>
            </div>
          </div>
        </div>

        {/* Global Controls & Status */}
        <div className="flex items-center gap-3 flex-wrap">
          <button
            onClick={() => setIsBenchmarkModalOpen(true)}
            className="px-3 py-1.5 bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-500/40 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
          >
            <Play className="w-3.5 h-3.5 fill-current" /> Run Benchmark Suite
          </button>

          <button
            onClick={fetchData}
            className="p-2 text-slate-400 hover:text-white bg-slate-900 border border-slate-800 rounded-lg hover:border-slate-700 transition-colors"
            title="Refresh Metrics"
          >
            <RefreshCw className="w-4 h-4" />
          </button>

          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span className="text-slate-300">Daemon Active (:7171)</span>
          </div>
        </div>
      </div>

      {/* KPI Metric Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Metric 1: Recovery Rate */}
        <div className="bg-[#101726] border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
          <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
            <span>AUTO-RECOVERY RATE</span>
            <Zap className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-3xl font-extrabold font-mono text-emerald-400">
              {metrics.recovery_rate.toFixed(1)}%
            </span>
            <span className="text-[11px] text-slate-400 font-mono">SLA &gt; 85%</span>
          </div>
          <p className="text-[11px] text-slate-500 mt-1">
            {metrics.healed_invocations} of {metrics.total_invocations} calls repaired
          </p>
        </div>

        {/* Metric 2: Financial ROI & Tokens Saved */}
        <div className="bg-[#101726] border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
          <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
            <span>FINANCIAL ROI SAVINGS</span>
            <DollarSign className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-3xl font-extrabold font-mono text-emerald-400">
              ${dollarsSaved}
            </span>
            <span className="text-[11px] text-slate-400 font-mono">
              ~{minutesSaved}m wait avoided
            </span>
          </div>
          <p className="text-[11px] text-slate-500 mt-1 font-mono">
            {metrics.tokens_saved.toLocaleString()} prompt tokens saved
          </p>
        </div>

        {/* Metric 3: Execution Overhead */}
        <div className="bg-[#101726] border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
          <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
            <span>AVG RECONCILIATION TIME</span>
            <Clock className="w-4 h-4 text-sky-400" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-3xl font-extrabold font-mono text-sky-400">
              {metrics.average_latency_us < 1000
                ? `${metrics.average_latency_us} µs`
                : `${(metrics.average_latency_us / 1000).toFixed(2)} ms`}
            </span>
            <span className="text-[11px] text-slate-400 font-mono">&lt; 2.0 ms</span>
          </div>
          <p className="text-[11px] text-slate-500 mt-1">O(1) fast-cache + fuzzy Levenshtein</p>
        </div>

        {/* Metric 4: Registered Tools */}
        <div className="bg-[#101726] border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
          <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
            <span>SCHEMA CATALOG</span>
            <Layers className="w-4 h-4 text-amber-400" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-3xl font-extrabold font-mono text-amber-400">
              {tools.length}
            </span>
            <span className="text-xs text-slate-400">active tools</span>
          </div>
          <p className="text-[11px] text-slate-500 mt-1">OpenAPI 3.1 &amp; MCP support</p>
        </div>
      </div>

      {/* Suggested Schema Patches Notification Widget */}
      <SuggestedPatchesWidget onPatchApplied={fetchData} />

      {/* Navigation Tabs */}
      <div className="flex items-center gap-2 border-b border-slate-800 pb-1">
        {[
          { id: 'stream', label: 'Live Telemetry & Invocations', icon: Activity },
          { id: 'sandbox', label: 'Interactive Sandbox', icon: Sparkles },
          { id: 'drifts', label: 'Schema Drift Analytics', icon: Lightbulb },
          { id: 'catalog', label: 'Tool Schema Catalog', icon: BookOpen },
        ].map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;

          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-xs font-semibold tracking-wide transition-all ${
                isActive
                  ? 'bg-sky-500 text-white shadow-md shadow-sky-500/20'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'
              }`}
            >
              <Icon className="w-4 h-4" />
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* Tab 1: Live Telemetry & Stream */}
      {activeTab === 'stream' && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-1">
              <HealingGauge
                percentage={metrics.recovery_rate}
                totalHealed={metrics.healed_invocations}
                totalInvocations={metrics.total_invocations}
              />
            </div>
            <div className="lg:col-span-2">
              <DriftHeatmap drifts={drifts} />
            </div>
          </div>

          <LiveInvocationLog invocations={invocations} isStreaming={isStreaming} />
        </div>
      )}

      {/* Tab 2: Interactive Sandbox */}
      {activeTab === 'sandbox' && (
        <ToolSandbox tools={tools} onHealComplete={fetchData} />
      )}

      {/* Tab 3: Schema Drift Analytics */}
      {activeTab === 'drifts' && (
        <div className="space-y-6">
          <DriftHeatmap drifts={drifts} />
        </div>
      )}

      {/* Tab 4: Tool Schema Catalog */}
      {activeTab === 'catalog' && (
        <ToolRegistryView tools={tools} onToolsReloaded={fetchData} />
      )}

      {/* Live Benchmark Suite Modal */}
      <BenchmarkRunnerModal
        isOpen={isBenchmarkModalOpen}
        onClose={() => {
          setIsBenchmarkModalOpen(false);
          fetchData();
        }}
      />
    </div>
  );
};
