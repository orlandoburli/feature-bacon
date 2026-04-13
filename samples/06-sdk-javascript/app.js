const express = require('express');
const { BaconClient } = require('feature-bacon');

const app = express();
app.disable('x-powered-by');
const port = process.env.PORT || 3000;
const baconUrl = process.env.BACON_URL || 'http://localhost:8080';
const apiKey = process.env.BACON_API_KEY || '';

const client = new BaconClient(baconUrl, { apiKey });

// Middleware to extract user context from headers/query
function userContext(req) {
  return {
    subjectId: req.query.user || req.headers['x-user-id'] || 'anonymous',
    environment: process.env.ENVIRONMENT || 'production',
    attributes: {
      plan: req.query.plan || 'free',
      country: req.query.country || 'US',
    },
  };
}

// Home — show all active features for user
app.get('/', async (req, res) => {
  const ctx = userContext(req);
  try {
    const results = await client.evaluateBatch(
      ['dark_mode', 'new_pricing', 'beta_features', 'checkout_redesign', 'maintenance_mode'],
      ctx
    );
    const features = {};
    results.forEach(r => {
      features[r.flagKey] = { enabled: r.enabled, variant: r.variant, reason: r.reason };
    });
    res.json({ service: 'ecommerce-api', user: ctx.subjectId, features });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Product listing — price varies based on flags
app.get('/products', async (req, res) => {
  const ctx = userContext(req);
  const [newPricing, variant] = await Promise.all([
    client.isEnabled('new_pricing', ctx),
    client.getVariant('checkout_redesign', ctx),
  ]);

  const discount = newPricing ? 0.9 : 1;
  const products = [
    { id: 1, name: 'Widget Pro', price: +(29.99 * discount).toFixed(2) },
    { id: 2, name: 'Widget Basic', price: +(9.99 * discount).toFixed(2) },
    { id: 3, name: 'Widget Enterprise', price: +(99.99 * discount).toFixed(2) },
  ];

  res.json({ products, checkoutVariant: variant, newPricingActive: newPricing });
});

// Health — includes Feature Bacon health
app.get('/health', async (_req, res) => {
  const healthy = await client.healthy();
  const status = healthy ? 'ok' : 'degraded';
  if (!healthy) res.status(503);
  res.json({ status, baconHealthy: healthy });
});

app.listen(port, () => console.log(`E-commerce API on :${port}`));
