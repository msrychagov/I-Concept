// Teacher-only report viewer with the complete, escaped model responses.
window.aiReportWidget = (() => {
    const escape = text => String(text ?? '').replace(/[&<>"']/g, c => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[c]));
    const number = v => v == null ? 'Нет расчёта' : Number(v).toFixed(2).replace('.', ',') + ' / 5';
    const items = values => (Array.isArray(values) ? values : []).map(value => `<li>${escape(value)}</li>`).join('');
    const section = (title, body, className = '') => `<section class="ai-report-section ${className}"><h4>${title}</h4>${body}</section>`;

    function mount({element, api, kind, id}) {
        if (!element || !id) return () => {
        };
        const path = kind === 'student'
            ? '/sessions/' + encodeURIComponent(id) + '/ai-report'
            : '/reports/ai/platoon?group=' + encodeURIComponent(id);
        let active = true, timeout = null;
        const alive = () => active && element.isConnected;
        const cleanup = () => {
            active = false;
            if (timeout) clearTimeout(timeout);
        };
        const showStatus = text => {
            element.innerHTML = `<p class="muted" role="status">${escape(text)}</p>`;
        };
        const poll = () => {
            if (alive()) timeout = setTimeout(() => request(false), 2600);
        };
        const render = state => {
            if (!alive()) return;
            const status = state.status;
            if (status === 'ready' && state.report) {
                const r = state.report;
                const structured = r.formatVersion === '2';
                const details = (Array.isArray(r.indicators) ? r.indicators : []).map(x => `<details class="evidence-card ai-indicator">
                    <summary><b>${escape(x.id)} · ${escape(x.name)}</b><span>${number(x.score)}${kind === 'platoon' ? ` · ${Number(x.n || 0)} участников` : ''}</span></summary>
                    ${structured ? `<div class="ai-indicator-body">
                        <p class="ai-indicator-status"><strong>Статус:</strong> ${escape(x.status)}</p>
                        ${x.definition ? `<p><strong>Что измеряет показатель:</strong> ${escape(x.definition)}</p>` : ''}
                        <p><strong>Фактические данные:</strong> ${escape(x.evidence)}</p>
                        <p><strong>Интерпретация:</strong> ${escape(x.interpretation)}</p>
                        ${x.nextStep ? `<p><strong>Следующий шаг:</strong> ${escape(x.nextStep)}</p>` : ''}
                        ${x.discussion ? `<p class="ai-discussion"><strong>Вопрос для обсуждения:</strong> ${escape(x.discussion)}</p>` : ''}
                    </div>` : `<p>${escape(x.text)}</p><small>${Number(x.coverage)} из ${Number(x.total)} ответов</small>`}
                </details>`).join('');
                const findings = structured ? section('2. Фактические результаты', `<ul>${items(r.findings)}</ul>`) : '';
                const dataQuality = structured ? section('1. Основание и полнота данных', `<p>${escape(r.dataQuality)}</p>${kind === 'platoon' ? `<p class="muted">Всего завершённых: ${Number(r.participants || 0)} · вошло в сводку: ${Number(r.included || 0)} · исключено: ${Number(r.excluded || 0)}</p>` : ''}`) : '';
                const source = structured ? `<p class="ai-report-source">${escape(r.sourceNote || '')}</p>` : '';
                const responses = (Array.isArray(r.modelResponses) ? r.modelResponses : []).map((text, i) => `<details${r.generationError || !r.summary ? ' open' : ''}><summary>Ответ модели ${i + 1}</summary><pre style="white-space:pre-wrap;overflow-wrap:anywhere">${escape(text)}</pre></details>`).join('');
                element.innerHTML = `<div class="ai-report-header"><div><h3>Аналитический отчёт ${kind === 'platoon' ? 'по взводу' : 'по студенту'}</h3><small>Локальная модель: ${escape(r.model || 'не указана')} · ${escape(new Date(r.created).toLocaleString('ru-RU'))}</small></div></div>
                    ${r.generationError ? `<p role="status">${escape(r.generationError)}</p>` : ''}
                    ${dataQuality}${findings}
                    ${section(structured ? '3. Общий вывод' : 'Общий вывод', `<p>${escape(r.summary)}</p>`, 'ai-report-summary')}
                    ${section(structured ? '4. Анализ каждого показателя' : 'Текст по каждому показателю', details ? `<div class="ai-indicators">${details}</div>` : '<p>В этой методике нет доступных для расчёта показателей.</p>')}
                    ${section(structured ? '5. Рекомендации преподавателю' : 'Возможные педагогические действия', `<ul>${items(r.recommendations)}</ul>`)}
                    ${section(structured ? '6. Ограничения анализа' : 'Ограничения анализа', `<p>${escape(r.limitations)}</p>`, 'ai-report-limitations')}
                    ${responses ? section('Полные ответы модели', responses) : ''}
                    ${source}`;
            } else if (status === 'working') {
                element.innerHTML = '<p role="status">Локальная модель формирует отчёт. Это может занять несколько минут…</p><div class="ai-working-bar" aria-hidden="true"><i></i></div>';
                poll();
                return;
            } else {
                const message = status === 'off' ? (state.reason || 'Локальная модель не подключена.') :
                    status === 'failed' ? (state.reason || 'Ошибка формирования отчёта.') :
                        'Сформировать единообразный отчёт с подтверждёнными данными, выводом и пояснениями по каждому показателю.';
                element.innerHTML = `<p class="muted" role="status">${escape(message)}</p>` + (status !== 'off' ? '<button type="button" class="secondary" data-ai-generate>Сформировать отчёт</button>' : '');
            }
            element.querySelector('[data-ai-generate]')?.addEventListener('click', () => request(true));
        };

        async function request(start) {
            if (!alive()) return;
            if (start) showStatus('Запуск анализа…');
            try {
                const state = await api(path, start ? {method: 'POST'} : {});
                render(state);
            } catch (err) {
                if (!alive()) return;
                element.innerHTML = `<p class="error" role="alert">${escape(err.message || 'Ошибка запроса')}</p><button type="button" class="secondary" data-ai-generate>Повторить</button>`;
                element.querySelector('[data-ai-generate]').addEventListener('click', () => request(true));
            }
        }

        request(false);
        return cleanup;
    }

    return {mount};
})();
