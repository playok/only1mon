// Metrics page - collector listing with toggle, accordion, and per-metric toggles
document.addEventListener('alpine:init', () => {
    Alpine.data('metricsPage', () => ({
        collectors: [],
        expandedCollector: null,

        async init() {
            await this.fetch();
        },

        async fetch() {
            try {
                this.collectors = await API.getCollectors();
                // Sort: enabled first, then by name
                this.collectors.sort((a, b) => {
                    if (a.enabled !== b.enabled) return b.enabled - a.enabled;
                    return a.name.localeCompare(b.name);
                });
            } catch (e) {
                console.error('Failed to load collectors:', e);
            }
        },

        toggleExpand(collectorId) {
            this.expandedCollector = this.expandedCollector === collectorId ? null : collectorId;
        },

        async toggle(collector) {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            try {
                if (collector.enabled) {
                    await API.disableCollector(collector.id);
                    collector.enabled = false;
                    window.dispatchEvent(new CustomEvent('toast', {
                        detail: { msg: `${collector.name} ${t('toast.disabled')}`, type: 'info' },
                    }));
                } else {
                    await API.enableCollector(collector.id);
                    collector.enabled = true;
                    window.dispatchEvent(new CustomEvent('toast', {
                        detail: { msg: `${collector.name} ${t('toast.enabled')}`, type: 'success' },
                    }));
                }
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', {
                    detail: { msg: `${t('toast.toggle_fail')}: ${collector.name}`, type: 'error' },
                }));
                await this.fetch();
            }
        },

        async toggleMetric(collector, ms) {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            try {
                if (ms.enabled) {
                    await API.disableMetric(ms.name);
                    ms.enabled = false;
                } else {
                    await API.enableMetric(ms.name);
                    ms.enabled = true;
                }
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', {
                    detail: { msg: `${t('toast.toggle_fail')}: ${ms.name}`, type: 'error' },
                }));
                await this.fetch();
            }
        },

        async toggleAllMetrics(collector, enabled) {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            try {
                if (enabled) {
                    await API.enableCollectorMetrics(collector.id);
                } else {
                    await API.disableCollectorMetrics(collector.id);
                }
                if (collector.metric_states) {
                    collector.metric_states.forEach(ms => ms.enabled = enabled);
                }
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', {
                    detail: { msg: `${t('toast.toggle_fail')}: ${collector.name}`, type: 'error' },
                }));
                await this.fetch();
            }
        },

        allMetricsEnabled(collector) {
            if (!collector.metric_states || collector.metric_states.length === 0) return true;
            return collector.metric_states.every(ms => ms.enabled);
        },

        someMetricsDisabled(collector) {
            if (!collector.metric_states || collector.metric_states.length === 0) return false;
            return collector.metric_states.some(ms => !ms.enabled);
        },

        enabledMetricCount(collector) {
            if (!collector.metric_states) return 0;
            return collector.metric_states.filter(ms => ms.enabled).length;
        },
    }));
});
