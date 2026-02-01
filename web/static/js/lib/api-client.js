// API Client - thin wrapper around fetch
const API = {
    base: ((window.__BASE_PATH && window.__BASE_PATH !== '/') ? window.__BASE_PATH : '') + '/api/v1',

    async get(path) {
        const res = await fetch(this.base + path);
        if (!res.ok) throw new Error(`GET ${path}: ${res.status}`);
        return res.json();
    },

    async put(path, body) {
        const res = await fetch(this.base + path, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        if (!res.ok) throw new Error(`PUT ${path}: ${res.status}`);
        return res.json();
    },

    async post(path, body) {
        const res = await fetch(this.base + path, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        if (!res.ok) throw new Error(`POST ${path}: ${res.status}`);
        return res.json();
    },

    async del(path) {
        const res = await fetch(this.base + path, { method: 'DELETE' });
        if (!res.ok) throw new Error(`DELETE ${path}: ${res.status}`);
        return res.json();
    },

    // Collectors
    getCollectors() { return this.get('/collectors'); },
    enableCollector(id) { return this.put(`/collectors/${id}/enable`); },
    disableCollector(id) { return this.put(`/collectors/${id}/disable`); },

    enableMetric(name) { return this.put(`/metrics/state/${name}/enable`); },
    disableMetric(name) { return this.put(`/metrics/state/${name}/disable`); },
    enableCollectorMetrics(id) { return this.put(`/collectors/${id}/metrics/enable`); },
    disableCollectorMetrics(id) { return this.put(`/collectors/${id}/metrics/disable`); },
    ensureMetricsEnabled(metrics) { return this.put('/metrics/ensure-enabled', { metrics }); },

    // Alerts
    getAlerts() { return this.get('/alerts'); },

    // Alert Rules
    getAlertRules() { return this.get('/alert-rules'); },
    createAlertRule(data) { return this.post('/alert-rules', data); },
    updateAlertRule(id, data) { return this.put(`/alert-rules/${id}`, data); },
    deleteAlertRule(id) { return this.del(`/alert-rules/${id}`); },

    // Metrics
    getAvailableMetrics() { return this.get('/metrics/available'); },
    queryMetrics(name, from, to, step) {
        const params = new URLSearchParams({ name });
        if (from) params.set('from', from);
        if (to) params.set('to', to);
        if (step) params.set('step', step);
        return this.get('/metrics/query?' + params.toString());
    },

    // Settings
    getSettings() { return this.get('/settings'); },
    updateSettings(data) { return this.put('/settings', data); },
    getDBInfo() { return this.get('/settings/db-info'); },
    purgeDB() { return this.del('/settings/db-purge'); },

    // Dashboard layouts
    getLayouts() { return this.get('/dashboard/layouts'); },
    getLayout(id) { return this.get(`/dashboard/layouts/${id}`); },
    createLayout(data) { return this.post('/dashboard/layouts', data); },
    updateLayout(id, data) { return this.put(`/dashboard/layouts/${id}`, data); },
    deleteLayout(id) { return this.del(`/dashboard/layouts/${id}`); },
};
