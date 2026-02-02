// Table Widget - displays metrics as a real-time updating grid/table
class TableWidget {
    constructor(container, options = {}) {
        this.container = container;
        this.title = options.title || 'Table';
        this.metricNames = options.metrics || [];
        this.type = 'table'; // distinguish from ChartWidget
        this.values = {}; // metric_name → latest value
        this.metricMeta = options.metricMeta || {}; // metric_name → { description, description_ko, unit }
        this.live = options.live !== undefined ? options.live : true;
        this.table = null;
        this.rows = {};

        this._initTable();
        this._bindWS();
    }

    _initTable() {
        this.container.innerHTML = '';
        this.container.style.overflow = 'auto';

        const table = document.createElement('table');
        table.className = 'table-widget';

        const thead = document.createElement('thead');
        const lang = (window.Alpine && Alpine.store('i18n')) ? Alpine.store('i18n').lang : 'en';
        thead.innerHTML = `<tr>
            <th class="tw-col-name">${lang === 'ko' ? '메트릭' : 'Metric'}</th>
            <th class="tw-col-value">${lang === 'ko' ? '값' : 'Value'}</th>
            <th class="tw-col-unit">${lang === 'ko' ? '단위' : 'Unit'}</th>
        </tr>`;
        table.appendChild(thead);

        const tbody = document.createElement('tbody');
        for (const name of this.metricNames) {
            const tr = document.createElement('tr');
            const meta = this.metricMeta[name] || {};
            const shortName = name.split('.').slice(1).join('.');

            tr.innerHTML = `
                <td class="tw-cell-name" title="${this._escapeHtml(name)}">
                    <span class="tw-metric-name">${this._escapeHtml(shortName)}</span>
                </td>
                <td class="tw-cell-value">
                    <span class="tw-value" data-metric="${this._escapeHtml(name)}">—</span>
                </td>
                <td class="tw-cell-unit">${this._escapeHtml(meta.unit || '')}</td>
            `;
            tbody.appendChild(tr);
            this.rows[name] = tr;
        }
        table.appendChild(tbody);

        this.container.appendChild(table);
        this.table = table;
    }

    _escapeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    _formatValue(value, unit) {
        if (value == null) return '—';
        if (unit === 'bytes' || unit === 'bytes/s') {
            return this._formatBytes(value, unit === 'bytes/s');
        }
        if (unit === '%') return value.toFixed(1) + '%';
        if (unit === 'us') {
            if (value >= 1000000) return (value / 1000000).toFixed(2) + 's';
            if (value >= 1000) return (value / 1000).toFixed(1) + 'ms';
            return value.toFixed(0) + 'μs';
        }
        if (unit === 'ms') {
            if (value >= 1000) return (value / 1000).toFixed(2) + 's';
            return value.toFixed(1) + 'ms';
        }
        if (Number.isInteger(value)) return value.toLocaleString();
        return value.toFixed(2);
    }

    _formatBytes(bytes, perSec) {
        const suffix = perSec ? '/s' : '';
        if (bytes === 0) return '0 B' + suffix;
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.min(Math.floor(Math.log(Math.abs(bytes)) / Math.log(1024)), units.length - 1);
        const val = bytes / Math.pow(1024, i);
        return val.toFixed(i === 0 ? 0 : 1) + ' ' + units[i] + suffix;
    }

    _bindWS() {
        this._wsHandler = (samples) => {
            if (!this.live) return;
            let updated = false;
            for (const s of samples) {
                if (this.rows[s.metric_name] === undefined) continue;
                this.values[s.metric_name] = s.value;
                updated = true;
            }
            if (updated) this._renderValues();
        };

        window.wsClient.on('metrics', this._wsHandler);
        window.wsClient.subscribe(this.metricNames);
    }

    _renderValues() {
        for (const name of this.metricNames) {
            const el = this.container.querySelector(`[data-metric="${CSS.escape(name)}"]`);
            if (!el) continue;
            const meta = this.metricMeta[name] || {};
            const value = this.values[name];
            const prev = el.textContent;
            const next = this._formatValue(value, meta.unit);
            if (prev !== next) {
                el.textContent = next;
                // Flash animation on value change
                el.classList.remove('tw-flash');
                void el.offsetWidth; // reflow
                el.classList.add('tw-flash');
            }
        }
    }

    resize() {
        // Table auto-fills, nothing to do
    }

    destroy() {
        window.wsClient.off('metrics', this._wsHandler);
        window.wsClient.unsubscribe(this.metricNames);
        if (this.table && this.table.parentNode) {
            this.table.parentNode.removeChild(this.table);
        }
    }

    reloadWithRange(from, to, live) {
        this.live = live;
        this.loadHistory(from, to);
    }

    async loadHistory(from, to) {
        if (from === undefined) {
            const now = Math.floor(Date.now() / 1000);
            from = now - 3600;
            to = now;
        }
        try {
            const samples = await API.queryMetrics(this.metricNames.join(','), from, to, 0);
            if (!samples || samples.length === 0) return;
            // Take the latest value per metric
            for (const s of samples) {
                if (this.rows[s.metric_name] !== undefined) {
                    this.values[s.metric_name] = s.value;
                }
            }
            this._renderValues();
        } catch (e) {
            console.error('Failed to load table history:', e);
        }
    }
}

window.TableWidget = TableWidget;
