# Frontend Ant Design v5 Redesign, Landing Page, & Multi-Theme Support Design

**Date:** 2026-11-28  
**Repository:** `/home/mriza/Projects/ntust-alumni`

## 1. Goals
1. **Ant Design v5 UI/UX Overhaul**: Replace legacy generic styles with clean Ant Design v5 hierarchy, spacing, cards, descriptions, segmented controls, steps, tags, status badges, floating toast notifications, and accessible focus rings.
2. **Pure HTML5 + CSS3 + Animation**: Use semantic HTML5 elements (`<header>`, `<nav>`, `<main>`, `<section>`, `<article>`, `<footer>`) and pure CSS3 custom properties + keyframe animations (`@keyframes antFadeUp`, `antFloat`, `antPulseDot`, `antShimmer`) with `prefers-reduced-motion` support and zero runtime build dependencies.
3. **Full Landing Page (`#/`)**: Present an inviting first-page experience featuring an animated Hero section with dual CTAs, Alumni Network overview cards, 4 key metrics strip, 4-feature Ant Card showcase, 3-step onboarding flow, Indonesia regional chapters showcase, and joining CTA banner.
4. **Light / Dark / System Theme Switcher**: Allow users to toggle smoothly between **Light**, **Dark**, and **System** (`prefers-color-scheme`) preferences, persisted in `localStorage` under `themePreference`, applied pre-paint via `<head>` script without FOUC.
5. **Zero Hardcoded Styles**: Eliminate 100% of inline `style="..."` attributes in HTML/JS, `.style.*` mutations in JS, and literal colors/radii/shadows in `style.css` — allowing Nginx CSP to enforce `style-src 'self'` without `'unsafe-inline'`.

## 2. Approaches Considered
- **Option A — Pure HTML5 + CSS3 Ant Design v5 Token System (Selected)**: Lightweight, zero npm build complexity, keeps existing vanilla SPA routes & security controls intact, satisfies user request to prioritize HTML5, CSS3 + animation, and enables strict `style-src 'self'` CSP.
- **Option B — React + Ant Design (`antd`)**: Requires adding Node/Vite build tooling and rewriting all tested DOM handlers; unnecessarily heavy for the current Go + Nginx stack.
- **Option C — CDN UI Framework**: Introduces third-party external origins, weakening CSP and offline reliability.

## 3. Architecture
- [`frontend/css/tokens.css`](../../../frontend/css/tokens.css): Single source of truth for all `--ant-*` design tokens across `:root`/`[data-theme="light"]`, `[data-theme="dark"]`, and `@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) }`.
- [`frontend/css/style.css`](../../../frontend/css/style.css): Component & layout rules referencing strictly `var(--ant-...)` tokens (zero raw hex/rgb literals), plus CSS3 keyframe animations and responsive rules.
- [`frontend/js/theme.js`](../../../frontend/js/theme.js): Loaded early in `<head>` to set `data-theme` / `data-theme-preference` before first paint and listen to `matchMedia` changes.
- [`frontend/index.html`](../../../frontend/index.html): Semantic shell with sticky frosted header, `#themeSwitcher` segmented control, `#toastContainer`, `<main id="app">`, and `<footer>`.
- [`frontend/js/app.js`](../../../frontend/js/app.js): Landing page + Ant Design-styled SPA views preserving every security helper (`esc`, `safeUrl`, `safeImageDataUrl`) and OAuth2 consent flow.
- [`frontend/nginx.conf`](../../../frontend/nginx.conf): Tightened `Content-Security-Policy` removing `'unsafe-inline'` from `style-src`.
