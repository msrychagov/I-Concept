window.teacherResult = async function ({view, api, esc, id}) {
    const dialog = view.querySelector('#result-dialog');
    const fmt = value => value == null ? 'Нет данных' : Number(value).toFixed(2).replace('.', ',');
    let data, page = 1, pageSize = 10, stopAI = () => {};
    dialog.addEventListener('close', () => stopAI(), {once: true});
    dialog.innerHTML = '<h2>Загрузка результата…</h2>';
    dialog.showModal();

    async function load() {
        data = await api('/sessions/' + encodeURIComponent(id) + '/report');
    }

    function draw() {
        const s = data.session, r = data.report;
        const rows = r.evidence || [];
        const answered = rows.reduce((sum, row) => sum + row.coverage, 0);
        const total = rows.reduce((sum, row) => sum + row.total, 0);
        const statements = data.items.filter(item => !item.responseOnly && (item.scored || ['B', 'C', 'D'].includes(item.module)));
        dialog.innerHTML = `<div class="section-title"><div><h2>${esc(s.participant)}</h2><p>Взвод ${esc(s.group)} · ${esc(s.wave)}</p></div><button class="secondary" id="close-result">Закрыть</button></div>
   <p>${s.completed ? 'Прохождение завершено' : 'Прохождение в процессе'} · ответов на утверждения ${answered} из ${total}</p>
   <section class="surface test-portrait" id="test-portrait-box" aria-label="Портрет по результатам тестирования"></section>
   <h3>Отчёт по утверждениям</h3><p class="muted">Средний балл по показателю рассчитывается по шкале 1–5 с учётом обратных утверждений. При неполном наборе ответов балл не рассчитывается; пропуски не заменяются нулём.</p>
   <section><div id="evidence-list"></div><nav class="list-pagination" id="evidence-pagination" aria-label="Страницы диагностических показателей"></nav></section>
   <details><summary>Ответы на утверждения и комментарии</summary>${statements.map(item => `<article class="bank-item"><b>${esc(item.text)}</b><p>${s.answers[item.id] === undefined ? 'Нет ответа' : esc(item.options?.[s.answers[item.id]]?.text ?? s.answers[item.id])}</p><small>${esc(s.notes[item.id] || '')}</small></article>`).join('')}</details>
   ${(r.psychological||[]).length?`<section class="surface"><h3>Мое самочувстсвие</h3><p class="muted">Ответы представлены по направлениям без диагностических порогов и без включения в педагогические баллы. Отказ, неприменимость и отсутствие ответа учитываются отдельно.</p>${r.psychological.map(group=>`<details class="evidence-card"><summary><b>${esc(group.name)}</b><span>${group.answered} ответов · ${group.total} вопросов</span></summary><p class="muted">${esc(group.limitations)}</p><p>Не относится: ${group.notApplicable} · Отказ: ${group.declined} · Без ответа: ${group.unanswered}</p>${group.answers.map(answer=>`<article class="bank-item"><small>${esc(answer.id)} · ${esc(answer.timeWindow)}</small><b>${esc(answer.text)}</b><p>${esc(answer.selected)}</p>${answer.note?`<small>${esc(answer.note)}</small>`:''}</article>`).join('')}</details>`).join('')}</section>`:''}
   ${(r.activity || []).length ? `<section class="surface"><h3>Мои действия и опыт</h3><p class="muted">Самоописание участия и достижений. Текстовые ответы не включаются в балльный расчёт.</p>${[...new Set(r.activity.map(x => x.topic || 'Опыт деятельности'))].map(topic => `<details class="evidence-card"><summary><b>${esc(topic)}</b></summary>${r.activity.filter(x => (x.topic || 'Опыт деятельности') === topic).map(x => `<article class="bank-item"><b>${esc(x.text)}</b><p class="activity-answer-text">${esc(x.value == null ? 'Нет ответа' : x.type === 'textarea' ? x.value : ['Ни разу','Один раз','Два–три раза','Четыре–пять раз','Шесть раз и более'][x.value] ?? x.value)}</p>${x.type !== 'textarea' && x.note ? `<small>${esc(x.note)}</small>` : ''}</article>`).join('')}</details>`).join('')}</section>` : ''}
   <a class="button secondary" href="/api/v1/sessions/${encodeURIComponent(id)}/export">Экспорт сессии</a>`;
        dialog.querySelector('#close-result').onclick = () => dialog.close();
        stopAI = window.testPortrait.mount({element: dialog.querySelector('#test-portrait-box'), api, esc, id, methods: r.methods, completed: s.completed});
        list();

    }

    function list() {
        const rows = data.report.evidence || [];
        const range = window.listPagination(dialog.querySelector('#evidence-pagination'), {
            total: rows.length,
            page,
            pageSize,
            onChange: (next, size) => {
                page = next;
                pageSize = size;
                list()
            }
        });
        page = range.page;
        dialog.querySelector('#evidence-list').innerHTML = rows.slice(range.start, range.end).map(row => `<details class="evidence-card"><summary><b>${esc(row.id)} · ${esc(row.name)}</b><span>${row.statements == null ? 'Недостаточно ответов' : fmt(row.statements) + ' / 5'} · ${row.coverage}/${row.total} пунктов</span></summary>
   <p>${esc(row.definition)}</p><p class="muted">${esc(row.limitations)}</p>
   <p><b>Средний балл:</b> ${row.statements == null ? 'Недостаточно ответов' : fmt(row.statements) + ' из 5'}. <b>Ответов:</b> ${row.coverage} из ${row.total}.</p>
   </details>`).join('');
    }

    try {
        await load();
        draw()
    } catch (error) {
        dialog.innerHTML = `<p class="error">${esc(error.message)}</p><button id="close-result">Закрыть</button>`;
        dialog.querySelector('#close-result').onclick = () => dialog.close()
    }
};
