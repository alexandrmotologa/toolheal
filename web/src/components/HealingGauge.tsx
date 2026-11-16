import React from 'react';
import { ShieldCheck, Zap } from 'lucide-react';

interface HealingGaugeProps {
  percentage: number;
  totalHealed: number;
  totalInvocations: number;
}

export const HealingGauge: React.FC<HealingGaugeProps> = ({
  percentage,
  totalHealed,
  totalInvocations,
}) => {
  const safePercent = Math.min(100, Math.max(0, percentage || 0));
  const radius = 64;
  const strokeWidth = 10;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (safePercent / 100) * circumference;

  const getStatusColor = () => {
    if (safePercent >= 90) return 'text-emerald-400 stroke-emerald-500';
    if (safePercent >= 80) return 'text-sky-400 stroke-sky-500';
    return 'text-amber-400 stroke-amber-500';
  };

  return (
    <div className="bg-[#101726] border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <ShieldCheck className="w-5 h-5 text-emerald-400" />
          <h3 className="text-sm font-semibold tracking-wide text-slate-300 uppercase">
            Auto-Repair Rate
          </h3>
        </div>
        <span className="px-2 py-0.5 text-xs font-mono font-medium rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center gap-1">
          <Zap className="w-3 h-3" /> SLA &gt; 85%
        </span>
      </div>

      <div className="flex items-center justify-around gap-6">
        {/* SVG Circular Gauge */}
        <div className="relative w-36 h-36 flex items-center justify-center">
          <svg className="w-full h-full transform -rotate-90">
            {/* Background Track */}
            <circle
              cx="72"
              cy="72"
              r={radius}
              stroke="#1e293b"
              strokeWidth={strokeWidth}
              fill="transparent"
            />
            {/* Animated Value Arc */}
            <circle
              cx="72"
              cy="72"
              r={radius}
              className={`transition-all duration-1000 ease-out ${getStatusColor()}`}
              strokeWidth={strokeWidth}
              strokeDasharray={circumference}
              strokeDashoffset={strokeDashoffset}
              strokeLinecap="round"
              fill="transparent"
            />
          </svg>

          {/* Centered Percentage Value */}
          <div className="absolute flex flex-col items-center justify-center text-center">
            <span className="text-3xl font-bold font-mono text-white tracking-tight">
              {safePercent.toFixed(1)}%
            </span>
            <span className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">
              {safePercent >= 85 ? 'Healthy' : 'Degraded'}
            </span>
          </div>
        </div>

        {/* Stats Summary */}
        <div className="flex flex-col gap-2">
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-lg p-2.5 min-w-[130px]">
            <span className="text-[11px] text-slate-400 font-medium block">
              Repaired Calls
            </span>
            <span className="text-lg font-bold font-mono text-emerald-400">
              {totalHealed.toLocaleString()}
            </span>
          </div>
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-lg p-2.5 min-w-[130px]">
            <span className="text-[11px] text-slate-400 font-medium block">
              Total Intercepted
            </span>
            <span className="text-lg font-bold font-mono text-slate-200">
              {totalInvocations.toLocaleString()}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
