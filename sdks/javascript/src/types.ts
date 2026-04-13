export interface EvaluationContext {
  subjectId: string;
  environment?: string;
  attributes?: Record<string, unknown>;
}

export interface EvaluationResult {
  tenantId: string;
  flagKey: string;
  enabled: boolean;
  variant: string;
  reason: string;
}

export interface BatchResult {
  results: EvaluationResult[];
}

export interface ClientOptions {
  apiKey?: string;
  timeout?: number;
  fetch?: typeof fetch;
}

export interface HealthResponse {
  status: string;
  modules?: Record<string, { status: string; latency_ms?: number; message?: string }>;
}
