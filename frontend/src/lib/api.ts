const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.detail || `API error ${res.status}`);
  }
  return res.json();
}

export interface Flag {
  key: string;
  type: string;
  semantics: string;
  enabled: boolean;
  description?: string;
  rules?: Rule[];
  defaultResult?: { enabled: boolean; variant: string };
  createdAt?: number;
  updatedAt?: number;
}

export interface Rule {
  conditions?: Condition[];
  rolloutPercentage: number;
  variant: string;
}

export interface Condition {
  attribute: string;
  operator: string;
  valueJson: string;
}

export interface Experiment {
  key: string;
  name: string;
  status: string;
  stickyAssignment: boolean;
  variants?: { key: string; description?: string }[];
  allocation?: { variantKey: string; percentage: number }[];
  createdAt?: number;
  updatedAt?: number;
}

export interface APIKey {
  id: string;
  keyPrefix: string;
  scope: string;
  name: string;
  createdAt?: number;
  revokedAt?: number;
}

export interface PaginatedResponse {
  pagination: { page: number; perPage: number; total: number; totalPages: number };
}

export const flagsAPI = {
  list: (page = 1) =>
    fetchAPI<{ flags: Flag[] } & PaginatedResponse>(`/api/v1/flags?page=${page}`),
  get: (key: string) => fetchAPI<{ flag: Flag }>(`/api/v1/flags/${key}`),
  create: (flag: Partial<Flag>) =>
    fetchAPI<{ flag: Flag }>('/api/v1/flags', { method: 'POST', body: JSON.stringify(flag) }),
  update: (key: string, flag: Partial<Flag>) =>
    fetchAPI<{ flag: Flag }>(`/api/v1/flags/${key}`, { method: 'PUT', body: JSON.stringify(flag) }),
  delete: (key: string) => fetchAPI<void>(`/api/v1/flags/${key}`, { method: 'DELETE' }),
};

export const experimentsAPI = {
  list: (page = 1) =>
    fetchAPI<{ experiments: Experiment[] } & PaginatedResponse>(
      `/api/v1/experiments?page=${page}`
    ),
  get: (key: string) => fetchAPI<{ experiment: Experiment }>(`/api/v1/experiments/${key}`),
  create: (exp: Partial<Experiment>) =>
    fetchAPI<{ experiment: Experiment }>('/api/v1/experiments', {
      method: 'POST',
      body: JSON.stringify(exp),
    }),
  update: (key: string, exp: Partial<Experiment>) =>
    fetchAPI<{ experiment: Experiment }>(`/api/v1/experiments/${key}`, {
      method: 'PUT',
      body: JSON.stringify(exp),
    }),
  start: (key: string) =>
    fetchAPI<void>(`/api/v1/experiments/${key}/start`, { method: 'POST' }),
  pause: (key: string) =>
    fetchAPI<void>(`/api/v1/experiments/${key}/pause`, { method: 'POST' }),
  complete: (key: string) =>
    fetchAPI<void>(`/api/v1/experiments/${key}/complete`, { method: 'POST' }),
};

export const apiKeysAPI = {
  list: (page = 1) =>
    fetchAPI<{ apiKeys: APIKey[] } & PaginatedResponse>(`/api/v1/api-keys?page=${page}`),
  create: (data: { name: string; scope: string }) =>
    fetchAPI<{ rawKey: string; apiKey: APIKey }>('/api/v1/api-keys', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  revoke: (id: string) => fetchAPI<void>(`/api/v1/api-keys/${id}`, { method: 'DELETE' }),
};

export const healthAPI = {
  readyz: () =>
    fetchAPI<{
      status: string;
      modules: Record<string, { status: string; latency_ms?: number; message?: string }>;
    }>('/readyz'),
};
