import { EvaluationContext, EvaluationResult, BatchResult, ClientOptions, HealthResponse } from './types';
import { BaconError } from './errors';

export class BaconClient {
  private baseURL: string;
  private apiKey?: string;
  private timeout: number;
  private fetchFn: typeof fetch;

  constructor(baseURL: string, options: ClientOptions = {}) {
    this.baseURL = baseURL.replace(/\/$/, '');
    this.apiKey = options.apiKey;
    this.timeout = options.timeout ?? 5000;
    this.fetchFn = options.fetch ?? globalThis.fetch;
  }

  async evaluate(flagKey: string, context: EvaluationContext): Promise<EvaluationResult> {
    return this.post<EvaluationResult>('/api/v1/evaluate', { flagKey, context });
  }

  async evaluateBatch(flagKeys: string[], context: EvaluationContext): Promise<EvaluationResult[]> {
    const result = await this.post<BatchResult>('/api/v1/evaluate/batch', { flagKeys, context });
    return result.results;
  }

  async isEnabled(flagKey: string, context: EvaluationContext): Promise<boolean> {
    try {
      const result = await this.evaluate(flagKey, context);
      return result.enabled;
    } catch {
      return false;
    }
  }

  async getVariant(flagKey: string, context: EvaluationContext): Promise<string> {
    try {
      const result = await this.evaluate(flagKey, context);
      return result.variant;
    } catch {
      return '';
    }
  }

  async healthy(): Promise<boolean> {
    try {
      const resp = await this.get<{ status: string }>('/healthz');
      return resp.status === 'ok';
    } catch {
      return false;
    }
  }

  async ready(): Promise<HealthResponse> {
    return this.get<HealthResponse>('/readyz');
  }

  private async post<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'POST',
      body: JSON.stringify(body),
    });
  }

  private async get<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: 'GET' });
  }

  private async request<T>(path: string, init: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await this.fetchFn(`${this.baseURL}${path}`, {
        ...init,
        headers: { ...headers, ...init.headers as Record<string, string> },
        signal: controller.signal,
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({})) as Record<string, string>;
        throw new BaconError(
          response.status,
          error.type ?? '',
          error.title ?? `HTTP ${response.status}`,
          error.detail ?? '',
          error.instance ?? path,
        );
      }

      return await response.json() as T;
    } finally {
      clearTimeout(timeoutId);
    }
  }
}
