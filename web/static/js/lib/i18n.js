// i18n - lightweight internationalization via Alpine.store
// Usage in templates: $store.i18n.t('key')
// Usage in JS: Alpine.store('i18n').t('key')

const translations = {
    en: {
        // Nav
        'nav.dashboard': 'Dashboard',
        'nav.metrics': 'Metrics',
        'nav.events': 'Events',
        'nav.settings': 'Settings',

        // Connection
        'conn.connected': 'Connected',
        'conn.disconnected': 'Disconnected',

        // Dashboard toolbar
        'dash.new_layout': '-- New Layout --',
        'dash.save': 'Save',
        'dash.add_widget': '+ Widget',

        // Widget context menu
        'widget.edit_title': 'Edit Chart Widget',
        'widget.edit': 'Edit',
        'widget.remove': 'Remove',

        // Add widget modal
        'widget.add_title': 'Add Chart Widget',
        'widget.title_label': 'Widget Title',
        'widget.title_placeholder': 'e.g. CPU Usage',
        'widget.select_metrics': 'Select Metrics',
        'widget.loading': 'Loading metrics...',
        'widget.selected': 'selected',
        'widget.metrics_selected': 'metric(s) selected',
        'widget.select_all': 'Select all',
        'widget.deselect_all': 'Deselect all',
        'widget.cancel': 'Cancel',
        'widget.add_btn': 'Add Widget',
        'widget.add_table': 'Add as Table',
        'widget.add_top': 'System Top',
        'widget.top_title': 'System Top',
        'widget.add_iotop': 'IoTop',
        'widget.iotop_title': 'Disk I/O Top',
        'widget.unit_prefix': 'Unit: ',
        'widget.fixed_axis': 'Fix Y-axis range 0\u2013100%',

        // Metrics page
        'metrics.title': 'Collectors',
        'metrics.select_all': 'Enable All',
        'metrics.deselect_all': 'Disable All',
        'metrics.metric_enabled': 'enabled',
        'metrics.metric_disabled': 'disabled',
        'metrics.collector_off_hint': 'Collector is off. Enable collector first.',
        'impact.none': 'No impact',
        'impact.low': 'low impact',
        'impact.medium': 'medium impact',
        'impact.high': 'high impact',
        'metrics.more': 'more',

        // Settings page
        'settings.title': 'Settings',
        'settings.retention': 'Data Retention (hours)',
        'settings.interval': 'Collection Interval (seconds)',
        'settings.save': 'Save Settings',
        'settings.top_process_count': 'Top Process Count (Top/IoTop)',
        'settings.chart_colors': 'Chart Series Colors',
        'settings.color_reset': 'Reset to Default',
        'settings.db_info': 'Database',
        'settings.db_path': 'Path',
        'settings.db_size': 'DB Size',
        'settings.db_wal_size': 'WAL Size',
        'settings.db_total_size': 'Total Size',
        'settings.db_purge': 'Purge All Data',
        'settings.db_purge_confirm': 'Are you sure you want to delete all metric data? This cannot be undone.',
        'settings.db_purge_success': 'All metric data has been purged',

        // Toasts
        'toast.layout_saved': 'Layout saved',
        'toast.layout_save_fail': 'Failed to save layout',
        'toast.settings_saved': 'Settings saved',
        'toast.settings_save_fail': 'Failed to save settings',
        'toast.enabled': 'enabled',
        'toast.disabled': 'disabled',
        'toast.toggle_fail': 'Failed to toggle',

        // Alerts
        'alerts.title': 'Performance Events',
        'alerts.no_alerts': 'No active alerts — system is running normally',
        'alerts.severity.critical': 'CRITICAL',
        'alerts.severity.warning': 'WARNING',
        'alerts.severity.info': 'INFO',

        // Events page (alert rules)
        'events.title': 'Performance Event Rules',
        'events.add_rule': '+ Add Rule',
        'events.no_rules': 'No event rules configured',
        'events.edit': 'Edit',
        'events.delete': 'Delete',
        'events.delete_confirm': 'Delete this rule?',
        'events.modal_add': 'Add Event Rule',
        'events.modal_edit': 'Edit Event Rule',
        'events.metric_pattern': 'Metric Pattern',
        'events.metric_pattern_hint': 'e.g. cpu.total.user, disk.*.used_pct',
        'events.operator': 'Operator',
        'events.threshold': 'Threshold',
        'events.severity': 'Severity',
        'events.message_en': 'Message (EN)',
        'events.message_ko': 'Message (KO)',
        'events.message_hint': 'Use %.1f for value placeholder',
        'events.cancel': 'Cancel',
        'events.save': 'Save',
        'events.enabled': 'Enabled',
        'toast.rule_saved': 'Rule saved',
        'toast.rule_deleted': 'Rule deleted',
        'toast.rule_save_fail': 'Failed to save rule',
        'toast.rule_delete_fail': 'Failed to delete rule',

        // Layout save prompt
        'prompt.layout_name': 'Layout name:',
    },
    ko: {
        // Nav
        'nav.dashboard': '대시보드',
        'nav.metrics': '메트릭',
        'nav.events': '이벤트',
        'nav.settings': '설정',

        // Connection
        'conn.connected': '연결됨',
        'conn.disconnected': '연결 끊김',

        // Dashboard toolbar
        'dash.new_layout': '-- 새 레이아웃 --',
        'dash.save': '저장',
        'dash.add_widget': '+ 위젯',

        // Widget context menu
        'widget.edit_title': '차트 위젯 편집',
        'widget.edit': '편집',
        'widget.remove': '삭제',

        // Add widget modal
        'widget.add_title': '차트 위젯 추가',
        'widget.title_label': '위젯 제목',
        'widget.title_placeholder': '예: CPU 사용률',
        'widget.select_metrics': '메트릭 선택',
        'widget.loading': '메트릭 로딩 중...',
        'widget.selected': '선택됨',
        'widget.metrics_selected': '개 메트릭 선택됨',
        'widget.select_all': '전체 선택',
        'widget.deselect_all': '전체 해제',
        'widget.cancel': '취소',
        'widget.add_btn': '위젯 추가',
        'widget.add_table': '표로 추가',
        'widget.add_top': '시스템 Top',
        'widget.top_title': '시스템 Top',
        'widget.add_iotop': 'IoTop',
        'widget.iotop_title': '디스크 I/O Top',
        'widget.unit_prefix': '단위: ',
        'widget.fixed_axis': 'Y축 범위 0~100% 고정',

        // Metrics page
        'metrics.title': '수집기',
        'metrics.select_all': '전체 활성화',
        'metrics.deselect_all': '전체 비활성화',
        'metrics.metric_enabled': '활성화됨',
        'metrics.metric_disabled': '비활성화됨',
        'metrics.collector_off_hint': '수집기가 꺼져있습니다. 수집기를 먼저 켜주세요.',
        'impact.none': '부하 없음',
        'impact.low': '부하 경미',
        'impact.medium': '부하 주의',
        'impact.high': '부하 높음',
        'metrics.more': '더보기',

        // Settings page
        'settings.title': '설정',
        'settings.retention': '데이터 보관 기간 (시간)',
        'settings.interval': '수집 주기 (초)',
        'settings.save': '설정 저장',
        'settings.top_process_count': 'Top 프로세스 출력 건수 (Top/IoTop)',
        'settings.chart_colors': '차트 시리즈 색상',
        'settings.color_reset': '기본값으로 초기화',
        'settings.db_info': '데이터베이스',
        'settings.db_path': '경로',
        'settings.db_size': 'DB 크기',
        'settings.db_wal_size': 'WAL 크기',
        'settings.db_total_size': '전체 크기',
        'settings.db_purge': '데이터 초기화',
        'settings.db_purge_confirm': '모든 메트릭 데이터를 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.',
        'settings.db_purge_success': '모든 메트릭 데이터가 삭제되었습니다',

        // Toasts
        'toast.layout_saved': '레이아웃이 저장되었습니다',
        'toast.layout_save_fail': '레이아웃 저장에 실패했습니다',
        'toast.settings_saved': '설정이 저장되었습니다',
        'toast.settings_save_fail': '설정 저장에 실패했습니다',
        'toast.enabled': '활성화됨',
        'toast.disabled': '비활성화됨',
        'toast.toggle_fail': '전환에 실패했습니다',

        // Alerts
        'alerts.title': '성능 이벤트',
        'alerts.no_alerts': '활성 알림 없음 — 시스템이 정상 작동 중입니다',
        'alerts.severity.critical': '위험',
        'alerts.severity.warning': '경고',
        'alerts.severity.info': '정보',

        // Events page (alert rules)
        'events.title': '성능 이벤트 규칙',
        'events.add_rule': '+ 규칙 추가',
        'events.no_rules': '설정된 이벤트 규칙이 없습니다',
        'events.edit': '수정',
        'events.delete': '삭제',
        'events.delete_confirm': '이 규칙을 삭제하시겠습니까?',
        'events.modal_add': '이벤트 규칙 추가',
        'events.modal_edit': '이벤트 규칙 수정',
        'events.metric_pattern': '메트릭 패턴',
        'events.metric_pattern_hint': '예: cpu.total.user, disk.*.used_pct',
        'events.operator': '연산자',
        'events.threshold': '임계값',
        'events.severity': '심각도',
        'events.message_en': '메시지 (EN)',
        'events.message_ko': '메시지 (KO)',
        'events.message_hint': '값 자리에 %.1f 사용',
        'events.cancel': '취소',
        'events.save': '저장',
        'events.enabled': '활성화',
        'toast.rule_saved': '규칙이 저장되었습니다',
        'toast.rule_deleted': '규칙이 삭제되었습니다',
        'toast.rule_save_fail': '규칙 저장에 실패했습니다',
        'toast.rule_delete_fail': '규칙 삭제에 실패했습니다',

        // Layout save prompt
        'prompt.layout_name': '레이아웃 이름:',
    },
};

document.addEventListener('alpine:init', () => {
    const saved = localStorage.getItem('only1mon_lang') || 'en';

    Alpine.store('i18n', {
        lang: saved,

        t(key) {
            const dict = translations[this.lang] || translations.en;
            return dict[key] || translations.en[key] || key;
        },

        setLang(lang) {
            this.lang = lang;
            localStorage.setItem('only1mon_lang', lang);
        },
    });
});
