import React from 'react';
import { Dashboard } from './pages/Dashboard';

export const App: React.FC = () => {
  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col justify-between">
      <main className="flex-1">
        <Dashboard />
      </main>

      <footer className="border-t border-slate-900 py-6 text-center text-xs text-slate-400">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-2">
          <span>toolheal: Autonomous Self-Healing Tool Calling Middleware</span>
          <span className="font-mono text-[11px] text-slate-400">
            Single-Binary Go Engine (&lt; 15MB RAM, &lt; 2ms SLA) + React 19 Console
          </span>
        </div>
      </footer>
    </div>
  );
};

export default App;
