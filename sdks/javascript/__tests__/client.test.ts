import { BaconClient } from '../src/client';
import { BaconError } from '../src/errors';

const mockFetch = jest.fn() as jest.MockedFunction<typeof fetch>;

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
    headers: new Headers(),
    redirected: false,
    statusText: '',
    type: 'basic',
    url: '',
    clone: () => ({} as Response),
    body: null,
    bodyUsed: false,
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(0)),
    blob: () => Promise.resolve(new Blob()),
    formData: () => Promise.resolve(new FormData()),
    text: () => Promise.resolve(''),
    bytes: () => Promise.resolve(new Uint8Array()),
  } as Response;
}

function makeClient(baseURL = 'http://localhost:8080') {
  return new BaconClient(baseURL, { apiKey: 'test-key', fetch: mockFetch });
}

const sampleContext = { subjectId: 'user-123', environment: 'production' };
const sampleResult = {
  tenantId: 'tenant-1',
  flagKey: 'dark-mode',
  enabled: true,
  variant: 'on',
  reason: 'targeted',
};

beforeEach(() => mockFetch.mockReset());

describe('evaluate', () => {
  it('sends correct request and returns result', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse(sampleResult));

    const result = await makeClient().evaluate('dark-mode', sampleContext);

    expect(result).toEqual(sampleResult);
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/evaluate',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ flagKey: 'dark-mode', context: sampleContext }),
      }),
    );
  });

  it('sends API key header', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse(sampleResult));

    await makeClient().evaluate('dark-mode', sampleContext);

    const callArgs = mockFetch.mock.calls[0][1] as RequestInit;
    expect((callArgs.headers as Record<string, string>)['X-API-Key']).toBe('test-key');
  });

  it('sends Content-Type header', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse(sampleResult));

    await makeClient().evaluate('dark-mode', sampleContext);

    const callArgs = mockFetch.mock.calls[0][1] as RequestInit;
    expect((callArgs.headers as Record<string, string>)['Content-Type']).toBe('application/json');
  });

  it('throws BaconError on non-2xx response', async () => {
    const errorBody = { type: 'not_found', title: 'Not Found', detail: 'flag not found', instance: '/api/v1/evaluate' };
    mockFetch.mockResolvedValueOnce(jsonResponse(errorBody, 404));

    const err = await makeClient().evaluate('missing', sampleContext).catch((e) => e);
    expect(err).toBeInstanceOf(BaconError);
    expect(err).toMatchObject({
      statusCode: 404,
      type: 'not_found',
      title: 'Not Found',
      detail: 'flag not found',
    });
  });

  it('handles error response with no JSON body', async () => {
    const resp = {
      ...jsonResponse({}, 500),
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error('no json')),
    } as Response;
    mockFetch.mockResolvedValueOnce(resp);

    const err = await makeClient().evaluate('flag', sampleContext).catch((e) => e);
    expect(err).toBeInstanceOf(BaconError);
    expect(err).toMatchObject({ statusCode: 500, title: 'HTTP 500' });
  });
});

describe('evaluateBatch', () => {
  it('sends batch request and returns results array', async () => {
    const batchResponse = { results: [sampleResult, { ...sampleResult, flagKey: 'beta' }] };
    mockFetch.mockResolvedValueOnce(jsonResponse(batchResponse));

    const results = await makeClient().evaluateBatch(['dark-mode', 'beta'], sampleContext);

    expect(results).toHaveLength(2);
    expect(results[0].flagKey).toBe('dark-mode');
    expect(results[1].flagKey).toBe('beta');
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/evaluate/batch',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ flagKeys: ['dark-mode', 'beta'], context: sampleContext }),
      }),
    );
  });
});

describe('isEnabled', () => {
  it('returns true when flag is enabled', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ ...sampleResult, enabled: true }));
    expect(await makeClient().isEnabled('dark-mode', sampleContext)).toBe(true);
  });

  it('returns false when flag is disabled', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ ...sampleResult, enabled: false }));
    expect(await makeClient().isEnabled('dark-mode', sampleContext)).toBe(false);
  });

  it('returns false on error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('network error'));
    expect(await makeClient().isEnabled('dark-mode', sampleContext)).toBe(false);
  });
});

describe('getVariant', () => {
  it('returns variant string on success', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ ...sampleResult, variant: 'blue' }));
    expect(await makeClient().getVariant('button-color', sampleContext)).toBe('blue');
  });

  it('returns empty string on error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('fail'));
    expect(await makeClient().getVariant('button-color', sampleContext)).toBe('');
  });
});

describe('healthy', () => {
  it('returns true when status is ok', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ status: 'ok' }));
    expect(await makeClient().healthy()).toBe(true);
  });

  it('returns false when status is not ok', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ status: 'degraded' }));
    expect(await makeClient().healthy()).toBe(false);
  });

  it('returns false on network error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('connection refused'));
    expect(await makeClient().healthy()).toBe(false);
  });
});

describe('ready', () => {
  it('returns full health response', async () => {
    const health = {
      status: 'ok',
      modules: { database: { status: 'ok', latency_ms: 2 } },
    };
    mockFetch.mockResolvedValueOnce(jsonResponse(health));

    const result = await makeClient().ready();
    expect(result).toEqual(health);
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/readyz',
      expect.objectContaining({ method: 'GET' }),
    );
  });
});

describe('client configuration', () => {
  it('strips trailing slash from base URL', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ status: 'ok' }));
    await new BaconClient('http://localhost:8080/', { fetch: mockFetch }).healthy();
    expect(mockFetch).toHaveBeenCalledWith('http://localhost:8080/healthz', expect.anything());
  });

  it('works without API key', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse(sampleResult));
    const c = new BaconClient('http://localhost:8080', { fetch: mockFetch });
    await c.evaluate('flag', sampleContext);
    const callArgs = mockFetch.mock.calls[0][1] as RequestInit;
    expect((callArgs.headers as Record<string, string>)['X-API-Key']).toBeUndefined();
  });

  it('applies abort signal for timeout', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse(sampleResult));
    await makeClient().evaluate('flag', sampleContext);
    const callArgs = mockFetch.mock.calls[0][1] as RequestInit;
    expect(callArgs.signal).toBeInstanceOf(AbortSignal);
  });
});
