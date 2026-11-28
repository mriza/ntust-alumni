// theme.js - Early Light / Dark / System theme controller (zero inline styles)
(function (root) {
    'use strict';

    var STORAGE_KEY = 'themePreference';
    var VALID_PREFS = { light: true, dark: true, system: true };

    function getPreference() {
        try {
            var saved = root.localStorage ? root.localStorage.getItem(STORAGE_KEY) : null;
            if (saved && VALID_PREFS[saved]) {
                return saved;
            }
        } catch (_) {}
        return 'system';
    }

    function getSystemTheme() {
        try {
            if (typeof root.matchMedia === 'function') {
                return root.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
            }
        } catch (_) {}
        return 'light';
    }

    function getResolvedTheme(pref) {
        var activePref = pref && VALID_PREFS[pref] ? pref : getPreference();
        return activePref === 'system' ? getSystemTheme() : activePref;
    }

    function notifyChange(pref) {
        if (typeof root.dispatchEvent === 'function' && typeof root.CustomEvent === 'function') {
            root.dispatchEvent(new root.CustomEvent('ntust:themechange', {
                detail: {
                    preference: pref,
                    resolved: getResolvedTheme(pref)
                }
            }));
        }
    }

    function applyTheme(pref) {
        var normalized = VALID_PREFS[pref] ? pref : 'system';
        var doc = root.document;
        if (doc && doc.documentElement) {
            if (normalized === 'light' || normalized === 'dark') {
                doc.documentElement.setAttribute('data-theme', normalized);
            } else {
                doc.documentElement.removeAttribute('data-theme');
            }
            doc.documentElement.setAttribute('data-theme-preference', normalized);
        }
        return normalized;
    }

    function setPreference(pref) {
        var normalized = applyTheme(pref);
        try {
            if (root.localStorage) {
                root.localStorage.setItem(STORAGE_KEY, normalized);
            }
        } catch (_) {}
        notifyChange(normalized);
        return normalized;
    }

    // Apply immediately before first paint to prevent FOUC
    applyTheme(getPreference());

    // Keep system preference reactive when OS appearance changes
    try {
        if (typeof root.matchMedia === 'function') {
            var mq = root.matchMedia('(prefers-color-scheme: dark)');
            if (mq && typeof mq.addEventListener === 'function') {
                mq.addEventListener('change', function () {
                    if (getPreference() === 'system') {
                        notifyChange('system');
                    }
                });
            }
        }
    } catch (_) {}

    root.ntustTheme = {
        getPreference: getPreference,
        setPreference: setPreference,
        getResolvedTheme: getResolvedTheme,
        applyTheme: applyTheme
    };
})(typeof window !== 'undefined' ? window : globalThis);
