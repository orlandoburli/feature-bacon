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

const VARIANT_STANDARD = 'standard';
const VARIANT_VOLUME_DISCOUNT = 'volume_discount';
const VARIANT_V1_EXACT = 'v1_exact';
const VARIANT_V2_FUZZY = 'v2_fuzzy';
const STATUS_OK = 'ok';

beforeEach(() => {
  jest.clearAllMocks();
});

describe('GET /api/products', () => {
  test('returns standard pricing for free users', async () => {
    mockClient.getVariant.mockResolvedValue(VARIANT_STANDARD);

    const res = await request(app)
      .get('/api/products?user=free_user&plan=free')
      .expect(200);

    expect(res.body.algorithm).toBe(VARIANT_STANDARD);
    expect(res.body.products).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ id: 1, name: 'Widget Pro', price: 29.99, pricingLabel: VARIANT_STANDARD }),
      ]),
    );
    expect(mockClient.getVariant).toHaveBeenCalledWith(
      'pricing_algorithm',
      expect.objectContaining({ subjectId: 'free_user', attributes: { plan: 'free' } }),
    );
  });

  test('returns volume discount for enterprise', async () => {
    mockClient.getVariant.mockResolvedValue(VARIANT_VOLUME_DISCOUNT);

    const res = await request(app)
      .get('/api/products?user=enterprise_user&plan=enterprise')
      .expect(200);

    expect(res.body.algorithm).toBe(VARIANT_VOLUME_DISCOUNT);
    res.body.products.forEach(p => {
      expect(p.pricingLabel).toBe(VARIANT_VOLUME_DISCOUNT);
    });
    expect(res.body.products[0].price).toBe(23.99);
  });
});

describe('GET /api/search', () => {
  test('v1 does exact match', async () => {
    mockClient.getVariant.mockResolvedValue(VARIANT_V1_EXACT);

    const res = await request(app)
      .get('/api/search?q=Widget&user=test')
      .expect(200);

    expect(res.body.version).toBe(VARIANT_V1_EXACT);
    expect(res.body.query).toBe('Widget');
    res.body.results.forEach(r => {
      expect(r.name.toLowerCase()).toContain('widget');
    });
  });

  test('v2 does fuzzy match', async () => {
    mockClient.getVariant.mockResolvedValue(VARIANT_V2_FUZZY);

    const res = await request(app)
      .get('/api/search?q=wdg&user=test')
      .expect(200);

    expect(res.body.version).toBe(VARIANT_V2_FUZZY);
    expect(res.body.results.length).toBeGreaterThan(0);
  });
});

describe('GET /api/premium/analytics', () => {
  test('returns 403 when flag disabled', async () => {
    mockClient.isEnabled.mockResolvedValue(false);

    const res = await request(app)
      .get('/api/premium/analytics?user=free_user&plan=free')
      .expect(403);

    expect(res.body.error).toBe('Premium API access required');
    expect(mockClient.isEnabled).toHaveBeenCalledWith(
      'premium_api',
      expect.objectContaining({ subjectId: 'free_user' }),
    );
  });

  test('returns data when flag enabled', async () => {
    mockClient.isEnabled.mockResolvedValue(true);

    const res = await request(app)
      .get('/api/premium/analytics?user=enterprise_user&plan=enterprise')
      .expect(200);

    expect(res.body.totalOrders).toBe(1284);
    expect(res.body.revenue).toBe(48210.5);
    expect(res.body.topProduct).toBe('Enterprise Suite');
  });
});

describe('GET /health', () => {
  test('returns 200 when healthy', async () => {
    mockClient.healthy.mockResolvedValue(true);

    const res = await request(app).get('/health').expect(200);

    expect(res.body).toEqual({ status: STATUS_OK, baconHealthy: true });
  });
});

describe('GET /', () => {
  test('renders dashboard', async () => {
    mockClient.evaluateBatch.mockResolvedValue([
      { flagKey: 'pricing_algorithm', enabled: true, variant: VARIANT_STANDARD, reason: 'default' },
      { flagKey: 'search_version', enabled: true, variant: VARIANT_V1_EXACT, reason: 'default' },
      { flagKey: 'rate_limit', enabled: true, variant: 'strict', reason: 'default' },
      { flagKey: 'cache_strategy', enabled: true, variant: 'conservative', reason: 'default' },
      { flagKey: 'premium_api', enabled: false, variant: '', reason: 'default' },
    ]);

    const res = await request(app)
      .get('/?user=free_user&plan=free')
      .expect(200);

    expect(res.text).toContain('Backend Feature Flags');
    expect(res.text).toContain('Pricing Engine');
    expect(res.text).toContain('Search Engine');
    expect(res.text).toContain('Rate Limiting');
    expect(res.text).toContain('Cache Strategy');
    expect(res.text).toContain('Premium API');
    expect(res.text).toContain('free_user');
  });
});
