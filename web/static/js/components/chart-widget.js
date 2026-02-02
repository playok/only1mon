// Chart Widget - wraps uPlot for real-time metric display
class ChartWidget {
    constructor(container, options = {}) {
        this.container = container;
        this.title = options.title || 'Chart';
        this.metricNames = options.metrics || [];
        this.metricMeta = options.metricMeta || {};
        this.fixedAxis = options.fixedAxis || false;
        this.live = options.live !== undefined ? options.live : true;
        this.maxPoints = options.maxPoints || 300;
        this.data = [[]]; // [timestamps, ...series]
        this.plot = null;
        this.seriesMap = {};
        this.tooltip = null;

        // Determine dominant unit for Y-axis formatting
        this._unit = this._detectUnit();

        // Initialize series data arrays
        for (let i = 0; i < this.metricNames.length; i++) {
            this.data.push([]);
            this.seriesMap[this.metricNames[i]] = i + 1;
        }

        this._initPlot();
        this._bindWS();
        this._observeResize();
    }

    _detectUnit() {
        const units = new Set();
        for (const name of this.metricNames) {
            const meta = this.metricMeta[name];
            if (meta && meta.unit) units.add(meta.unit);
        }
        // If all metrics share the same unit, use it
        if (units.size === 1) return [...units][0];
        // If mixed but all are byte-related, use bytes
        if (units.size > 0 && [...units].every(u => u === 'bytes' || u === 'bytes/s')) return [...units][0];
        return null;
    }

    _formatValue(v) {
        return this._formatWithUnit(v, this._unit);
    }

    _formatWithUnit(v, unit) {
        if (v == null) return '—';
        if (unit === 'bytes') return this._formatBytes(v, false);
        if (unit === 'bytes/s') return this._formatBytes(v, true);
        if (unit === '%') return v.toFixed(1) + '%';
        if (unit === 'us') {
            if (v >= 1000000) return (v / 1000000).toFixed(2) + 's';
            if (v >= 1000) return (v / 1000).toFixed(1) + 'ms';
            return v.toFixed(0) + 'μs';
        }
        if (unit === 'ms') {
            if (v >= 1000) return (v / 1000).toFixed(2) + 's';
            return v.toFixed(1) + 'ms';
        }
        if (unit === '°C') return v.toFixed(1) + '°C';
        if (unit === 'W') return v.toFixed(1) + 'W';
        if (unit === 'MHz') return v.toFixed(0) + ' MHz';
        if (unit) return v.toFixed(2) + ' ' + unit;
        return v.toFixed(2);
    }

    _unitOf(metricName) {
        const meta = this.metricMeta[metricName];
        return (meta && meta.unit) || this._unit || null;
    }

    _formatBytes(bytes, perSec) {
        const suffix = perSec ? '/s' : '';
        if (bytes === 0) return '0 B' + suffix;
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.min(Math.floor(Math.log(Math.abs(bytes)) / Math.log(1024)), units.length - 1);
        const val = bytes / Math.pow(1024, i);
        return val.toFixed(i === 0 ? 0 : 1) + ' ' + units[i] + suffix;
    }

    _getSize() {
        const rect = this.container.getBoundingClientRect();
        return {
            width: Math.floor(rect.width) || 400,
            height: Math.floor(rect.height) || 200,
        };
    }

    _getColors() {
        const store = window.Alpine && Alpine.store('chartColors');
        return (store && store.list) || ['#6C8EFF','#3DD68C','#F5A623','#FF6B6B','#A78BFA','#F472B6','#56CCF2','#FB923C','#34D399','#F87171'];
    }

    _initPlot() {
        const colors = this._getColors();
        const self = this;

        const series = [{}]; // timestamp series
        this.metricNames.forEach((name, i) => {
            const shortName = name.split('.').slice(-1)[0];
            const unit = this._unitOf(name);
            series.push({
                label: shortName,
                stroke: colors[i % colors.length],
                width: 2,
                value: (u, v) => self._formatWithUnit(v, unit),
            });
        });

        const size = this._getSize();

        // Create tooltip element
        this.tooltip = document.createElement('div');
        this.tooltip.className = 'chart-tooltip';
        this.container.appendChild(this.tooltip);

        const opts = {
            width: size.width,
            height: size.height,
            series: series,
            axes: [
                {
                    stroke: '#5d6380',
                    grid: { stroke: 'rgba(255,255,255,0.04)', width: 1 },
                    ticks: { stroke: 'rgba(255,255,255,0.06)', width: 1 },
                    gap: 8,
                    size: 28,
                    space: (u, axisIdx, scaleMin, scaleMax, plotDim) => {
                        // Minimum pixel space per tick — wider = fewer labels
                        return Math.max(60, Math.round(plotDim / 6));
                    },
                    values: (u, ticks) => {
                        const plotW = u.bbox.width / devicePixelRatio;
                        // Pick format based on available width per tick
                        const pxPerTick = ticks.length > 1 ? plotW / ticks.length : plotW;
                        const useShort = pxPerTick < 80;
                        return ticks.map(v => {
                            const d = new Date(v * 1000);
                            if (useShort) {
                                // HH:MM only
                                return d.getHours().toString().padStart(2, '0') + ':' +
                                       d.getMinutes().toString().padStart(2, '0');
                            }
                            // HH:MM:SS
                            return d.getHours().toString().padStart(2, '0') + ':' +
                                   d.getMinutes().toString().padStart(2, '0') + ':' +
                                   d.getSeconds().toString().padStart(2, '0');
                        });
                    },
                },
                {
                    stroke: '#5d6380',
                    grid: { stroke: 'rgba(255,255,255,0.04)', width: 1 },
                    ticks: { stroke: 'rgba(255,255,255,0.06)', width: 1 },
                    gap: 4,
                    size: self._unit && (self._unit === 'bytes' || self._unit === 'bytes/s') ? 60 : 45,
                    values: (u, ticks) => ticks.map(v => self._formatValue(v)),
                },
            ],
            padding: [8, 8, 0, 0],
            cursor: {
                show: true,
                points: { show: true, size: 5, fill: '#171b2d' },
                sync: {
                    key: 'only1mon-sync',
                    setSeries: false,
                },
            },
            legend: { show: false },
            scales: {
                x: { time: false },
                ...(self.fixedAxis ? { y: { range: [0, 100] } } : {}),
            },
            hooks: {
                setCursor: [(u) => { self._updateTooltip(u); }],
            },
        };

        this.plot = new uPlot(opts, this.data, this.container);
    }

    _updateTooltip(u) {
        const { left, top, idx } = u.cursor;

        if (idx == null || left < 0 || top < 0) {
            this.tooltip.style.display = 'none';
            return;
        }

        const ts = this.data[0][idx];
        if (ts == null) {
            this.tooltip.style.display = 'none';
            return;
        }

        const time = new Date(ts * 1000).toLocaleTimeString();
        let rows = `<div class="chart-tooltip-time">${time}</div>`;
        const colors = this._getColors();

        for (let i = 0; i < this.metricNames.length; i++) {
            const sIdx = i + 1;
            const val = this.data[sIdx][idx];
            const name = this.metricNames[i];
            const label = name.split('.').slice(-1)[0];
            const color = colors[i % colors.length];
            const display = this._formatWithUnit(val, this._unitOf(name));
            rows += `<div class="chart-tooltip-row"><span class="chart-tooltip-dot" style="background:${color}"></span>${label}: <strong>${display}</strong></div>`;
        }

        this.tooltip.innerHTML = rows;
        this.tooltip.style.display = 'block';

        // Position tooltip: follow cursor, stay inside container
        const rect = this.container.getBoundingClientRect();
        const tw = this.tooltip.offsetWidth;
        let lx = left + 12;
        if (lx + tw > rect.width) {
            lx = left - tw - 12;
        }
        let ly = top + 12;
        const th = this.tooltip.offsetHeight;
        if (ly + th > rect.height) {
            ly = top - th - 12;
        }
        if (ly < 0) ly = 4;

        this.tooltip.style.left = lx + 'px';
        this.tooltip.style.top = ly + 'px';
    }

    _bindWS() {
        this._wsHandler = (samples) => {
            if (!this.live) return;
            let updated = false;
            for (const s of samples) {
                const seriesIdx = this.seriesMap[s.metric_name];
                if (seriesIdx === undefined) continue;

                // Add timestamp if new
                const lastTs = this.data[0][this.data[0].length - 1];
                if (lastTs !== s.timestamp) {
                    this.data[0].push(s.timestamp);
                    // Fill nulls for other series
                    for (let i = 1; i < this.data.length; i++) {
                        if (this.data[i].length < this.data[0].length) {
                            this.data[i].push(null);
                        }
                    }
                }

                // Set value at correct position
                const tsIdx = this.data[0].indexOf(s.timestamp);
                if (tsIdx >= 0) {
                    this.data[seriesIdx][tsIdx] = s.value;
                }
                updated = true;
            }

            if (updated) {
                // Trim to maxPoints
                if (this.data[0].length > this.maxPoints) {
                    const excess = this.data[0].length - this.maxPoints;
                    for (let i = 0; i < this.data.length; i++) {
                        this.data[i] = this.data[i].slice(excess);
                    }
                }
                this.plot.setData(this.data);
            }
        };

        window.wsClient.on('metrics', this._wsHandler);
        window.wsClient.subscribe(this.metricNames);
    }

    _observeResize() {
        this._resizeObs = new ResizeObserver((entries) => {
            for (const entry of entries) {
                const { width, height } = entry.contentRect;
                if (width > 0 && height > 0 && this.plot) {
                    this.plot.setSize({
                        width: Math.floor(width),
                        height: Math.floor(height),
                    });
                }
            }
        });
        this._resizeObs.observe(this.container);
    }

    resize(width, height) {
        if (this.plot) {
            this.plot.setSize({
                width: Math.floor(width) || 400,
                height: Math.floor(height) || 200,
            });
        }
    }

    destroy() {
        if (this._resizeObs) {
            this._resizeObs.disconnect();
            this._resizeObs = null;
        }
        window.wsClient.off('metrics', this._wsHandler);
        window.wsClient.unsubscribe(this.metricNames);
        if (this.tooltip && this.tooltip.parentNode) {
            this.tooltip.parentNode.removeChild(this.tooltip);
        }
        if (this.plot) {
            this.plot.destroy();
            this.plot = null;
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

            // Reset data
            this.data = [[]];
            for (let i = 0; i < this.metricNames.length; i++) {
                this.data.push([]);
            }

            // Group by timestamp
            const tsSet = new Set();
            for (const s of samples) {
                tsSet.add(s.timestamp);
            }
            const timestamps = [...tsSet].sort((a, b) => a - b);
            this.data[0] = timestamps;

            // Initialize series with nulls
            for (let i = 1; i < this.data.length; i++) {
                this.data[i] = new Array(timestamps.length).fill(null);
            }

            // Fill values
            for (const s of samples) {
                const seriesIdx = this.seriesMap[s.metric_name];
                if (seriesIdx === undefined) continue;
                const tsIdx = timestamps.indexOf(s.timestamp);
                if (tsIdx >= 0) {
                    this.data[seriesIdx][tsIdx] = s.value;
                }
            }

            this.plot.setData(this.data);
        } catch (e) {
            console.error('Failed to load history:', e);
        }
    }
}

window.ChartWidget = ChartWidget;
