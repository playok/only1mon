// Events page — alert rule management
document.addEventListener('alpine:init', () => {
    Alpine.data('eventsPage', () => ({
        rules: [],
        showModal: false,
        editing: false,
        form: { metric_pattern: '', operator: 'gt', threshold: 0, severity: 'warning', message_en: '', message_ko: '', enabled: true },
        editId: null,

        async init() {
            await this.loadRules();
        },

        async loadRules() {
            try {
                this.rules = await API.getAlertRules();
            } catch (e) {
                console.error('Failed to load alert rules:', e);
            }
        },

        openAdd() {
            this.editing = false;
            this.editId = null;
            this.form = { metric_pattern: '', operator: 'gt', threshold: 0, severity: 'warning', message_en: '', message_ko: '', enabled: true };
            this.showModal = true;
        },

        openEdit(rule) {
            this.editing = true;
            this.editId = rule.id;
            this.form = {
                metric_pattern: rule.metric_pattern,
                operator: rule.operator,
                threshold: rule.threshold,
                severity: rule.severity,
                message_en: rule.message_en,
                message_ko: rule.message_ko,
                enabled: rule.enabled,
            };
            this.showModal = true;
        },

        async saveRule() {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            try {
                const data = { ...this.form, threshold: parseFloat(this.form.threshold) };
                if (this.editing) {
                    await API.updateAlertRule(this.editId, data);
                } else {
                    await API.createAlertRule(data);
                }
                this.showModal = false;
                await this.loadRules();
                window.dispatchEvent(new CustomEvent('toast', { detail: { msg: t('toast.rule_saved'), type: 'success' } }));
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', { detail: { msg: t('toast.rule_save_fail'), type: 'error' } }));
            }
        },

        async deleteRule(rule) {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            if (!confirm(t('events.delete_confirm'))) return;
            try {
                await API.deleteAlertRule(rule.id);
                await this.loadRules();
                window.dispatchEvent(new CustomEvent('toast', { detail: { msg: t('toast.rule_deleted'), type: 'success' } }));
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', { detail: { msg: t('toast.rule_delete_fail'), type: 'error' } }));
            }
        },

        async toggleEnabled(rule) {
            const t = Alpine.store('i18n').t.bind(Alpine.store('i18n'));
            try {
                await API.updateAlertRule(rule.id, { ...rule, enabled: !rule.enabled });
                await this.loadRules();
            } catch (e) {
                window.dispatchEvent(new CustomEvent('toast', { detail: { msg: t('toast.rule_save_fail'), type: 'error' } }));
            }
        },

        operatorLabel(op) {
            const map = { gt: '>', gte: '>=', lt: '<', lte: '<=' };
            return map[op] || op;
        },
    }));
});
