jest.mock('feature-bacon', () => {
  const mockClient = {
    evaluateBatch: jest.fn(),
    isEnabled: jest.fn(),
    getVariant: jest.fn(),
    healthy: jest.fn(),
  };
  return { BaconClient: jest.fn(() => mockClient), __mockClient: mockClient };
});

const request = require('supertest');
const { __mockClient: mockClient } = require('feature-bacon');
const app = require('./app');

const THEME_DARK_BG = '#1a1a2e';
const BODY_DARK = 'class="theme-dark"';
const BODY_LIGHT = 'class="theme-light"';
const MAINTENANCE_TEXT = 'Under Maintenance';
const SUMMER_SALE_TEXT = 'Summer Sale';
const HERO_DEFAULT_TEXT = 'Welcome to Feature Bacon';
const CTA_BOLD_CLASS = 'cta--bold';
const CTA_MINIMAL_CLASS = 'cta--minimal';
const CTA_CLASSIC_CLASS = 'cta--classic';

function makeBatchResults(overrides = {}) {
  const defaults = {
    dark_mode: { enabled: false, variant: null },
    hero_banner: { enabled: true, variant: 'default' },
    new_navigation: { enabled: false, variant: null },
    cta_experiment: { enabled: true, variant: 'classic' },
    maintenance_mode: { enabled: false, variant: null },
  };
  const merged = { ...defaults, ...overrides };
  return Object.entries(merged).map(([flagKey, val]) => ({
    flagKey,
    enabled: val.enabled,
    variant: val.variant,
    reason: 'mock',
  }));
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe('GET /', () => {
  test('renders dark theme when dark_mode enabled', async () => {
    mockClient.evaluateBatch.mockResolvedValue(
      makeBatchResults({ dark_mode: { enabled: true, variant: null } }),
    );

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(BODY_DARK);
    expect(res.text).not.toContain(BODY_LIGHT);
  });

  test('renders light theme when dark_mode disabled', async () => {
    mockClient.evaluateBatch.mockResolvedValue(makeBatchResults());

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(BODY_LIGHT);
    expect(res.text).not.toContain(BODY_DARK);
  });

  test('shows maintenance overlay when maintenance_mode on', async () => {
    mockClient.evaluateBatch.mockResolvedValue(
      makeBatchResults({ maintenance_mode: { enabled: true, variant: null } }),
    );

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(MAINTENANCE_TEXT);
    expect(res.text).toContain('maintenance-overlay');
  });

  test('shows hero banner variant', async () => {
    mockClient.evaluateBatch.mockResolvedValue(
      makeBatchResults({ hero_banner: { enabled: true, variant: 'summer_sale' } }),
    );

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(SUMMER_SALE_TEXT);
    expect(res.text).toContain('hero--summer');
  });

  test('shows default hero banner when variant is default', async () => {
    mockClient.evaluateBatch.mockResolvedValue(makeBatchResults());

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(HERO_DEFAULT_TEXT);
    expect(res.text).toContain('hero--default');
  });

  test('shows CTA button variant bold', async () => {
    mockClient.evaluateBatch.mockResolvedValue(
      makeBatchResults({ cta_experiment: { enabled: true, variant: 'bold' } }),
    );

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(CTA_BOLD_CLASS);
    expect(res.text).toContain('GET STARTED NOW');
  });

  test('shows CTA button variant minimal', async () => {
    mockClient.evaluateBatch.mockResolvedValue(
      makeBatchResults({ cta_experiment: { enabled: true, variant: 'minimal' } }),
    );

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(CTA_MINIMAL_CLASS);
  });

  test('shows CTA button variant classic by default', async () => {
    mockClient.evaluateBatch.mockResolvedValue(makeBatchResults());

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain(CTA_CLASSIC_CLASS);
  });

  test('uses visitor as default user', async () => {
    mockClient.evaluateBatch.mockResolvedValue(makeBatchResults());

    const res = await request(app).get('/').expect(200);

    expect(res.text).toContain('User: visitor');
    expect(mockClient.evaluateBatch).toHaveBeenCalledWith(
      expect.any(Array),
      expect.objectContaining({ subjectId: 'visitor' }),
    );
  });

  test('uses user query parameter', async () => {
    mockClient.evaluateBatch.mockResolvedValue(makeBatchResults());

    const res = await request(app).get('/?user=alice').expect(200);

    expect(res.text).toContain('User: alice');
    expect(mockClient.evaluateBatch).toHaveBeenCalledWith(
      expect.any(Array),
      expect.objectContaining({ subjectId: 'alice' }),
    );
  });

  test('returns 500 on SDK error', async () => {
    mockClient.evaluateBatch.mockRejectedValue(new Error('connection refused'));

    const res = await request(app).get('/').expect(500);

    expect(res.text).toContain('connection refused');
  });
});

describe('GET /health', () => {
  test('returns 200 when healthy', async () => {
    mockClient.healthy.mockResolvedValue(true);

    const res = await request(app).get('/health').expect(200);

    expect(res.body).toEqual({ status: 'ok', baconHealthy: true });
  });

  test('returns 503 when unhealthy', async () => {
    mockClient.healthy.mockResolvedValue(false);

    const res = await request(app).get('/health').expect(503);

    expect(res.body).toEqual({ status: 'degraded', baconHealthy: false });
  });
});

describe('GET /api/flags', () => {
  test('returns JSON flag evaluations', async () => {
    mockClient.evaluateBatch.mockResolvedValue(makeBatchResults());

    const res = await request(app).get('/api/flags').expect(200);

    expect(res.body.user).toBe('visitor');
    expect(res.body.flags).toBeDefined();
    expect(res.body.flags.dark_mode).toEqual(
      expect.objectContaining({ enabled: false }),
    );
  });

  test('returns JSON for specific user', async () => {
    mockClient.evaluateBatch.mockResolvedValue(makeBatchResults());

    const res = await request(app).get('/api/flags?user=bob').expect(200);

    expect(res.body.user).toBe('bob');
  });

  test('returns 500 on SDK error', async () => {
    mockClient.evaluateBatch.mockRejectedValue(new Error('timeout'));

    const res = await request(app).get('/api/flags').expect(500);

    expect(res.body.error).toBe('timeout');
  });
});
