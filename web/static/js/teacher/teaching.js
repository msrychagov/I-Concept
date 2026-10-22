window.teacherConceptTesting = async function ({view, state, api, esc}) {
    let data, search = '', platoon = '', test = '', page = 1, pageSize = 10;
    const error = e => {
        const el = view.querySelector('#dashboard-error');
        if (el) el.innerHTML = `<p class="error">${esc(e.message || e)}</p>`
    };

    async function reload() {
        data = await api('/dashboard');
        if (view.isConnected) draw()
    }

    function entries() {
        const rows = [];
        for (const as of data.assignments) {
            for (const student of data.students) {
                const session = data.sessions.find(s => s.studentId === student.id && s.assignmentId === as.id);
                if (!session && (student.disabled || (student.vuc && !student.platoonId) || (as.platoon && as.platoon !== student.platoon))) continue;
                rows.push({as, student, session})
            }
        }
        return rows
    }

    function draw() {
        const platoons = [...new Set(data.students.map(s => s.platoon))].sort((a, b) => a.localeCompare(b, 'ru', {numeric: true}));
        view.innerHTML = `<header class="page-heading"><div><h2>Назначения и результаты</h2></div></header><div id="dashboard-error" role="alert"></div>
 <section class="surface"><h2>Назначить тестирование</h2><form id="assignment-form"><div class="form-grid"><label>Название тестирования<input name="title" required maxlength="200" placeholder="Например, Входная диагностика"></label><label>Шаблон<select name="profileId" required><option value="">Выберите шаблон</option>${state.profiles.filter(p => p.approved).map(p => `<option value="${p.id}">${esc(p.name)}</option>`).join('')}</select></label><label>Кому назначить<select id="audience"><option value="all">Всем взводам</option><option value="platoon">Одному взводу</option></select></label><label id="platoon-field" hidden>Номер взвода<input name="platoon" list="platoon-options" maxlength="80" placeholder="Например, 101"><datalist id="platoon-options">${platoons.map(p => `<option value="${esc(p)}">`).join('')}</datalist></label></div><div class="options-line"><label class="check"><input name="activity" type="checkbox"> Мои действия и опыт</label><label class="check"><input name="motivation" type="checkbox"> Мотивация</label><label class="check"><input name="methods" type="checkbox" checked> Методики уровней Я-концепции</label></div><button type="submit">Назначить тестирование</button><p class="muted" style="margin-top:1rem">При назначении фиксируется выборка: до 3 расчётных вопросов на показатель и до 2 вопросов на описательное направление. Связанные блоки сохраняются целиком. При включении «Мои действия и опыт» добавляются 38 вопросов с развёрнутыми ответами об участии и достижениях. Порядок вопросов внутри секций перемешивается для каждого студента. Назначение доступно и студентам, зарегистрировавшимся позднее.</p></form></section>
 <section class="surface"><h2>Назначенные тестирования</h2>${data.assignments.map(a => `<div class="assignment-row"><div><b>${esc(a.title)}</b><small>${a.platoon ? 'Взвод ' + esc(a.platoon) : 'Все взводы'} · ${esc(a.profileName)} · ${a.active ? 'Активно' : 'Закрыто'}</small></div><button class="secondary" data-toggle="${a.id}" data-active="${!a.active}">${a.active ? 'Закрыть доступ' : 'Возобновить'}</button></div>`).join('') || '<p class="muted">Назначений пока нет.</p>'}</section>
 <section class="surface"><h2>Уровни методик по взводу</h2><div class="toolbar"><label>Взвод<select id="ai-platoon-choice"><option value="">Все взводы</option>${platoons.filter(Boolean).map(p => `<option value="${esc(p)}">${esc(p)}</option>`).join('')}</select></label></div><div id="methods-summary" aria-live="polite"><p class="muted">Загрузка…</p></div></section>
 <section class="surface"><h2>Результаты студентов</h2><div class="dashboard-filters"><label>ФИО<input id="student-search" placeholder="Поиск по фамилии или имени" value="${esc(search)}"></label><label>Взвод<select id="platoon-filter"><option value="">Все взводы</option>${platoons.map(p => `<option value="${esc(p)}">${esc(p)}</option>`).join('')}</select></label><label>Тестирование<select id="test-filter"><option value="">Все тестирования</option>${data.assignments.map(a => `<option value="${a.id}">${esc(a.title)}</option>`).join('')}</select></label></div><div id="result-table"></div><nav id="results-pagination" class="list-pagination" aria-label="Страницы результатов студентов"></nav></section><dialog id="result-dialog" class="result-dialog"></dialog>`;
        const aiChoose = view.querySelector('#ai-platoon-choice');
        aiChoose.value = platoon || '';
        let summary, selected = null;
        const summaryBox = view.querySelector('#methods-summary');
        const drawSummary = () => {
            summaryBox.innerHTML = window.testPortrait.summary(summary, esc, selected) + window.testPortrait.students(summary, selected, esc);
            summaryBox.querySelectorAll('[data-row]').forEach(b => b.onclick = () => {
                const next = {row: b.dataset.row, level: b.dataset.level};
                selected = selected && selected.row === next.row && selected.level === next.level ? null : next;
                drawSummary()
            });
            summaryBox.querySelectorAll('[data-session]').forEach(b => b.onclick = () => result(b.dataset.session));
        };
        const aiRefresh = async () => {
            selected = null;
            summaryBox.innerHTML = '<p class="muted">Загрузка…</p>';
            try {
                summary = await api('/reports/methods' + (aiChoose.value ? '?group=' + encodeURIComponent(aiChoose.value) : ''));
                drawSummary()
            } catch (err) {
                summaryBox.innerHTML = `<p class="error">${esc(err.message)}</p>`
            }
        };
        aiChoose.onchange = aiRefresh;
        aiRefresh();
        view.querySelector('#platoon-filter').value = platoon;
        view.querySelector('#test-filter').value = test;
        view.querySelector('#student-search').oninput = e => {
            search = e.target.value;
            page = 1;
            table()
        };
        view.querySelector('#platoon-filter').onchange = e => {
            platoon = e.target.value;
            page = 1;
            table()
        };
        view.querySelector('#test-filter').onchange = e => {
            test = e.target.value;
            page = 1;
            table()
        };
        view.querySelector('#audience').onchange = e => {
            const label = view.querySelector('#platoon-field');
            label.hidden = e.target.value === 'all';
            label.querySelector('input').required = !label.hidden
        };
        view.querySelector('#assignment-form').onsubmit = async e => {
            e.preventDefault();
            const form = e.target, fields = Object.fromEntries(new FormData(form));
            fields.platoon = view.querySelector('#audience').value === 'all' ? '' : fields.platoon;
            fields.activity = !!fields.activity;
            fields.motivation = !!fields.motivation;
            fields.methods = !!fields.methods;
            form.querySelector('button').disabled = true;
            try {
                await api('/assignments', {method: 'POST', body: fields});
                await reload()
            } catch (err) {
                error(err);
                form.querySelector('button').disabled = false
            }
        };
        view.querySelectorAll('[data-toggle]').forEach(b => b.onclick = async () => {
            try {
                await api('/assignments/' + encodeURIComponent(b.dataset.toggle), {
                    method: 'PATCH',
                    body: {active: b.dataset.active === 'true'}
                });
                await reload()
            } catch (e) {
                error(e)
            }
        });
        table();
    }

    function table() {
        const rows = entries().filter(x => (!platoon || x.student.platoon === platoon) && (!test || x.as.id === test) && (!search || x.student.name.toLowerCase().includes(search.toLowerCase())));
        const pagination = window.listPagination(view.querySelector('#results-pagination'), {
            total: rows.length,
            page,
            pageSize,
            onChange: (next, size) => {
                page = next;
                pageSize = size;
                table()
            }
        });
        page = pagination.page;
        view.querySelector('#result-table').innerHTML = `<p class="muted">Найдено назначений: ${rows.length}</p><div class="table-scroll"><table><thead><tr><th>ФИО</th><th>Взвод</th><th>Тестирование</th><th>Статус</th><th>Ответы</th><th></th></tr></thead><tbody>${rows.slice(pagination.start, pagination.end).map(({
                                                                                                                                                                                                                                                                                                                              student: s,
                                                                                                                                                                                                                                                                                                                              as,
                                                                                                                                                                                                                                                                                                                              session: r
                                                                                                                                                                                                                                                                                                                          }) => `<tr><td><b>${esc(s.name)}</b><small>${esc(s.email)}</small></td><td>${esc(s.platoon)}</td><td>${esc(as.title)}</td><td>${r?.completed ? 'Пройдено' : r ? 'В процессе' : 'Не начато'}</td><td>${r ? r.answered + ' / ' + r.total : '—'}</td><td>${r ? `<button class="secondary" data-result="${r.id}">Результат</button>` : '—'}</td></tr>`).join('') || '<tr><td colspan="6">Результаты не найдены. Измените поиск или фильтры.</td></tr>'}</tbody></table></div>`;
        view.querySelectorAll('[data-result]').forEach(b => b.onclick = () => result(b.dataset.result))
    }

    function result(id) {
        return window.teacherResult({view, api, esc, id})
    }

    view.innerHTML = '<p role="status">Загрузка назначений и результатов…</p>';
    try {
        await reload()
    } catch (e) {
        view.innerHTML = `<p class="error">${esc(e.message)}</p><a href="/teacher">Войти в кабинет</a>`
    }
};
