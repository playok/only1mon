// IoTop Widget - Linux iotop-like per-process I/O monitor
class IoTopWidget {
    constructor(container, options = {}) {
        this.container = container;
        this.title = options.title || 'IoTop';
        this.type = 'iotop';
        this.live = options.live !== undefined ? options.live : true;
        this._topN = 0;
        this.metricNames = [];
        this.values = {};
        this.procNames = {};
        this._rebuildMetricNames();
        this._initDOM();
        this._bindWS();
    }

    _getTopN() {
        try {
            return Alpine.store('topProcessCount').value || 10;
        } catch (e) {
            return 10;
        }
    }

    _rebuildMetricNames() {
        const n = this._getTopN();
        if (n === this._topN) return false;
        this._topN = n;

        this.metricNames = [
            'proc.io.total_read_bps', 'proc.io.total_write_bps',
        ];
        for (let i = 0; i < n; i++) {
            this.metricNames.push(
                `proc.top_io.${i}.pid`,
                `proc.top_io.${i}.name`,
                `proc.top_io.${i}.read_bps`,
                `proc.top_io.${i}.write_bps`,
            );
        }
        return true;
    }

    _initDOM() {
        this.container.innerHTML = '';
        this.container.style.overflow = 'auto';
        this.container.style.fontFamily = 'var(--font-mono)';
        this.container.style.fontSize = '11px';
        this.container.style.lineHeight = '1.6';
        this.container.style.padding = '6px 8px';

        this.el = document.createElement('div');
        this.el.className = 'iotop-widget';
        this.container.appendChild(this.el);
        this._render();
    }

    _fmtRate(bps) {
        if (bps == null || isNaN(bps) || bps === 0) return '0 B/s';
        const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
        const i = Math.min(Math.floor(Math.log(Math.abs(bps)) / Math.log(1024)), units.length - 1);
        const val = bps / Math.pow(1024, i);
        return val.toFixed(i === 0 ? 0 : 1) + ' ' + units[i];
    }

    _bar(value, maxValue, color) {
        const pct = maxValue > 0 ? Math.min(100, (value / maxValue) * 100) : 0;
        return `<span class="iotop-bar"><span class="iotop-bar-fill" style="width:${pct.toFixed(0)}%;background:${color}"></span></span>`;
    }

    _escapeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str || '';
        return div.innerHTML;
    }

    _render() {
        const v = this.values;
        const totalRead = v['proc.io.total_read_bps'] || 0;
        const totalWrite = v['proc.io.total_write_bps'] || 0;

        // Collect process rows and find max for bar scaling
        const procs = [];
        let maxIO = 0;
        for (let i = 0; i < this._topN; i++) {
            const pid = v[`proc.top_io.${i}.pid`];
            if (pid == null) continue;
            const name = this.procNames[`proc.top_io.${i}.name`] || '—';
            const read = v[`proc.top_io.${i}.read_bps`] || 0;
            const write = v[`proc.top_io.${i}.write_bps`] || 0;
            const total = read + write;
            if (total > maxIO) maxIO = total;
            procs.push({ pid, name, read, write, total });
        }

        let procRows = '';
        for (const p of procs) {
            const readClass = p.read > 1048576 ? 'iotop-val-high' : '';
            const writeClass = p.write > 1048576 ? 'iotop-val-high' : '';
            procRows += `<tr>
                <td class="iotop-col-pid">${Math.round(p.pid)}</td>
                <td class="iotop-col-name">${this._escapeHtml(p.name)}</td>
                <td class="iotop-col-rate ${readClass}">${this._fmtRate(p.read)}</td>
                <td class="iotop-col-rate ${writeClass}">${this._fmtRate(p.write)}</td>
                <td class="iotop-col-bar">${this._bar(p.total, maxIO || 1, '#6C8EFF')}</td>
            </tr>`;
        }

        this.el.innerHTML = `
            <div class="iotop-summary">
                <div class="iotop-summary-row">
                    <span class="iotop-summary-label">Total READ</span>
                    <span class="iotop-summary-value iotop-read">${this._fmtRate(totalRead)}</span>
                    <span class="iotop-summary-sep">│</span>
                    <span class="iotop-summary-label">Total WRITE</span>
                    <span class="iotop-summary-value iotop-write">${this._fmtRate(totalWrite)}</span>
                </div>
            </div>
            <table class="iotop-table">
                <thead><tr>
                    <th class="iotop-col-pid">PID</th>
                    <th class="iotop-col-name">COMMAND</th>
                    <th class="iotop-col-rate">DISK READ</th>
                    <th class="iotop-col-rate">DISK WRITE</th>
                    <th class="iotop-col-bar">IO</th>
                </tr></thead>
                <tbody>${procRows || '<tr><td colspan="5" style="text-align:center;color:var(--text-muted)">—</td></tr>'}</tbody>
            </table>
        `;
    }

    _bindWS() {
        this._wsHandler = (samples) => {
            if (!this.live) return;
            // Check if topN changed and re-subscribe if needed
            if (this._rebuildMetricNames()) {
                window.wsClient.subscribe(this.metricNames);
            }

            let updated = false;
            for (const s of samples) {
                if (!this.metricNames.includes(s.metric_name)) continue;
                if (s.metric_name.endsWith('.name') && s.labels) {
                    this.procNames[s.metric_name] = s.labels;
                }
                this.values[s.metric_name] = s.value;
                updated = true;
            }
            if (updated) this._render();
        };
        window.wsClient.on('metrics', this._wsHandler);
        window.wsClient.subscribe(this.metricNames);
    }

    resize() { /* auto layout */ }

    destroy() {
        window.wsClient.off('metrics', this._wsHandler);
        window.wsClient.unsubscribe(this.metricNames);
        if (this.el && this.el.parentNode) {
            this.el.parentNode.removeChild(this.el);
        }
    }

    reloadWithRange(seconds, live) {
        this.live = live;
        this.loadHistory(seconds);
    }

    async loadHistory(seconds = 3600) {
        const now = Math.floor(Date.now() / 1000);
        const from = now - seconds;
        try {
            const samples = await API.queryMetrics(this.metricNames.join(','), from, now, 0);
            if (!samples || samples.length === 0) return;
            for (const s of samples) {
                if (s.metric_name.endsWith('.name') && s.labels) {
                    this.procNames[s.metric_name] = s.labels;
                }
                this.values[s.metric_name] = s.value;
            }
            this._render();
        } catch (e) {
            console.error('Failed to load iotop history:', e);
        }
    }
}

window.IoTopWidget = IoTopWidget;
