// Main Alpine.js app initialization
document.addEventListener('alpine:init', () => {
    // Toast notification system
    Alpine.data('toasts', () => ({
        list: [],
        nextId: 0,

        init() {
            window.addEventListener('toast', (e) => {
                this.add(e.detail.msg, e.detail.type || 'info');
            });
        },

        add(msg, type) {
            const id = this.nextId++;
            this.list.push({ id, msg, type });
        },

        remove(id) {
            this.list = this.list.filter(t => t.id !== id);
        },
    }));

    // Alert event grid
    Alpine.data('alertGrid', () => ({
        alerts: [],

        init() {
            // Load initial alerts
            this.fetchAlerts();

            // Listen for real-time alerts via WebSocket
            window.wsClient.on('alerts', (newAlerts) => {
                this.alerts = newAlerts;
            });
        },

        async fetchAlerts() {
            try {
                this.alerts = await API.getAlerts();
            } catch (e) {
                console.error('Failed to load alerts:', e);
            }
        },

        formatAlertTime(ts) {
            return new Date(ts * 1000).toLocaleTimeString();
        },
    }));

    // Chart series colors (shared store)
    const defaultColors = ['#6C8EFF','#3DD68C','#F5A623','#FF6B6B','#A78BFA','#F472B6','#56CCF2','#FB923C','#34D399','#F87171'];
    Alpine.store('chartColors', {
        list: [...defaultColors],
        defaults: defaultColors,
        async load() {
            try {
                const data = await API.getSettings();
                if (data.chart_colors) {
                    const parsed = JSON.parse(data.chart_colors);
                    if (Array.isArray(parsed) && parsed.length > 0) {
                        this.list = parsed;
                    }
                }
            } catch (e) { /* use defaults */ }
        },
    });
    Alpine.store('chartColors').load();

    // Top process count (shared store)
    Alpine.store('topProcessCount', {
        value: 10,
        async load() {
            try {
                const data = await API.getSettings();
                if (data.top_process_count) {
                    const n = parseInt(data.top_process_count, 10);
                    if (n > 0) this.value = n;
                }
            } catch (e) { /* use default */ }
        },
    });
    Alpine.store('topProcessCount').load();

    // Main app state
    Alpine.data('app', () => ({
        page: 'dashboard',
        wsConnected: false,

        init() {
            // Connect WebSocket
            window.wsClient.on('connection', (data) => {
                this.wsConnected = data.connected;
            });
            window.wsClient.connect();
        },
    }));
});
