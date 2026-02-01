// WebSocket client with auto-reconnect and subscription management
class WSClient {
    constructor() {
        this.ws = null;
        this.listeners = new Map();
        this.connected = false;
        this.reconnectTimer = null;
        this.subscriptions = new Set();
    }

    connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const bp = (window.__BASE_PATH && window.__BASE_PATH !== '/') ? window.__BASE_PATH : '';
        const url = `${proto}//${location.host}${bp}/api/v1/ws`;

        try {
            this.ws = new WebSocket(url);
        } catch (e) {
            this._scheduleReconnect();
            return;
        }

        this.ws.onopen = () => {
            this.connected = true;
            this._notify('connection', { connected: true });
            // Resubscribe
            if (this.subscriptions.size > 0) {
                this.ws.send(JSON.stringify({
                    type: 'subscribe',
                    metrics: [...this.subscriptions],
                }));
            }
        };

        this.ws.onclose = () => {
            this.connected = false;
            this._notify('connection', { connected: false });
            this._scheduleReconnect();
        };

        this.ws.onerror = () => {
            this.ws.close();
        };

        this.ws.onmessage = (evt) => {
            try {
                const data = JSON.parse(evt.data);
                if (data.type === 'metrics' && data.samples) {
                    this._notify('metrics', data.samples);
                } else if (data.type === 'alerts' && data.alerts) {
                    this._notify('alerts', data.alerts);
                }
            } catch (e) {
                // ignore parse errors
            }
        };
    }

    subscribe(metricPrefixes) {
        metricPrefixes.forEach(m => this.subscriptions.add(m));
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'subscribe',
                metrics: metricPrefixes,
            }));
        }
    }

    unsubscribe(metricPrefixes) {
        metricPrefixes.forEach(m => this.subscriptions.delete(m));
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'unsubscribe',
                metrics: metricPrefixes,
            }));
        }
    }

    on(event, callback) {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, []);
        }
        this.listeners.get(event).push(callback);
    }

    off(event, callback) {
        const list = this.listeners.get(event);
        if (list) {
            const idx = list.indexOf(callback);
            if (idx >= 0) list.splice(idx, 1);
        }
    }

    _notify(event, data) {
        const list = this.listeners.get(event);
        if (list) list.forEach(fn => fn(data));
    }

    _scheduleReconnect() {
        if (this.reconnectTimer) return;
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
        }, 2000);
    }

    disconnect() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}

// Global singleton
window.wsClient = new WSClient();
