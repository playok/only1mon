// Dashboard page with gridstack.js integration
// Uses Alpine.store('dashboard') for state shared with the header toolbar.
document.addEventListener('alpine:init', () => {
    // Shared store so the header toolbar can read layouts/selectedLayout
    Alpine.store('dashboard', {
        layouts: [],
        selectedLayout: '',
        showAddWidget: false,
        showEditWidget: false,
        editWidgetId: null,
    });

    Alpine.data('dashboardPage', () => ({
        grid: null,
        widgets: {},
        newWidget: { title: '', metrics: '', fixedAxis: false },
        metricGroups: [],
        selectedMetrics: [],
        editTitle: '',
        editFixedAxis: false,

        // Context menu state
        ctxMenu: { show: false, x: 0, y: 0, widgetId: null },

        init() {
            // Expose component instance globally for header toolbar actions
            window.dashboard = this;

            // Load available metrics when add modal opens
            this.$watch('$store.dashboard.showAddWidget', async (open) => {
                if (open) {
                    await this.loadAvailableMetrics();
                } else {
                    this.selectedMetrics = [];
                    this.newWidget = { title: '', metrics: '' };
                }
            });

            // Load available metrics when edit modal opens
            this.$watch('$store.dashboard.showEditWidget', async (open) => {
                if (open) {
                    await this.loadAvailableMetrics();
                } else {
                    this.selectedMetrics = [];
                    this.editTitle = '';
                }
            });

            // Watch time range changes
            this.$watch('$store.timeRange._v', () => this._reloadAllWidgets());
            this.$watch('$store.timeRange.live', () => this._reloadAllWidgets());

            // Close context menu on any click
            document.addEventListener('click', () => {
                this.ctxMenu.show = false;
            });

            this.$nextTick(() => {
                this.grid = GridStack.init({
                    column: 12,
                    cellHeight: 80,
                    margin: 4,
                    animate: true,
                    float: false,
                    removable: false,
                }, '#dashboard-grid');

                this.grid.on('resizestop', (event, el) => {
                    const id = el.getAttribute('gs-id');
                    if (this.widgets[id]) {
                        const body = el.querySelector('.widget-body');
                        this.widgets[id].resize(body.clientWidth, body.clientHeight);
                    }
                });

                this.fetchLayouts();
            });
        },

        _reloadAllWidgets() {
            const tr = Alpine.store('timeRange');
            const from = tr.getFrom();
            const to = tr.getTo();
            for (const w of Object.values(this.widgets)) {
                if (w.reloadWithRange) {
                    w.reloadWithRange(from, to, tr.live);
                }
            }
        },

        _currentFrom() {
            return Alpine.store('timeRange').getFrom();
        },

        _currentTo() {
            return Alpine.store('timeRange').getTo();
        },

        _currentLive() {
            return Alpine.store('timeRange').live;
        },

        get store() {
            return Alpine.store('dashboard');
        },

        async fetchLayouts() {
            try {
                this.store.layouts = await API.getLayouts();
                // Auto-load the first layout if nothing is selected yet and grid is empty
                if (!this.store.selectedLayout && this.store.layouts.length > 0 && Object.keys(this.widgets).length === 0) {
                    this.store.selectedLayout = this.store.layouts[0].id;
                    await this.loadLayout();
                }
            } catch (e) {
                console.error('Failed to load layouts:', e);
            }
        },

        async loadAvailableMetrics() {
            try {
                const groups = await API.getAvailableMetrics();
                // Add _open state for tree toggling
                for (const g of groups) {
                    g._open = false;
                    if (g.children) {
                        for (const sub of g.children) {
                            sub._open = false;
                        }
                    }
                }
                this.metricGroups = groups;
            } catch (e) {
                console.error('Failed to load available metrics:', e);
            }
        },

        toggleMetric(name) {
            const idx = this.selectedMetrics.indexOf(name);
            if (idx >= 0) {
                this.selectedMetrics.splice(idx, 1);
            } else {
                this.selectedMetrics.push(name);
            }
        },

        toggleSubGroup(sub) {
            const all = this.isSubGroupAllSelected(sub);
            for (const m of (sub.metrics || [])) {
                const idx = this.selectedMetrics.indexOf(m.name);
                if (all && idx >= 0) {
                    this.selectedMetrics.splice(idx, 1);
                } else if (!all && idx < 0) {
                    this.selectedMetrics.push(m.name);
                }
            }
        },

        isSubGroupAllSelected(sub) {
            return (sub.metrics || []).length > 0 &&
                   (sub.metrics || []).every(m => this.selectedMetrics.includes(m.name));
        },

        countGroupSelected(group) {
            let count = 0;
            for (const m of (group.metrics || [])) {
                if (this.selectedMetrics.includes(m.name)) count++;
            }
            for (const sub of (group.children || [])) {
                for (const m of (sub.metrics || [])) {
                    if (this.selectedMetrics.includes(m.name)) count++;
                }
            }
            return count;
        },

        // --- Context Menu ---
        showContextMenu(event, widgetId) {
            event.preventDefault();
            event.stopPropagation();
            this.ctxMenu = { show: true, x: event.clientX, y: event.clientY, widgetId };
        },

        ctxEdit() {
            const id = this.ctxMenu.widgetId;
            this.ctxMenu.show = false;
            if (!id || !this.widgets[id]) return;

            const chart = this.widgets[id];
            this.store.editWidgetId = id;
            this.editTitle = chart.title;
            this.editFixedAxis = chart.fixedAxis || false;
            this.selectedMetrics = [...chart.metricNames];
            this.store.showEditWidget = true;
        },

        ctxRemove() {
            const id = this.ctxMenu.widgetId;
            this.ctxMenu.show = false;
            if (id) this.removeWidget(id);
        },

        applyEditWidget() {
            const id = this.store.editWidgetId;
            if (!id) return;
            const metrics = [...this.selectedMetrics];
            if (metrics.length === 0) return;
            const title = this.editTitle || metrics.slice(0, 2).join(', ') + (metrics.length > 2 ? '...' : '');

            this._ensureMetricsEnabled(metrics);

            // Destroy old chart
            if (this.widgets[id]) {
                this.widgets[id].destroy();
                delete this.widgets[id];
            }

            // Update widget header title
            const el = this.grid.getGridItems().find(item => item.getAttribute('gs-id') === id);
            if (el) {
                const headerSpan = el.querySelector('.widget-header > span');
                if (headerSpan) headerSpan.textContent = title;
            }

            // Recreate chart in the same container
            const editMetricMeta = this._buildMetricMeta(metrics);
            this.$nextTick(() => {
                const container = document.getElementById('chart-' + id);
                if (container) {
                    // Clear old chart DOM
                    container.innerHTML = '';
                    const chart = new ChartWidget(container, { title, metrics, metricMeta: editMetricMeta, fixedAxis: this.editFixedAxis, live: this._currentLive() });
                    chart.loadHistory(this._currentFrom(), this._currentTo());
                    this.widgets[id] = chart;
                }
            });

            this.store.showEditWidget = false;
        },

        // --- Ensure metrics are enabled in the collector ---
        _ensureMetricsEnabled(metrics) {
            API.ensureMetricsEnabled(metrics).catch(e => {
                console.error('Failed to ensure metrics enabled:', e);
            });
        },

        // --- Add / Remove ---
        _createWidgetHTML(id, title) {
            return `
                <div class="widget-header" oncontextmenu="window.dashboardContextMenu(event, '${id}')">
                    <span>${title}</span>
                    <button class="btn btn-sm btn-danger" onclick="window.dashboardRemoveWidget('${id}')">&times;</button>
                </div>
                <div class="widget-body" id="chart-${id}"></div>
            `;
        },

        _buildMetricMeta(metricNames) {
            const meta = {};
            for (const group of this.metricGroups) {
                for (const m of (group.metrics || [])) {
                    if (metricNames.includes(m.name)) {
                        meta[m.name] = { description: m.description, description_ko: m.description_ko, unit: m.unit };
                    }
                }
                for (const sub of (group.children || [])) {
                    for (const m of (sub.metrics || [])) {
                        if (metricNames.includes(m.name)) {
                            meta[m.name] = { description: m.description, description_ko: m.description_ko, unit: m.unit };
                        }
                    }
                }
            }
            return meta;
        },

        addWidget() {
            const metrics = this.selectedMetrics.length > 0
                ? [...this.selectedMetrics]
                : this.newWidget.metrics.split(',').map(m => m.trim()).filter(Boolean);
            if (metrics.length === 0) return;
            const title = this.newWidget.title || metrics.slice(0, 2).join(', ') + (metrics.length > 2 ? '...' : '');

            this._ensureMetricsEnabled(metrics);

            const id = 'w-' + Date.now();

            this.grid.addWidget({
                id: id,
                w: 6,
                h: 3,
                content: this._createWidgetHTML(id, title),
            });

            const metricMeta = this._buildMetricMeta(metrics);

            this.$nextTick(() => {
                const container = document.getElementById('chart-' + id);
                if (container) {
                    const chart = new ChartWidget(container, {
                        title: title,
                        metrics: metrics,
                        metricMeta: metricMeta,
                        fixedAxis: this.newWidget.fixedAxis,
                        live: this._currentLive(),
                    });
                    chart.loadHistory(this._currentFrom(), this._currentTo());
                    this.widgets[id] = chart;
                }
            });

            this.store.showAddWidget = false;
            this.selectedMetrics = [];
            this.newWidget = { title: '', metrics: '', fixedAxis: false };
        },

        addTableWidget(sub) {
            const metrics = (sub.metrics || []).map(m => m.name);
            if (metrics.length === 0) return;
            const title = sub.label;

            this._ensureMetricsEnabled(metrics);

            // Build metric metadata from the sub object
            const metricMeta = {};
            for (const m of (sub.metrics || [])) {
                metricMeta[m.name] = { description: m.description, description_ko: m.description_ko, unit: m.unit };
            }

            const id = 'w-' + Date.now();
            const rowCount = metrics.length;
            // auto size: header row + data rows, ~1 row ≈ 0.4 cellHeight units, min 3
            const h = Math.max(3, Math.min(8, Math.ceil((rowCount + 1) * 0.45)));

            this.grid.addWidget({
                id: id,
                w: 6,
                h: h,
                content: this._createWidgetHTML(id, title),
            });

            this.$nextTick(() => {
                const container = document.getElementById('chart-' + id);
                if (container) {
                    const widget = new TableWidget(container, {
                        title: title,
                        metrics: metrics,
                        metricMeta: metricMeta,
                        live: this._currentLive(),
                    });
                    widget.loadHistory(this._currentFrom(), this._currentTo());
                    this.widgets[id] = widget;
                }
            });

            this.store.showAddWidget = false;
            this.selectedMetrics = [];
            this.newWidget = { title: '', metrics: '' };
        },

        addTopWidget() {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            const title = t('widget.top_title');
            const id = 'w-' + Date.now();

            // The TopWidget internally knows which metrics it needs
            const topN = Alpine.store('topProcessCount').value || 10;
            const topMetrics = [
                'cpu.total.usage', 'cpu.total.user', 'cpu.total.system', 'cpu.total.iowait', 'cpu.total.idle',
                'cpu.load.1', 'cpu.load.5', 'cpu.load.15',
                'mem.total', 'mem.used', 'mem.available', 'mem.cached', 'mem.buffers',
                'mem.swap.total', 'mem.swap.used', 'mem.used_pct', 'proc.total_count',
            ];
            for (let i = 0; i < topN; i++) {
                topMetrics.push(`proc.top_cpu.${i}.pid`, `proc.top_cpu.${i}.name`,
                    `proc.top_cpu.${i}.cpu_pct`, `proc.top_cpu.${i}.mem_pct`);
            }
            this._ensureMetricsEnabled(topMetrics);

            this.grid.addWidget({
                id: id,
                w: 6,
                h: 4,
                content: this._createWidgetHTML(id, title),
            });

            this.$nextTick(() => {
                const container = document.getElementById('chart-' + id);
                if (container) {
                    const widget = new TopWidget(container, { title, live: this._currentLive() });
                    widget.loadHistory(this._currentFrom(), this._currentTo());
                    this.widgets[id] = widget;
                }
            });

            this.store.showAddWidget = false;
        },

        addIoTopWidget() {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            const title = t('widget.iotop_title');
            const id = 'w-' + Date.now();

            const ioTopN = Alpine.store('topProcessCount').value || 10;
            const ioMetrics = ['proc.io.total_read_bps', 'proc.io.total_write_bps'];
            for (let i = 0; i < ioTopN; i++) {
                ioMetrics.push(`proc.top_io.${i}.pid`, `proc.top_io.${i}.name`,
                    `proc.top_io.${i}.read_bps`, `proc.top_io.${i}.write_bps`);
            }
            this._ensureMetricsEnabled(ioMetrics);

            this.grid.addWidget({
                id: id,
                w: 6,
                h: 4,
                content: this._createWidgetHTML(id, title),
            });

            this.$nextTick(() => {
                const container = document.getElementById('chart-' + id);
                if (container) {
                    const widget = new IoTopWidget(container, { title, live: this._currentLive() });
                    widget.loadHistory(this._currentFrom(), this._currentTo());
                    this.widgets[id] = widget;
                }
            });

            this.store.showAddWidget = false;
        },

        removeWidget(id) {
            if (this.widgets[id]) {
                this.widgets[id].destroy();
                delete this.widgets[id];
            }
            const el = this.grid.getGridItems().find(item => item.getAttribute('gs-id') === id);
            if (el) this.grid.removeWidget(el);
        },

        async saveLayout() {
            const items = this.grid.save(true, true);
            const widgetMeta = {};
            for (const [id, w] of Object.entries(this.widgets)) {
                widgetMeta[id] = {
                    title: w.title,
                    metrics: w.metricNames,
                    type: w.type || 'chart',
                    fixedAxis: w.fixedAxis || false,
                };
                if (w.metricMeta && Object.keys(w.metricMeta).length > 0) {
                    widgetMeta[id].metricMeta = w.metricMeta;
                }
            }

            const layoutData = JSON.stringify({ grid: items, widgets: widgetMeta });
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            const name = prompt(t('prompt.layout_name'), 'default') || 'default';

            try {
                if (this.store.selectedLayout) {
                    await API.updateLayout(this.store.selectedLayout, { name, layout: layoutData });
                } else {
                    const res = await API.createLayout({ name, layout: layoutData });
                    this.store.selectedLayout = res.id;
                }
                await this.fetchLayouts();
                window.dispatchEvent(new CustomEvent('toast', { detail: { msg: t('toast.layout_saved'), type: 'success' } }));
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', { detail: { msg: t('toast.layout_save_fail'), type: 'error' } }));
            }
        },

        async loadLayout() {
            if (!this.store.selectedLayout) {
                this.grid.removeAll();
                Object.values(this.widgets).forEach(w => w.destroy());
                this.widgets = {};
                return;
            }

            try {
                const layout = await API.getLayout(this.store.selectedLayout);
                const data = JSON.parse(layout.layout);

                this.grid.removeAll();
                Object.values(this.widgets).forEach(w => w.destroy());
                this.widgets = {};

                if (data.grid && data.widgets) {
                    // Load available metrics for unit metadata
                    if (this.metricGroups.length === 0) {
                        await this.loadAvailableMetrics();
                    }

                    // Ensure all metrics used by widgets are enabled
                    const allMetrics = [];
                    for (const meta of Object.values(data.widgets)) {
                        if (meta.metrics) allMetrics.push(...meta.metrics);
                    }
                    if (allMetrics.length > 0) {
                        this._ensureMetricsEnabled(allMetrics);
                    }

                    // grid.save() returns {children:[...]} object in gridstack 10.x,
                    // bootstrap creates a plain array — handle both.
                    const gridItems = Array.isArray(data.grid) ? data.grid : (data.grid.children || []);
                    for (const item of gridItems) {
                        const id = item.id;
                        const meta = data.widgets[id];
                        if (!meta) continue;

                        // Build metricMeta from available metrics if not saved in layout
                        const metricMeta = (meta.metricMeta && Object.keys(meta.metricMeta).length > 0)
                            ? meta.metricMeta
                            : this._buildMetricMeta(meta.metrics || []);

                        this.grid.addWidget({
                            id: id,
                            x: item.x,
                            y: item.y,
                            w: item.w,
                            h: item.h,
                            content: this._createWidgetHTML(id, meta.title),
                        });

                        this.$nextTick(() => {
                            const container = document.getElementById('chart-' + id);
                            if (container) {
                                let widget;
                                const live = this._currentLive();
                                if (meta.type === 'top') {
                                    widget = new TopWidget(container, {
                                        title: meta.title,
                                        live: live,
                                    });
                                } else if (meta.type === 'iotop') {
                                    widget = new IoTopWidget(container, {
                                        title: meta.title,
                                        live: live,
                                    });
                                } else if (meta.type === 'table') {
                                    widget = new TableWidget(container, {
                                        title: meta.title,
                                        metrics: meta.metrics,
                                        metricMeta: metricMeta,
                                        live: live,
                                    });
                                } else {
                                    widget = new ChartWidget(container, {
                                        title: meta.title,
                                        metrics: meta.metrics,
                                        metricMeta: metricMeta,
                                        fixedAxis: meta.fixedAxis || false,
                                        live: live,
                                    });
                                }
                                widget.loadHistory(this._currentFrom(), this._currentTo());
                                this.widgets[id] = widget;
                            }
                        });
                    }
                }
            } catch (e) {
                console.error('Failed to load layout:', e);
            }
        },
    }));

    // Global callbacks for cross-scope access
    window.dashboardRemoveWidget = (id) => {
        if (window.dashboard) window.dashboard.removeWidget(id);
    };
    window.dashboardSaveLayout = () => {
        if (window.dashboard) window.dashboard.saveLayout();
    };
    window.dashboardLoadLayout = () => {
        if (window.dashboard) window.dashboard.loadLayout();
    };
    window.dashboardShowAddWidget = () => {
        Alpine.store('dashboard').showAddWidget = true;
    };
    window.dashboardContextMenu = (event, id) => {
        if (window.dashboard) window.dashboard.showContextMenu(event, id);
    };
    window.dashboardAddTableWidget = (subIndex, groupIndex) => {
        if (window.dashboard) {
            const group = window.dashboard.metricGroups[groupIndex];
            if (group && group.children && group.children[subIndex]) {
                window.dashboard.addTableWidget(group.children[subIndex]);
            }
        }
    };
});
