// Top Widget - Linux top-like system overview with process list
class TopWidget {
    constructor(container, options = {}) {
        this.container = container;
        this.title = options.title || 'System Top';
        this.type = 'top';
        this._topN = 0;
        this.metricNames = [];
        this.values = {};
        this.procNames = {}; // pid metric → name (from labels)
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
            'cpu.total.usage', 'cpu.total.user', 'cpu.total.system', 'cpu.total.iowait', 'cpu.total.idle',
            'cpu.load.1', 'cpu.load.5', 'cpu.load.15',
            'mem.total', 'mem.used', 'mem.available', 'mem.cached', 'mem.buffers',
            'mem.swap.total', 'mem.swap.used',
            'mem.used_pct',
            'proc.total_count',
        ];
        for (let i = 0; i < n; i++) {
            this.metricNames.push(
                `proc.top_cpu.${i}.pid`,
                `proc.top_cpu.${i}.name`,
                `proc.top_cpu.${i}.cpu_pct`,
                `proc.top_cpu.${i}.mem_pct`,
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
        this.el.className = 'top-widget';
        this.container.appendChild(this.el);
        this._render();
    }

    _fmt(v, decimals) {
        if (v == null || isNaN(v)) return '—';
        return v.toFixed(decimals !== undefined ? decimals : 1);
    }

    _fmtBytes(bytes) {
        if (bytes == null || isNaN(bytes)) return '—';
        if (bytes === 0) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.min(Math.floor(Math.log(Math.abs(bytes)) / Math.log(1024)), units.length - 1);
        const val = bytes / Math.pow(1024, i);
        return val.toFixed(i === 0 ? 0 : 1) + ' ' + units[i];
    }

    _bar(pct, color) {
        const p = Math.min(100, Math.max(0, pct || 0));
        const w = Math.round(p);
        const empty = 100 - w;
        return `<span class="top-bar"><span class="top-bar-fill" style="width:${w}%;background:${color}"></span><span class="top-bar-empty" style="width:${empty}%"></span></span><span class="top-bar-pct">${this._fmt(pct, 1)}%</span>`;
    }

    _render() {
        const v = this.values;
        const load1 = this._fmt(v['cpu.load.1'], 2);
        const load5 = this._fmt(v['cpu.load.5'], 2);
        const load15 = this._fmt(v['cpu.load.15'], 2);
        const tasks = v['proc.total_count'] != null ? Math.round(v['proc.total_count']) : '—';

        // CPU breakdown
        const cpuUser = v['cpu.total.user'] || 0;
        const cpuSys = v['cpu.total.system'] || 0;
        const cpuIo = v['cpu.total.iowait'] || 0;
        const cpuIdle = v['cpu.total.idle'] || 0;
        const cpuUsage = v['cpu.total.usage'] || 0;

        // Memory
        const memTotal = v['mem.total'] || 0;
        const memUsed = v['mem.used'] || 0;
        const memAvail = v['mem.available'] || 0;
        const memCached = v['mem.cached'] || 0;
        const memBuffers = v['mem.buffers'] || 0;
        const memPct = v['mem.used_pct'] || 0;

        // Swap
        const swapTotal = v['mem.swap.total'] || 0;
        const swapUsed = v['mem.swap.used'] || 0;
        const swapPct = swapTotal > 0 ? (swapUsed / swapTotal) * 100 : 0;

        // Build process rows
        let procRows = '';
        for (let i = 0; i < this._topN; i++) {
            const pid = v[`proc.top_cpu.${i}.pid`];
            const name = this.procNames[`proc.top_cpu.${i}.name`] || '—';
            const cpu = v[`proc.top_cpu.${i}.cpu_pct`];
            const mem = v[`proc.top_cpu.${i}.mem_pct`];
            if (pid == null) continue;
            const cpuColor = (cpu || 0) > 50 ? 'top-val-crit' : (cpu || 0) > 20 ? 'top-val-warn' : 'top-val-ok';
            procRows += `<tr>
                <td class="top-proc-pid">${Math.round(pid)}</td>
                <td class="top-proc-name">${this._escapeHtml(name)}</td>
                <td class="top-proc-num ${cpuColor}">${this._fmt(cpu, 1)}%</td>
                <td class="top-proc-num">${this._fmt(mem, 1)}%</td>
            </tr>`;
        }

        this.el.innerHTML = `
            <div class="top-summary">
                <div class="top-row">
                    <span class="top-label">Tasks:</span> <span class="top-val">${tasks}</span>
                    <span class="top-sep">│</span>
                    <span class="top-label">Load:</span>
                    <span class="top-val">${load1}</span>
                    <span class="top-val">${load5}</span>
                    <span class="top-val">${load15}</span>
                </div>
                <div class="top-row">
                    <span class="top-label">CPU</span>
                    ${this._bar(cpuUsage, '#6C8EFF')}
                    <span class="top-detail">us:${this._fmt(cpuUser,1)} sy:${this._fmt(cpuSys,1)} io:${this._fmt(cpuIo,1)} id:${this._fmt(cpuIdle,1)}</span>
                </div>
                <div class="top-row">
                    <span class="top-label">Mem</span>
                    ${this._bar(memPct, '#3DD68C')}
                    <span class="top-detail">${this._fmtBytes(memUsed)}/${this._fmtBytes(memTotal)} avl:${this._fmtBytes(memAvail)} buf/c:${this._fmtBytes(memBuffers + memCached)}</span>
                </div>
                <div class="top-row">
                    <span class="top-label">Swp</span>
                    ${this._bar(swapPct, '#F5A623')}
                    <span class="top-detail">${this._fmtBytes(swapUsed)}/${this._fmtBytes(swapTotal)}</span>
                </div>
            </div>
            <table class="top-procs">
                <thead><tr>
                    <th class="top-proc-pid">PID</th>
                    <th class="top-proc-name">COMMAND</th>
                    <th class="top-proc-num">%CPU</th>
                    <th class="top-proc-num">%MEM</th>
                </tr></thead>
                <tbody>${procRows || '<tr><td colspan="4" style="text-align:center;color:var(--text-muted)">—</td></tr>'}</tbody>
            </table>
        `;
    }

    _escapeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str || '';
        return div.innerHTML;
    }

    _bindWS() {
        this._wsHandler = (samples) => {
            // Check if topN changed and re-subscribe if needed
            if (this._rebuildMetricNames()) {
                window.wsClient.subscribe(this.metricNames);
            }

            let updated = false;
            for (const s of samples) {
                if (!this.metricNames.includes(s.metric_name)) continue;
                // Process names are stored in labels field
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

    async loadHistory(hours = 1) {
        const now = Math.floor(Date.now() / 1000);
        const from = now - hours * 3600;
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
            console.error('Failed to load top history:', e);
        }
    }
}

window.TopWidget = TopWidget;
