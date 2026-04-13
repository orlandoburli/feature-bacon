const express = require('express');
const { BaconClient } = require('feature-bacon');

const app = express();
app.disable('x-powered-by');
const port = process.env.PORT || 3000;
const baconUrl = process.env.BACON_URL || 'http://localhost:8080';
const apiKey = process.env.BACON_API_KEY || '';

const client = new BaconClient(baconUrl, { apiKey });

const FLAG_KEYS = ['dark_mode', 'hero_banner', 'new_navigation', 'cta_experiment', 'maintenance_mode'];

function userContext(req) {
  const user = req.query.user || 'visitor';
  return {
    subjectId: user,
    environment: process.env.ENVIRONMENT || 'production',
    attributes: {
      theme_preference: user === 'alice' ? 'dark' : 'light',
    },
  };
}

function renderHeroBanner(variant) {
  if (variant === 'summer_sale') {
    return `
      <section class="hero hero--summer">
        <h2>Summer Sale — Up to 60% Off</h2>
        <p>Limited time offer on all premium plans. Don't miss out!</p>
      </section>`;
  }
  return `
    <section class="hero hero--default">
      <h2>Welcome to Feature Bacon</h2>
      <p>Ship features faster with confidence. Toggle anything, anytime.</p>
    </section>`;
}

function renderCtaButton(variant) {
  const styles = {
    bold: 'cta cta--bold',
    minimal: 'cta cta--minimal',
    classic: 'cta cta--classic',
  };
  const labels = {
    bold: 'GET STARTED NOW',
    minimal: 'Get started',
    classic: 'Get Started Free',
  };
  const cls = styles[variant] || styles.classic;
  const label = labels[variant] || labels.classic;
  return `<button class="${cls}">${label}</button>`;
}

function renderNavigation(isSidebar, currentUser) {
  const users = ['alice', 'bob', 'visitor'];
  const userLinks = users.map(u => {
    const active = u === currentUser ? ' class="active"' : '';
    return `<a href="?user=${u}"${active}>${u}</a>`;
  }).join('');

  const navItems = `
    <a href="?user=${currentUser}">Dashboard</a>
    <a href="?user=${currentUser}">Products</a>
    <a href="?user=${currentUser}">Analytics</a>
    <a href="?user=${currentUser}">Settings</a>`;

  if (isSidebar) {
    return `
      <nav class="sidebar">
        <div class="sidebar__brand">Feature Bacon</div>
        ${navItems}
        <div class="sidebar__divider"></div>
        <div class="sidebar__section">Switch User</div>
        ${userLinks}
      </nav>`;
  }

  return `
    <nav class="topnav">
      <div class="topnav__brand">Feature Bacon</div>
      <div class="topnav__links">${navItems}</div>
      <div class="topnav__users">${userLinks}</div>
    </nav>`;
}

function renderFlagPanel(flags) {
  const rows = Object.entries(flags).map(([key, val]) => {
    const dot = val.enabled ? 'dot dot--on' : 'dot dot--off';
    const variant = val.variant ? `<span class="flag-variant">${val.variant}</span>` : '';
    return `
      <div class="flag-row">
        <span class="${dot}"></span>
        <span class="flag-key">${key}</span>
        ${variant}
      </div>`;
  }).join('');

  return `
    <aside class="flag-panel">
      <h3>Flag Status</h3>
      ${rows}
    </aside>`;
}

function renderProducts(ctaVariant) {
  const products = [
    { name: 'Starter Plan', price: '$9/mo', desc: 'Perfect for side projects' },
    { name: 'Pro Plan', price: '$29/mo', desc: 'For growing teams' },
    { name: 'Enterprise', price: '$99/mo', desc: 'Unlimited everything' },
  ];

  const cards = products.map(p => `
    <div class="card">
      <h3 class="card__title">${p.name}</h3>
      <div class="card__price">${p.price}</div>
      <p class="card__desc">${p.desc}</p>
      ${renderCtaButton(ctaVariant)}
    </div>`).join('');

  return `<section class="products">${cards}</section>`;
}

function renderPage(flags, currentUser) {
  const darkMode = flags.dark_mode.enabled;
  const heroVariant = flags.hero_banner.variant || 'default';
  const sidebarNav = flags.new_navigation.enabled;
  const ctaVariant = flags.cta_experiment.variant || 'classic';
  const maintenance = flags.maintenance_mode.enabled;

  const themeClass = darkMode ? 'theme-dark' : 'theme-light';

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Feature Bacon — UI Demo</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    :root {
      --bg: #f5f7fa;
      --text: #1a1a2e;
      --card-bg: #ffffff;
      --card-border: #e2e8f0;
      --muted: #64748b;
      --accent: #6366f1;
      --radius: 12px;
      --shadow: 0 1px 3px rgba(0,0,0,0.08), 0 4px 12px rgba(0,0,0,0.04);
      --transition: 0.3s ease;
    }

    .theme-dark {
      --bg: #1a1a2e;
      --text: #e2e8f0;
      --card-bg: #16213e;
      --card-border: #2d3a5c;
      --muted: #94a3b8;
      --accent: #818cf8;
      --shadow: 0 1px 3px rgba(0,0,0,0.3), 0 4px 12px rgba(0,0,0,0.2);
    }

    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: var(--bg);
      color: var(--text);
      transition: background var(--transition), color var(--transition);
      min-height: 100vh;
      line-height: 1.6;
    }

    .maintenance-overlay {
      position: fixed; inset: 0; z-index: 9999;
      background: rgba(15, 23, 42, 0.95);
      display: flex; align-items: center; justify-content: center;
      flex-direction: column; color: #fff;
    }
    .maintenance-overlay h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
    .maintenance-overlay p { font-size: 1.2rem; opacity: 0.8; }

    .topnav {
      display: flex; align-items: center; gap: 1rem;
      padding: 0.75rem 1.5rem;
      background: var(--card-bg); border-bottom: 1px solid var(--card-border);
      box-shadow: var(--shadow);
      transition: background var(--transition);
    }
    .topnav__brand {
      font-weight: 700; font-size: 1.1rem; color: var(--accent);
      margin-right: auto;
    }
    .topnav__links { display: flex; gap: 0.75rem; }
    .topnav__links a, .topnav__users a {
      text-decoration: none; color: var(--muted); font-size: 0.9rem;
      padding: 0.35rem 0.7rem; border-radius: 6px;
      transition: background var(--transition), color var(--transition);
    }
    .topnav__links a:hover, .topnav__users a:hover { background: var(--accent); color: #fff; }
    .topnav__users { display: flex; gap: 0.5rem; margin-left: 1rem; }
    .topnav__users a.active {
      background: var(--accent); color: #fff; font-weight: 600;
    }

    .sidebar {
      position: fixed; left: 0; top: 0; bottom: 0; width: 220px;
      background: var(--card-bg); border-right: 1px solid var(--card-border);
      padding: 1.5rem 1rem; display: flex; flex-direction: column; gap: 0.25rem;
      box-shadow: var(--shadow);
      transition: background var(--transition);
    }
    .sidebar__brand {
      font-weight: 700; font-size: 1.2rem; color: var(--accent);
      margin-bottom: 1.5rem;
    }
    .sidebar a {
      text-decoration: none; color: var(--muted); font-size: 0.9rem;
      padding: 0.5rem 0.75rem; border-radius: 8px;
      transition: background var(--transition), color var(--transition);
    }
    .sidebar a:hover { background: var(--accent); color: #fff; }
    .sidebar a.active { background: var(--accent); color: #fff; font-weight: 600; }
    .sidebar__divider {
      height: 1px; background: var(--card-border); margin: 1rem 0;
    }
    .sidebar__section {
      font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em;
      color: var(--muted); padding: 0 0.75rem; margin-bottom: 0.25rem;
    }

    .layout { display: flex; min-height: 100vh; }
    .layout--sidebar .main { margin-left: 220px; }
    .main {
      flex: 1; padding: 2rem; max-width: 960px;
      margin-left: auto; margin-right: auto;
      transition: margin var(--transition);
    }
    .layout--sidebar .main { margin-right: 280px; }
    .layout--topnav .main { margin-right: 280px; }

    .header {
      text-align: center; margin-bottom: 2rem;
      padding: 1.5rem; border-radius: var(--radius);
      background: var(--card-bg); border: 1px solid var(--card-border);
      box-shadow: var(--shadow);
      transition: background var(--transition);
    }
    .header h1 { font-size: 1.5rem; margin-bottom: 0.25rem; }
    .header p { color: var(--muted); font-size: 0.95rem; }
    .header .user-badge {
      display: inline-block; margin-top: 0.75rem;
      padding: 0.25rem 0.75rem; border-radius: 20px;
      background: var(--accent); color: #fff; font-size: 0.85rem; font-weight: 600;
    }

    .hero {
      border-radius: var(--radius); padding: 2.5rem 2rem;
      margin-bottom: 2rem; color: #fff; text-align: center;
      box-shadow: var(--shadow);
    }
    .hero h2 { font-size: 1.8rem; margin-bottom: 0.5rem; }
    .hero p { font-size: 1.1rem; opacity: 0.9; }
    .hero--summer {
      background: linear-gradient(135deg, #f97316, #ec4899);
    }
    .hero--default {
      background: linear-gradient(135deg, #3b82f6, #6366f1);
    }

    .products {
      display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: 1.5rem; margin-bottom: 2rem;
    }
    .card {
      background: var(--card-bg); border: 1px solid var(--card-border);
      border-radius: var(--radius); padding: 1.5rem;
      box-shadow: var(--shadow); text-align: center;
      transition: transform 0.2s, box-shadow 0.2s, background var(--transition);
    }
    .card:hover { transform: translateY(-4px); box-shadow: 0 8px 24px rgba(0,0,0,0.12); }
    .card__title { font-size: 1.1rem; margin-bottom: 0.5rem; }
    .card__price {
      font-size: 2rem; font-weight: 700; color: var(--accent); margin-bottom: 0.5rem;
    }
    .card__desc { color: var(--muted); font-size: 0.9rem; margin-bottom: 1.25rem; }

    .cta {
      cursor: pointer; border: none; font-size: 0.95rem;
      padding: 0.65rem 1.5rem; border-radius: 8px;
      transition: transform 0.15s, box-shadow 0.15s;
    }
    .cta:hover { transform: scale(1.05); }
    .cta--classic {
      background: linear-gradient(135deg, #3b82f6, #6366f1);
      color: #fff; border-radius: 24px;
    }
    .cta--bold {
      background: linear-gradient(135deg, #ef4444, #dc2626);
      color: #fff; font-size: 1.1rem; font-weight: 800;
      text-transform: uppercase; letter-spacing: 0.05em;
      padding: 0.85rem 2rem; border-radius: 8px;
    }
    .cta--minimal {
      background: transparent; color: var(--accent);
      border: 2px solid var(--accent); border-radius: 8px;
    }

    .flag-panel {
      position: fixed; right: 0; top: 0; bottom: 0; width: 260px;
      background: var(--card-bg); border-left: 1px solid var(--card-border);
      padding: 1.5rem 1rem; overflow-y: auto;
      box-shadow: var(--shadow);
      transition: background var(--transition);
    }
    .flag-panel h3 {
      font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.08em;
      color: var(--muted); margin-bottom: 1rem;
    }
    .flag-row {
      display: flex; align-items: center; gap: 0.5rem;
      padding: 0.5rem 0; border-bottom: 1px solid var(--card-border);
    }
    .dot {
      width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
    }
    .dot--on { background: #22c55e; box-shadow: 0 0 6px rgba(34,197,94,0.5); }
    .dot--off { background: #ef4444; box-shadow: 0 0 6px rgba(239,68,68,0.4); }
    .flag-key { font-size: 0.85rem; font-weight: 500; font-family: monospace; }
    .flag-variant {
      margin-left: auto; font-size: 0.75rem; padding: 0.15rem 0.5rem;
      background: var(--accent); color: #fff; border-radius: 10px;
    }

    .footer {
      text-align: center; padding: 2rem; color: var(--muted); font-size: 0.85rem;
    }

    @media (max-width: 768px) {
      .sidebar { position: static; width: 100%; }
      .layout--sidebar .main { margin-left: 0; }
      .flag-panel { position: static; width: 100%; }
      .layout--sidebar .main, .layout--topnav .main { margin-right: 0; }
      .products { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body class="${themeClass}">
  ${maintenance ? `
  <div class="maintenance-overlay">
    <h1>Under Maintenance</h1>
    <p>We're performing scheduled maintenance. Please check back soon.</p>
  </div>` : ''}

  <div class="layout ${sidebarNav ? 'layout--sidebar' : 'layout--topnav'}">
    ${renderNavigation(sidebarNav, currentUser)}

    <main class="main">
      <div class="header">
        <h1>UI Feature Flags Demo</h1>
        <p>This page is controlled by Feature Bacon feature flags</p>
        <span class="user-badge">User: ${currentUser}</span>
      </div>

      ${renderHeroBanner(heroVariant)}
      ${renderProducts(ctaVariant)}

      <div class="footer">
        Powered by Feature Bacon &middot; Switch users above to see different flag evaluations
      </div>
    </main>

    ${renderFlagPanel(flags)}
  </div>
</body>
</html>`;
}

app.get('/', async (req, res) => {
  const ctx = userContext(req);
  try {
    const results = await client.evaluateBatch(FLAG_KEYS, ctx);
    const flags = {};
    results.forEach(r => {
      flags[r.flagKey] = { enabled: r.enabled, variant: r.variant };
    });
    res.type('html').send(renderPage(flags, ctx.subjectId));
  } catch (err) {
    res.status(500).send(`<h1>Error</h1><p>${err.message}</p>`);
  }
});

app.get('/health', async (_req, res) => {
  const healthy = await client.healthy();
  const status = healthy ? 'ok' : 'degraded';
  if (!healthy) res.status(503);
  res.json({ status, baconHealthy: healthy });
});

app.get('/api/flags', async (req, res) => {
  const ctx = userContext(req);
  try {
    const results = await client.evaluateBatch(FLAG_KEYS, ctx);
    const flags = {};
    results.forEach(r => {
      flags[r.flagKey] = { enabled: r.enabled, variant: r.variant, reason: r.reason };
    });
    res.json({ user: ctx.subjectId, flags });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

/* istanbul ignore next */
if (require.main === module) {
  app.listen(port, () => console.log(`UI Feature Flags demo on :${port}`));
}

module.exports = app;
