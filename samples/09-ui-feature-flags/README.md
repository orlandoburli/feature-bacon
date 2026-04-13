# 09 — UI Feature Flags Demo

A visual web application demonstrating Feature Bacon feature flags controlling UI elements in real-time. Built with Express.js and the Feature Bacon JavaScript SDK, this sample serves server-rendered HTML pages where every visual element is driven by feature flags.

## What This Demo Shows

- **Dark mode toggle** — full theme switch between light and dark via the `dark_mode` flag
- **Hero banner variants** — a summer sale gradient or a default blue banner controlled by `hero_banner`
- **Navigation layout** — horizontal top nav vs vertical sidebar via `new_navigation`
- **CTA button A/B/C test** — classic (blue rounded), bold (red uppercase), or minimal (outline) styles via `cta_experiment`
- **Maintenance overlay** — full-screen blocking overlay when `maintenance_mode` is enabled
- **Flag status panel** — a fixed right-side panel showing all flag states with green/red indicators

## What You'll See

Open the app in your browser and you'll see a modern, card-based product page. The right panel shows every flag's current state. Switch between users using the links at the top to see how different users get different flag evaluations based on rollout percentages and targeting rules.

- **Alice** — gets dark mode (her `theme_preference` attribute is `"dark"`)
- **Bob** — gets light mode, may see different rollout results
- **Visitor** — the default anonymous user experience

## Quick Start

```bash
docker compose up --build
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

Try switching users: [alice](http://localhost:3000/?user=alice) · [bob](http://localhost:3000/?user=bob) · [visitor](http://localhost:3000/?user=visitor)

## Running Without Docker

```bash
# Start the Feature Bacon sidecar (must be running on port 8080)
# Then:
npm install
npm start
```

## Running Tests

```bash
# Unit tests
npm test

# Integration tests (requires running app)
bash test.sh
```

## Endpoints

| Endpoint | Description |
|---|---|
| `GET /` | HTML page with all flags visualized |
| `GET /health` | Health check (includes SDK connectivity) |
| `GET /api/flags` | Raw JSON flag evaluations for debugging |

All endpoints accept `?user=` to specify the evaluation subject.

## Modifying Flags

Edit `flags.yaml` and restart the sidecar to see changes reflected in the UI. Try:

- Setting `maintenance_mode.enabled: true` to see the overlay
- Changing `cta_experiment` rollout percentages
- Adjusting `new_navigation` rollout to 100 to give everyone the sidebar

## Project Structure

```
├── app.js              Express server with inline HTML rendering
├── app.test.js         Jest + supertest unit tests
├── flags.yaml          Feature flag definitions
├── package.json        Dependencies
├── Dockerfile          Multi-stage container build
├── docker-compose.yaml Bacon sidecar + app services
├── test.sh             Integration test script
└── README.md           This file
```
