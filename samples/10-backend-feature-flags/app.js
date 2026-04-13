const express = require('express');
const { BaconClient } = require('feature-bacon');

const app = express();
app.disable('x-powered-by');
const port = process.env.PORT || 3000;
const baconUrl = process.env.BACON_URL || 'http://localhost:8080';
const apiKey = process.env.BACON_API_KEY || '';

const client = new BaconClient(baconUrl, { apiKey });

const FLAG_KEYS = [
  'pricing_algorithm',
  'search_version',
  'rate_limit',
  'cache_strategy',
  'premium_api',
];

const PRODUCTS = [
  { id: 1, name: 'Widget Pro', category: 'widgets', basePrice: 29.99 },
  { id: 2, name: 'Widget Basic', category: 'widgets', basePrice: 9.99 },
  { id: 3, name: 'Gadget X', category: 'gadgets', basePrice: 49.99 },
  { id: 4, name: 'Gadget Mini', category: 'gadgets', basePrice: 19.99 },
  { id: 5, name: 'Mega Bundle', category: 'bundles', basePrice: 99.99 },
  { id: 6, name: 'Starter Kit', category: 'kits', basePrice: 14.99 },
  { id: 7, name: 'Enterprise Suite', category: 'bundles', basePrice: 199.99 },
  { id: 8, name: 'Widget Nano', category: 'widgets', basePrice: 4.99 },
];

const RATE_LIMITS = {
  strict: { requests: 100, window: '1m', burst: 10 },
  relaxed: { requests: 1000, window: '1m', burst: 100 },
};

const CACHE_CONFIGS = {
  conservative: { ttl: 60, maxSize: 100, staleWhileRevalidate: false },
  aggressive: { ttl: 300, maxSize: 1000, staleWhileRevalidate: true },
};

function userContext(req) {
  return {
    subjectId: req.query.user || req.headers['x-user-id'] || 'anonymous',
    environment: process.env.ENVIRONMENT || 'production',
    attributes: {
      plan: req.query.plan || 'free',
    },
  };
}

function applyPricing(products, algorithm) {
  const hour = new Date().getHours();
  return products.map(p => {
    let price = p.basePrice;
    let label = 'standard';
    if (algorithm === 'dynamic') {
      const multiplier = hour >= 9 && hour <= 17 ? 1.15 : 0.9;
      price = +(p.basePrice * multiplier).toFixed(2);
      label = 'dynamic';
    } else if (algorithm === 'volume_discount') {
      price = +(p.basePrice * 0.8).toFixed(2);
      label = 'volume_discount';
    }
    return { id: p.id, name: p.name, category: p.category, price, pricingLabel: label };
  });
}

function searchExact(products, query) {
  const q = query.toLowerCase();
  return products.filter(p => p.name.toLowerCase().includes(q));
}

function searchFuzzy(products, query) {
  const q = query.toLowerCase();
  return products.filter(p => {
    const name = p.name.toLowerCase();
    if (name.includes(q)) return true;
    if (name.split(/\s+/).some(w => w.startsWith(q))) return true;
    let qi = 0;
    for (let i = 0; i < name.length && qi < q.length; i++) {
      if (name[i] === q[qi]) qi++;
    }
    return qi === q.length;
  });
}

function escapeHtml(str) {
  return String(str).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;');
}

function syntaxHighlight(json) {
  return json
    .replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
    .replaceAll(/"([^"]+)":/g, '<span class="key">"$1"</span>:')
    .replaceAll(/: "([^"]*)"/g, ': <span class="str">"$1"</span>')
    .replaceAll(': true', ': <span class="bool-true">true</span>')
    .replaceAll(': false', ': <span class="bool-false">false</span>')
    .replaceAll(': null', ': <span class="null">null</span>')
    .replaceAll(/: (-?\d+\.?\d*)/g, ': <span class="num">$1</span>');
}

function badgeClass(flag) {
  if (!flag.enabled) return 'badge-red';
  const defaults = ['standard', 'v1_exact', 'strict', 'conservative'];
  if (defaults.includes(flag.variant)) return 'badge-yellow';
  return 'badge-green';
}

function renderBadges(flags) {
  return FLAG_KEYS.map(k => {
    const f = flags[k];
    const val = f.variant || (f.enabled ? 'on' : 'off');
    const cls = badgeClass(f);
    return `<span class="badge ${cls}"><span class="dot"></span>${escapeHtml(k)}: ${escapeHtml(val)}</span>`;
  }).join('\n      ');
}

function renderPricingCard(pricingAlg, samplePrices) {
  let indicatorClass;
  if (pricingAlg === 'standard') indicatorClass = 'indicator-yellow';
  else if (pricingAlg === 'dynamic') indicatorClass = 'indicator-blue';
  else indicatorClass = 'indicator-green';

  let multiplier;
  if (pricingAlg === 'dynamic') multiplier = 'time-based';
  else if (pricingAlg === 'volume_discount') multiplier = '0.8x';
  else multiplier = '1.0x';

  let compare;
  if (pricingAlg === 'volume_discount') compare = '<strong>Enterprise</strong> gets 20% volume discounts';
  else if (pricingAlg === 'dynamic') compare = '<strong>Dynamic</strong> pricing varies by time of day';
  else compare = 'Other users may see <strong>dynamic</strong> or <strong>volume_discount</strong> pricing';

  const escaped = escapeHtml(pricingAlg);
  return `<div class="card">
      <div class="card-header">
        <div><h3>Pricing Engine</h3><span class="flag-name">pricing_algorithm</span></div>
        <span class="indicator ${indicatorClass}">${escaped}</span>
      </div>
      <div class="card-body">
        <div class="stat-row"><span class="stat-label">Algorithm</span><span class="stat-value">${escaped}</span></div>
        <div class="stat-row"><span class="stat-label">Multiplier</span><span class="stat-value">${escapeHtml(multiplier)}</span></div>
        <div class="try-it">
          <div class="try-it-label">GET /api/products &mdash; sample</div>
          <pre class="json">${syntaxHighlight(JSON.stringify({ algorithm: pricingAlg, products: samplePrices }, null, 2))}</pre>
        </div>
        <div class="compare">${compare}</div>
      </div>
    </div>`;
}

function renderSearchCard(searchVer, sampleSearch) {
  const escaped = escapeHtml(searchVer);
  const indicatorClass = searchVer === 'v2_fuzzy' ? 'indicator-green' : 'indicator-yellow';
  const algorithm = searchVer === 'v2_fuzzy' ? 'Fuzzy (case-insensitive, partial)' : 'Exact (substring match)';
  const compare = searchVer === 'v2_fuzzy'
    ? '<strong>v2_fuzzy</strong> matches partial words and subsequences'
    : '50% of users see <strong>v2_fuzzy</strong> with smarter matching';

  return `<div class="card">
      <div class="card-header">
        <div><h3>Search Engine</h3><span class="flag-name">search_version</span></div>
        <span class="indicator ${indicatorClass}">${escaped}</span>
      </div>
      <div class="card-body">
        <div class="stat-row"><span class="stat-label">Algorithm</span><span class="stat-value">${algorithm}</span></div>
        <div class="stat-row"><span class="stat-label">Query</span><span class="stat-value">"wid"</span></div>
        <div class="try-it">
          <div class="try-it-label">GET /api/search?q=wid &mdash; sample</div>
          <pre class="json">${syntaxHighlight(JSON.stringify({ version: searchVer, query: 'wid', results: sampleSearch.map(p => ({ id: p.id, name: p.name })) }, null, 2))}</pre>
        </div>
        <div class="compare">${compare}</div>
      </div>
    </div>`;
}

function renderRateLimitCard(rateLimit, rateCfg) {
  const escaped = escapeHtml(rateLimit);
  const indicatorClass = rateLimit === 'relaxed' ? 'indicator-green' : 'indicator-yellow';
  const compare = rateLimit === 'relaxed'
    ? '<strong>Premium</strong> users enjoy 10x higher limits'
    : '<strong>Premium</strong> users get <strong>relaxed</strong> limits (1000 req/min)';

  return `<div class="card">
      <div class="card-header">
        <div><h3>Rate Limiting</h3><span class="flag-name">rate_limit</span></div>
        <span class="indicator ${indicatorClass}">${escaped}</span>
      </div>
      <div class="card-body">
        <div class="stat-row"><span class="stat-label">Requests</span><span class="stat-value">${escapeHtml(rateCfg.requests)} / ${escapeHtml(rateCfg.window)}</span></div>
        <div class="stat-row"><span class="stat-label">Burst</span><span class="stat-value">${escapeHtml(rateCfg.burst)}</span></div>
        <div class="stat-row"><span class="stat-label">Strategy</span><span class="stat-value">${escaped}</span></div>
        <div class="try-it">
          <div class="try-it-label">Active rate limit config</div>
          <pre class="json">${syntaxHighlight(JSON.stringify({ strategy: rateLimit, ...rateCfg }, null, 2))}</pre>
        </div>
        <div class="compare">${compare}</div>
      </div>
    </div>`;
}

function renderCacheCard(cacheStrat, cacheCfg) {
  const escaped = escapeHtml(cacheStrat);
  const indicatorClass = cacheStrat === 'aggressive' ? 'indicator-green' : 'indicator-yellow';
  const compare = cacheStrat === 'aggressive'
    ? '<strong>Aggressive</strong> caching: 5min TTL with stale-while-revalidate'
    : '40% of users get <strong>aggressive</strong> caching (5x longer TTL)';

  return `<div class="card">
      <div class="card-header">
        <div><h3>Cache Strategy</h3><span class="flag-name">cache_strategy</span></div>
        <span class="indicator ${indicatorClass}">${escaped}</span>
      </div>
      <div class="card-body">
        <div class="stat-row"><span class="stat-label">TTL</span><span class="stat-value">${escapeHtml(cacheCfg.ttl)}s</span></div>
        <div class="stat-row"><span class="stat-label">Max Size</span><span class="stat-value">${escapeHtml(cacheCfg.maxSize)}</span></div>
        <div class="stat-row"><span class="stat-label">Stale-While-Revalidate</span><span class="stat-value">${escapeHtml(cacheCfg.staleWhileRevalidate)}</span></div>
        <div class="try-it">
          <div class="try-it-label">Active cache config</div>
          <pre class="json">${syntaxHighlight(JSON.stringify({ strategy: cacheStrat, ...cacheCfg }, null, 2))}</pre>
        </div>
        <div class="compare">${compare}</div>
      </div>
    </div>`;
}

function renderPremiumCard(premiumEnabled) {
  const premiumSample = premiumEnabled
    ? syntaxHighlight(JSON.stringify({ totalOrders: 1284, revenue: 48210.5, topProduct: 'Enterprise Suite', conversionRate: 0.032, period: 'last_30_days' }, null, 2))
    : syntaxHighlight(JSON.stringify({ error: 'Premium API access required', hint: 'Upgrade to premium or enterprise plan' }, null, 2));
  const compare = premiumEnabled
    ? '<strong>Premium/Enterprise</strong> users can access analytics data'
    : 'Upgrade to <strong>premium</strong> or <strong>enterprise</strong> to unlock this endpoint';

  return `<div class="card">
      <div class="card-header">
        <div><h3>Premium API</h3><span class="flag-name">premium_api</span></div>
        <span class="indicator ${premiumEnabled ? 'indicator-green' : 'indicator-red'}">${premiumEnabled ? 'enabled' : 'disabled'}</span>
      </div>
      <div class="card-body">
        <div class="stat-row"><span class="stat-label">Access</span><span class="stat-value">${premiumEnabled ? 'Granted' : 'Denied'}</span></div>
        <div class="stat-row"><span class="stat-label">Endpoint</span><span class="stat-value">/api/premium/analytics</span></div>
        <div class="try-it">
          <div class="try-it-label">GET /api/premium/analytics</div>
          <pre class="json">${premiumSample}</pre>
        </div>
        <div class="compare">${compare}</div>
      </div>
    </div>`;
}

app.get('/api/products', async (req, res) => {
  const ctx = userContext(req);
  try {
    const algorithm = await client.getVariant('pricing_algorithm', ctx);
    const variant = algorithm || 'standard';
    const products = applyPricing(PRODUCTS, variant);
    res.json({ algorithm: variant, products });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.get('/api/search', async (req, res) => {
  const ctx = userContext(req);
  const query = req.query.q || '';
  try {
    const version = await client.getVariant('search_version', ctx);
    const variant = version || 'v1_exact';
    const results = variant === 'v2_fuzzy'
      ? searchFuzzy(PRODUCTS, query)
      : searchExact(PRODUCTS, query);
    res.json({ version: variant, query, results });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.get('/api/premium/analytics', async (req, res) => {
  const ctx = userContext(req);
  try {
    const enabled = await client.isEnabled('premium_api', ctx);
    if (!enabled) {
      return res.status(403).json({
        error: 'Premium API access required',
        hint: 'Upgrade to premium or enterprise plan',
      });
    }
    res.json({
      totalOrders: 1284,
      revenue: 48210.5,
      topProduct: 'Enterprise Suite',
      conversionRate: 0.032,
      period: 'last_30_days',
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.get('/health', async (_req, res) => {
  const healthy = await client.healthy();
  const status = healthy ? 'ok' : 'degraded';
  if (!healthy) res.status(503);
  res.json({ status, baconHealthy: healthy });
});

app.get('/', async (req, res) => {
  const ctx = userContext(req);
  let flags = {};
  try {
    const results = await client.evaluateBatch(FLAG_KEYS, ctx);
    results.forEach(r => {
      flags[r.flagKey] = { enabled: r.enabled, variant: r.variant, reason: r.reason };
    });
  } catch {
    FLAG_KEYS.forEach(k => {
      flags[k] = { enabled: false, variant: 'unknown', reason: 'error' };
    });
  }

  const pricingAlg = flags.pricing_algorithm?.variant || 'standard';
  const searchVer = flags.search_version?.variant || 'v1_exact';
  const rateLimit = flags.rate_limit?.variant || 'strict';
  const cacheStrat = flags.cache_strategy?.variant || 'conservative';
  const premiumEnabled = flags.premium_api?.enabled || false;

  const samplePrices = applyPricing(PRODUCTS.slice(0, 3), pricingAlg);
  const sampleSearch = (searchVer === 'v2_fuzzy' ? searchFuzzy : searchExact)(PRODUCTS, 'wid');
  const rateCfg = RATE_LIMITS[rateLimit] || RATE_LIMITS.strict;
  const cacheCfg = CACHE_CONFIGS[cacheStrat] || CACHE_CONFIGS.conservative;

  const user = ctx.subjectId;
  const plan = ctx.attributes.plan;

  res.send(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Backend Feature Flags Dashboard</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f1117; color: #e1e4e8; line-height: 1.6; }
  .container { max-width: 1200px; margin: 0 auto; padding: 2rem 1.5rem; }
  header { text-align: center; margin-bottom: 2.5rem; display: flex; flex-direction: column; align-items: center; }
  header h1 { font-size: 2rem; font-weight: 700; background: linear-gradient(135deg, #f97316, #ef4444); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
  header p { color: #8b949e; font-size: 0.95rem; margin-top: 0.25rem; }
  .user-bar { display: flex; align-items: center; justify-content: center; gap: 0.75rem; flex-wrap: wrap; margin-bottom: 2rem; padding: 1rem; background: #161b22; border: 1px solid #30363d; border-radius: 12px; }
  .user-bar span { color: #8b949e; font-size: 0.85rem; }
  .user-bar a { display: inline-block; padding: 0.4rem 1rem; border-radius: 6px; text-decoration: none; font-size: 0.85rem; font-weight: 500; transition: all 0.15s; }
  .user-bar a.free { background: #21262d; color: #8b949e; border: 1px solid #30363d; }
  .user-bar a.premium { background: #1c1e3a; color: #a78bfa; border: 1px solid #5b21b6; }
  .user-bar a.enterprise { background: #1a2332; color: #58a6ff; border: 1px solid #1f6feb; }
  .user-bar a.active { box-shadow: 0 0 0 2px #f97316; }
  .user-bar a:hover { filter: brightness(1.2); }
  .config-panel { background: #161b22; border: 1px solid #30363d; border-radius: 12px; padding: 1.25rem; margin-bottom: 2rem; }
  .config-panel h2 { font-size: 1rem; font-weight: 600; color: #f0f6fc; margin-bottom: 0.75rem; display: flex; align-items: center; gap: 0.5rem; }
  .badges { display: flex; flex-wrap: wrap; gap: 0.5rem; }
  .badge { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.35rem 0.75rem; border-radius: 20px; font-size: 0.8rem; font-weight: 500; }
  .badge .dot { width: 8px; height: 8px; border-radius: 50%; }
  .badge-green { background: #0d1f0d; color: #3fb950; border: 1px solid #238636; }
  .badge-green .dot { background: #3fb950; }
  .badge-yellow { background: #1f1d0d; color: #d29922; border: 1px solid #9e6a03; }
  .badge-yellow .dot { background: #d29922; }
  .badge-red { background: #1f0d0d; color: #f85149; border: 1px solid #da3633; }
  .badge-red .dot { background: #f85149; }
  .badge-blue { background: #0d1525; color: #58a6ff; border: 1px solid #1f6feb; }
  .badge-blue .dot { background: #58a6ff; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 1.25rem; }
  .card { background: #161b22; border: 1px solid #30363d; border-radius: 12px; overflow: hidden; }
  .card-header { padding: 1rem 1.25rem; border-bottom: 1px solid #30363d; display: flex; align-items: center; justify-content: space-between; }
  .card-header h3 { font-size: 0.95rem; font-weight: 600; }
  .card-header .flag-name { font-size: 0.75rem; color: #8b949e; font-family: monospace; }
  .card-body { padding: 1.25rem; }
  .indicator { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.25rem 0.6rem; border-radius: 6px; font-size: 0.8rem; font-weight: 600; }
  .indicator-green { background: #0d1f0d; color: #3fb950; }
  .indicator-yellow { background: #1f1d0d; color: #d29922; }
  .indicator-red { background: #1f0d0d; color: #f85149; }
  .indicator-blue { background: #0d1525; color: #58a6ff; }
  .try-it { margin-top: 1rem; }
  .try-it-label { font-size: 0.75rem; color: #8b949e; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.5rem; font-weight: 600; }
  pre.json { background: #0d1117; border: 1px solid #21262d; border-radius: 8px; padding: 0.75rem; font-size: 0.78rem; line-height: 1.5; overflow-x: auto; font-family: 'SF Mono', Menlo, monospace; }
  .json .key { color: #79c0ff; }
  .json .str { color: #a5d6ff; }
  .json .num { color: #d2a8ff; }
  .json .bool-true { color: #3fb950; }
  .json .bool-false { color: #f85149; }
  .json .null { color: #8b949e; }
  .compare { margin-top: 0.75rem; padding: 0.6rem 0.75rem; background: #0d1117; border-radius: 8px; border-left: 3px solid #30363d; font-size: 0.8rem; color: #8b949e; }
  .compare strong { color: #e1e4e8; }
  .stat-row { display: flex; justify-content: space-between; padding: 0.4rem 0; border-bottom: 1px solid #21262d; font-size: 0.85rem; }
  .stat-row:last-child { border-bottom: none; }
  .stat-label { color: #8b949e; }
  .stat-value { color: #e1e4e8; font-weight: 500; font-family: monospace; }
  footer { text-align: center; margin-top: 3rem; color: #484f58; font-size: 0.8rem; }
</style>
</head>
<body>
<div class="container">
  <header>
    <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 512 512" fill="none"><g transform="rotate(-2 256 161)"><rect x="56" y="128" width="400" height="66" rx="33" fill="#DC2626"/><rect x="92" y="136" width="328" height="10" rx="5" fill="#FEE2E2"/><rect x="382" y="149" width="46" height="24" rx="12" fill="#22C55E"/><circle cx="416" cy="161" r="8" fill="#fff"/></g><g transform="rotate(0.8 256 256)"><rect x="56" y="223" width="400" height="66" rx="33" fill="#DC2626"/><rect x="92" y="231" width="328" height="10" rx="5" fill="#FEE2E2"/><rect x="382" y="244" width="46" height="24" rx="12" fill="#22C55E"/><circle cx="416" cy="256" r="8" fill="#fff"/></g><g transform="rotate(2.5 256 351)"><rect x="56" y="318" width="400" height="66" rx="33" fill="#DC2626" opacity="0.25"/><rect x="92" y="326" width="328" height="10" rx="5" fill="#FEE2E2" opacity="0.25"/><rect x="82" y="339" width="46" height="24" rx="12" fill="#9CA3AF"/><circle cx="94" cy="351" r="8" fill="#fff"/></g></svg>
    <h1>Backend Feature Flags</h1>
    <p>Real-time dashboard &mdash; see how flags change backend behavior for each user</p>
  </header>

  <div class="user-bar">
    <span>Switch user:</span>
    <a href="/?user=free_user&plan=free" class="free${plan === 'free' ? ' active' : ''}">Free User</a>
    <a href="/?user=premium_user&plan=premium" class="premium${plan === 'premium' ? ' active' : ''}">Premium User</a>
    <a href="/?user=enterprise_user&plan=enterprise" class="enterprise${plan === 'enterprise' ? ' active' : ''}">Enterprise User</a>
    <span style="margin-left:0.5rem;color:#484f58">Current: <strong style="color:#e1e4e8">${escapeHtml(user)}</strong> (${escapeHtml(plan)})</span>
  </div>

  <div class="config-panel">
    <h2>&#9881; Active Configuration</h2>
    <div class="badges">
      ${renderBadges(flags)}
    </div>
  </div>

  <div class="grid">
    ${renderPricingCard(pricingAlg, samplePrices)}

    ${renderSearchCard(searchVer, sampleSearch)}

    ${renderRateLimitCard(rateLimit, rateCfg)}

    ${renderCacheCard(cacheStrat, cacheCfg)}

    ${renderPremiumCard(premiumEnabled)}
  </div>

  <footer>Feature Bacon &mdash; Backend Feature Flags Demo</footer>
</div>
</body>
</html>`);
});

/* istanbul ignore next -- startup guard */
if (require.main === module) {
  app.listen(port, () => console.log(`Backend flags dashboard on :${port}`));
}

module.exports = app;
