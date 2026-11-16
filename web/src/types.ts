export interface ParameterDiff {
  original_key: string;
  healed_key: string;
  original_value: any;
  healed_value: any;
  fix_type: 'KEY_CASE' | 'KEY_ALIAS' | 'KEY_FUZZY' | 'TYPE_COERCE' | 'ENUM_SNAP' | 'DATE_NORMALIZE' | 'DEFAULT_INJECT' | 'ENVELOPE_UNWRAP' | string;
  description: string;
  confidence: number;
}

export interface InvocationRecord {
  id: string;
  tool_name: string;
  status: 'healed_success' | 'passthrough_success' | 'fallback' | 'invalid' | string;
  original_args: Record<string, any>;
  healed_args: Record<string, any>;
  diffs: ParameterDiff[];
  healed: boolean;
  valid: boolean;
  tokens_saved: number;
  latency_us: number;
  created_at: string;
}

export interface TelemetryMetrics {
  total_invocations: number;
  healed_invocations: number;
  recovery_rate: number;
  tokens_saved: number;
  average_latency_us: number;
}

export interface DriftItem {
  tool_name: string;
  original_key: string;
  healed_key: string;
  fix_type: string;
  occurrences: number;
}

export interface PropertySchema {
  type: string;
  description?: string;
  enum?: string[];
  format?: string;
  default?: any;
  aliases?: string[];
}

export interface ToolDefinition {
  name: string;
  description: string;
  parameters: {
    type: string;
    required?: string[];
    properties: Record<string, PropertySchema>;
  };
}

export interface SuggestedPatch {
  tool_name: string;
  original_key: string;
  canonical_key: string;
  fix_type: string;
  confidence: number;
  hits: number;
}

export interface BenchmarkResult {
  total_cases: number;
  healed_cases: number;
  valid_cases: number;
  recovery_rate: number;
  avg_latency_us: number;
  tokens_saved: number;
  dollars_saved: number;
  fix_breakdown: Record<string, number>;
}
