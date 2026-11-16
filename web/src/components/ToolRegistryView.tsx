import React from 'react';
import { ToolDefinition } from '../types';
import { BookOpen, Key, Tag } from 'lucide-react';

interface ToolRegistryViewProps {
  tools: ToolDefinition[];
}

export const ToolRegistryView: React.FC<ToolRegistryViewProps> = ({ tools }) => {
  return (
    <div className="bg-[#101726] border border-slate-800 rounded-xl p-6 shadow-xl space-y-6">
      <div className="flex items-center justify-between border-b border-slate-800 pb-4">
        <div>
          <h2 className="text-base font-semibold text-white flex items-center gap-2">
            <BookOpen className="w-5 h-5 text-sky-400" />
            Canonical Tool Schema Catalog
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Registered tools, parameter constraints, enum sets, and semantic alias mappings.
          </p>
        </div>
        <span className="text-xs font-mono bg-sky-500/10 text-sky-400 border border-sky-500/20 px-2.5 py-1 rounded-full">
          {tools.length} Tools Registered
        </span>
      </div>

      <div className="space-y-6">
        {tools.map((tool) => {
          const props = tool.parameters.properties || {};
          const required = tool.parameters.required || [];

          return (
            <div
              key={tool.name}
              className="border border-slate-800 rounded-lg p-5 bg-slate-900/40 space-y-3"
            >
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <div>
                  <h3 className="text-sm font-bold font-mono text-emerald-400">
                    {tool.name}
                  </h3>
                  <p className="text-xs text-slate-400 mt-0.5">{tool.description}</p>
                </div>
              </div>

              {/* Parameters Table */}
              <div className="border border-slate-800/80 rounded-lg overflow-x-auto">
                <table className="w-full text-left text-xs font-mono">
                  <thead className="bg-slate-950/80 text-slate-400 border-b border-slate-800">
                    <tr>
                      <th className="py-2.5 px-3">Parameter Key</th>
                      <th className="py-2.5 px-3">Type / Format</th>
                      <th className="py-2.5 px-3">Constraints & Aliases</th>
                      <th className="py-2.5 px-3">Description</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60 bg-slate-900/30">
                    {Object.entries(props).map(([propKey, prop]) => {
                      const isRequired = required.includes(propKey);

                      return (
                        <tr key={propKey} className="hover:bg-slate-800/30">
                          <td className="py-2.5 px-3 font-semibold text-slate-200">
                            <span className="flex items-center gap-1.5">
                              <Key className="w-3 h-3 text-sky-400" />
                              {propKey}
                              {isRequired && (
                                <span className="text-[10px] text-rose-400 font-sans font-bold bg-rose-500/10 px-1 rounded">
                                  REQUIRED
                                </span>
                              )}
                            </span>
                          </td>

                          <td className="py-2.5 px-3 text-sky-400">
                            {prop.type}
                            {prop.format && (
                              <span className="text-slate-400 text-[11px] block">
                                format: {prop.format}
                              </span>
                            )}
                          </td>

                          <td className="py-2.5 px-3">
                            {prop.enum && (
                              <div className="flex flex-wrap gap-1 mb-1">
                                {prop.enum.map((e) => (
                                  <span
                                    key={e}
                                    className="px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-300 text-[10px] border border-amber-500/20"
                                  >
                                    {e}
                                  </span>
                                ))}
                              </div>
                            )}

                            {prop.aliases && prop.aliases.length > 0 && (
                              <div className="flex items-center gap-1 flex-wrap text-[11px] text-slate-400">
                                <Tag className="w-3 h-3 text-emerald-400 shrink-0" />
                                <span>{prop.aliases.join(', ')}</span>
                              </div>
                            )}
                          </td>

                          <td className="py-2.5 px-3 font-sans text-slate-300">
                            {prop.description || '-'}
                            {prop.default !== undefined && (
                              <span className="block text-[11px] text-slate-500 font-mono">
                                default: {JSON.stringify(prop.default)}
                              </span>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
