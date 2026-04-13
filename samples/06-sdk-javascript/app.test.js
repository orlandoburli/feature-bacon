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

beforeEach(() => {
  jest.clearAllMocks();
});

describe('GET /', () => {
  const batchResults = [
    { flagKey: 'dark_mode', enabled: true, variant: 'on', reason: 'rule-match' },
    { flagKey: 'new_pricing', enabled: false, variant: null, reason: 'default' },
    { flagKey: 'beta_features', enabled: true, variant: 'group-a', reason: 'rollout' },
    { flagKey: 'checkout_redesign', enabled: false, variant: null, reason: 'disabled' },
    { flagKey: 'maintenance_mode', enabled: false, variant: null, reason: 'default' },
  ];

  test('returns features for user', async () => {
    mockClient.evaluateBatch.mockResolvedValue(batchResults);

    const res = await request(app).get('/?user=alice').expect(200);

    expect(res.body.service).toBe('ecommerce-api');
    expect(res.body.user).toBe('alice');
    expect(res.body.features.dark_mode).toEqual({
      enabled: true, variant: 'on', reason: 'rule-match',
    });
    expect(res.body.features.new_pricing).toEqual({
      enabled: false, variant: null, reason: 'default',
    });
    expect(mockClient.evaluateBatch).toHaveBeenCalledWith(
      ['dark_mode', 'new_pricing', 'beta_features', 'checkout_redesign', 'maintenance_mode'],
      expect.objectContaining({ subjectId: 'alice' }),
    );
  });

  test('uses anonymous when no user provided', async () => {
    mockClient.evaluateBatch.mockResolvedValue(batchResults);

    const res = await request(app).get('/').expect(200);

    expect(res.body.user).toBe('anonymous');
    expect(mockClient.evaluateBatch).toHaveBeenCalledWith(
      expect.any(Array),
      expect.objectContaining({ subjectId: 'anonymous' }),
    );
  });

  test('uses X-User-Id header', async () => {
    mockClient.evaluateBatch.mockResolvedValue(batchResults);

    const res = await request(app)
      .get('/')
      .set('X-User-Id', 'header-user')
      .expect(200);

    expect(res.body.user).toBe('header-user');
    expect(mockClient.evaluateBatch).toHaveBeenCalledWith(
      expect.any(Array),
      expect.objectContaining({ subjectId: 'header-user' }),
    );
  });

  test('returns 500 on SDK error', async () => {
    mockClient.evaluateBatch.mockRejectedValue(new Error('connection refused'));

    const res = await request(app).get('/').expect(500);

    expect(res.body.error).toBe('connection refused');
  });
});

describe('GET /products', () => {
  test('applies discount when new pricing enabled', async () => {
    mockClient.isEnabled.mockResolvedValue(true);
    mockClient.getVariant.mockResolvedValue('v2');

    const res = await request(app).get('/products').expect(200);

    expect(res.body.newPricingActive).toBe(true);
    expect(res.body.checkoutVariant).toBe('v2');
    expect(res.body.products).toEqual([
      { id: 1, name: 'Widget Pro', price: 26.99 },
      { id: 2, name: 'Widget Basic', price: 8.99 },
      { id: 3, name: 'Widget Enterprise', price: 89.99 },
    ]);
  });

  test('uses full prices when new pricing disabled', async () => {
    mockClient.isEnabled.mockResolvedValue(false);
    mockClient.getVariant.mockResolvedValue('control');

    const res = await request(app).get('/products').expect(200);

    expect(res.body.newPricingActive).toBe(false);
    expect(res.body.checkoutVariant).toBe('control');
    expect(res.body.products).toEqual([
      { id: 1, name: 'Widget Pro', price: 29.99 },
      { id: 2, name: 'Widget Basic', price: 9.99 },
      { id: 3, name: 'Widget Enterprise', price: 99.99 },
    ]);
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
