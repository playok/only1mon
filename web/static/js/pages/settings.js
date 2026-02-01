// Settings page
document.addEventListener('alpine:init', () => {
    Alpine.data('settingsPage', () => ({
        settings: {
            retention_hours: '24',
            collect_interval: '5',
            top_process_count: '10',
        },
        chartColors: [],
        dbInfo: { path: '', size: 0, wal_size: 0 },

        async init() {
            try {
                const data = await API.getSettings();
                if (data.retention_hours) this.settings.retention_hours = data.retention_hours;
                if (data.collect_interval) this.settings.collect_interval = data.collect_interval;
                if (data.top_process_count) this.settings.top_process_count = data.top_process_count;
            } catch (e) {
                console.error('Failed to load settings:', e);
            }
            // Load chart colors from store
            const store = Alpine.store('chartColors');
            this.chartColors = [...store.list];
            // Pad to 10
            while (this.chartColors.length < 10) {
                this.chartColors.push(store.defaults[this.chartColors.length] || '#888888');
            }
            this.loadDBInfo();
        },

        async loadDBInfo() {
            try {
                this.dbInfo = await API.getDBInfo();
            } catch (e) {
                console.error('Failed to load DB info:', e);
            }
        },

        formatBytes(bytes) {
            if (!bytes || bytes === 0) return '0 B';
            const units = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.min(Math.floor(Math.log(Math.abs(bytes)) / Math.log(1024)), units.length - 1);
            const val = bytes / Math.pow(1024, i);
            return val.toFixed(i === 0 ? 0 : 2) + ' ' + units[i];
        },

        get dbTotalSize() {
            return (this.dbInfo.size || 0) + (this.dbInfo.wal_size || 0);
        },

        async purgeData() {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            if (!confirm(t('settings.db_purge_confirm'))) return;
            try {
                await API.purgeDB();
                window.dispatchEvent(new CustomEvent('toast', {
                    detail: { msg: t('settings.db_purge_success'), type: 'success' },
                }));
                await this.loadDBInfo();
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', {
                    detail: { msg: e.message, type: 'error' },
                }));
            }
        },

        resetColors() {
            const defaults = Alpine.store('chartColors').defaults;
            this.chartColors = [...defaults];
            while (this.chartColors.length < 10) {
                this.chartColors.push('#888888');
            }
        },

        async save() {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            try {
                const payload = {
                    ...this.settings,
                    chart_colors: JSON.stringify(this.chartColors),
                };
                await API.updateSettings(payload);
                // Update shared stores so existing widgets pick up new values
                Alpine.store('chartColors').list = [...this.chartColors];
                const topN = parseInt(this.settings.top_process_count, 10) || 10;
                Alpine.store('topProcessCount').value = topN;
                window.dispatchEvent(new CustomEvent('toast', {
                    detail: { msg: t('toast.settings_saved'), type: 'success' },
                }));
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', {
                    detail: { msg: t('toast.settings_save_fail'), type: 'error' },
                }));
            }
        },
    }));
});
