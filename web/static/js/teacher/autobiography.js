// Teacher section "Автобиография": assign the questionnaire, follow progress, read answers
// and the potential by platoon.
window.teacherAutobiography = async function ({view, api, esc}) {
    const FORM = 'autobiography';
    let data, search = '', platoon = '', page = 1, pageSize = 10, form = null;
    let potential, potentialPlatoon = '', selected = null;
    const fmtDate = v => v ? new Date(v).toLocaleDateString('ru-RU') : '';
    const error = e => {
        const el = view.querySelector('#autobiography-error');
        if (el) el.innerHTML = `<p class="error">${esc(e.message || e)}</p>`;
    };

    const loadPotential = () => api(`/forms/${FORM}/potential` + (potentialPlatoon ? '?platoon=' + encodeURIComponent(potentialPlatoon) : ''));

    async function reload() {
        [data, potential] = await Promise.all([api(`/forms/${FORM}/overview`), loadPotential()]);
        if (view.isConnected) draw();
    }

    // Every student the assignment applies to, or who already started it.
    function entries() {
        const rows = [];
        for (const as of data.assignments) {
            for (const student of data.students) {
                const submission = data.submissions.find(s => s.studentId === student.id && s.assignmentId === as.id);
                if (!submission && (student.disabled || (student.vuc && !student.platoonId) || (as.platoon && as.platoon !== student.platoon))) continue;
                rows.push({as, student, submission});
            }
        }
        return rows;
    }

    const status = s => !s ? 'Не начато' : s.completed ? `Сдано ${fmtDate(s.completedAt || s.updated)}` : `Заполнено ${s.answered} из ${s.total}`;

    function draw() {
        const platoons = [...new Set(data.students.map(s => s.platoon))].sort((a, b) => a.localeCompare(b, 'ru', {numeric: true}));
        const all = entries(), done = all.filter(x => x.submission?.completed).length;
        view.innerHTML = `<header class="page-heading"><div><div class="eyebrow">Кабинет преподавателя</div><h1>Автобиография</h1></div><div class="toolbar"><button id="preview" class="secondary">Посмотреть опросник</button><button id="refresh" class="secondary">Обновить</button></div></header><div id="autobiography-error" role="alert"></div>
  <div class="dashboard-metrics"><article><strong>${data.assignments.filter(a => a.active).length}</strong>Активных назначений</article><article><strong>${all.length}</strong>Анкет назначено</article><article><strong>${all.filter(x => x.submission && !x.submission.completed).length}</strong>В процессе</article><article><strong>${done} / ${all.length}</strong>Сдано</article></div>
  <section class="surface"><h2>Назначить </h2><form id="form-assignment"><div class="form-grid"><label>Название<input name="title" required maxlength="200" value="Автобиография"></label><label>Кому назначить<select id="form-audience"><option value="all">Всем взводам</option><option value="platoon">Одному взводу</option></select></label><div id="form-platoon-slot"></div></div><button type="submit">Назначить тестирование</button><p class="muted" style="margin-top:1rem">Назначение появится в кабинетах студентов выбранного взвода, включая зарегистрировавшихся позднее. Опросник один и не редактируется.</p></form></section>
  <section class="surface"><h2>Назначения</h2>${data.assignments.map(a => `<div class="assignment-row"><div><b>${esc(a.title)}</b><small>${a.platoon ? 'Взвод ' + esc(a.platoon) : 'Все взводы'} · ${a.active ? 'Активно' : 'Закрыто'}</small></div><button class="secondary" data-toggle="${esc(a.id)}" data-active="${!a.active}">${a.active ? 'Закрыть доступ' : 'Возобновить'}</button></div>`).join('') || '<p class="muted">Назначений пока нет.</p>'}</section>
  <section class="surface"><h2>Заполнение автобиографий</h2><div class="dashboard-filters"><label>ФИО<input id="form-search" placeholder="Поиск по фамилии или имени" value="${esc(search)}"></label><label>Взвод<select id="form-platoon-filter"><option value="">Все взводы</option>${platoons.map(p => `<option value="${esc(p)}">${esc(p)}</option>`).join('')}</select></label></div><div id="form-table"></div><nav id="form-pagination" class="list-pagination" aria-label="Страницы списка автобиографий"></nav></section>
  <section class="surface"><h2>Потенциал по взводу</h2><div class="toolbar"><label>Взвод<select id="potential-platoon"><option value="">Все взводы</option>${[...potential.platoons].sort((a, b) => a.localeCompare(b, 'ru', {numeric: true})).map(p => `<option value="${esc(p)}">${esc(p)}</option>`).join('')}</select></label></div><div id="potential-body"></div></section><dialog id="form-dialog" class="result-dialog"></dialog>`;
        view.querySelector('#form-platoon-filter').value = platoon;
        view.querySelector('#form-search').oninput = e => {
            search = e.target.value;
            page = 1;
            table();
        };
        view.querySelector('#form-platoon-filter').onchange = e => {
            platoon = e.target.value;
            page = 1;
            table();
        };
        // The platoon field is added to the page only when needed: a hidden attribute loses to the label styles.
        view.querySelector('#form-audience').onchange = e => {
            view.querySelector('#form-platoon-slot').innerHTML = e.target.value === 'all' ? '' : `<label>Номер взвода<input name="platoon" list="form-platoon-options" maxlength="80" required placeholder="Например, 101"><datalist id="form-platoon-options">${platoons.map(p => `<option value="${esc(p)}">`).join('')}</datalist></label>`;
        };
        view.querySelector('#form-assignment').onsubmit = async e => {
            e.preventDefault();
            const form = e.target, fields = Object.fromEntries(new FormData(form));
            const body = {
                title: fields.title,
                formId: FORM,
                platoon: view.querySelector('#form-audience').value === 'all' ? '' : fields.platoon || ''
            };
            form.querySelector('button').disabled = true;
            try {
                await api('/assignments', {method: 'POST', body});
                await reload();
            } catch (err) {
                error(err);
                form.querySelector('button').disabled = false;
            }
        };
        view.querySelectorAll('[data-toggle]').forEach(b => b.onclick = async () => {
            try {
                await api('/assignments/' + encodeURIComponent(b.dataset.toggle), {
                    method: 'PATCH',
                    body: {active: b.dataset.active === 'true'}
                });
                await reload();
            } catch (err) {
                error(err);
            }
        });
        view.querySelector('#potential-platoon').value = potentialPlatoon;
        view.querySelector('#potential-platoon').onchange = async e => {
            potentialPlatoon = e.target.value;
            selected = null;
            try {
                potential = await loadPotential();
                drawPotential();
            } catch (err) {
                error(err);
            }
        };
        view.querySelector('#refresh').onclick = () => reload().catch(error);
        view.querySelector('#preview').onclick = preview;
        table();
        drawPotential();
    }

    // Counts by area and level; a count shows the students, a name opens their answers.
    function drawPotential() {
        const body = view.querySelector('#potential-body');
        body.innerHTML = window.potentialView.summary(potential, esc, selected) + window.potentialView.students(potential, selected, esc);
        body.querySelectorAll('[data-level]').forEach(b => b.onclick = () => {
            const same = selected && selected.area === b.dataset.area && selected.level === b.dataset.level;
            selected = same ? null : {area: b.dataset.area, level: b.dataset.level};
            drawPotential();
        });
        body.querySelectorAll('[data-answers]').forEach(b => b.onclick = () => answers(b.dataset.answers));
    }

    function table() {
        const rows = entries().filter(x => (!platoon || x.student.platoon === platoon) && (!search || x.student.name.toLowerCase().includes(search.toLowerCase())));
        const range = window.listPagination(view.querySelector('#form-pagination'), {
            total: rows.length,
            page,
            pageSize,
            onChange: (next, size) => {
                page = next;
                pageSize = size;
                table();
            }
        });
        page = range.page;
        view.querySelector('#form-table').innerHTML = `<p class="muted">Найдено анкет: ${rows.length}</p><div class="table-scroll"><table><thead><tr><th>ФИО</th><th>Взвод</th><th>Назначение</th><th>Статус</th><th></th></tr></thead><tbody>${rows.slice(range.start, range.end).map(({
                                                                                                                                                                                                                                                                                            student: s,
                                                                                                                                                                                                                                                                                            as,
                                                                                                                                                                                                                                                                                            submission
                                                                                                                                                                                                                                                                                        }) => `<tr><td><b>${esc(s.name)}</b><small>${esc(s.email)}</small></td><td>${esc(s.platoon)}</td><td>${esc(as.title)}</td><td>${esc(status(submission))}</td><td>${submission ? `<button class="secondary" data-answers="${esc(submission.id)}">Анкета</button>` : '—'}</td></tr>`).join('') || '<tr><td colspan="5">Анкеты не найдены. Измените поиск или фильтры.</td></tr>'}</tbody></table></div>`;
        view.querySelectorAll('[data-answers]').forEach(b => b.onclick = () => answers(b.dataset.answers));
    }

    function openDialog(html) {
        const dialog = view.querySelector('#form-dialog');
        dialog.innerHTML = html;
        if (!dialog.open) dialog.showModal();
        dialog.querySelector('[data-close]')?.addEventListener('click', () => dialog.close());
        return dialog;
    }

    async function answers(id) {
        openDialog('<h2>Загрузка анкеты…</h2>');
        try {
            const d = await api('/submissions/' + encodeURIComponent(id));
            const s = d.submission, who = d.student;
            openDialog(`<div class="section-title"><div><h2>${esc(who?.name || 'Студент')}</h2><p>Взвод ${esc(who?.platoon || '')} · ${esc(d.form.title)}</p></div><button class="secondary" data-close>Закрыть</button></div>
   <p class="${s.completed ? '' : 'draft-note'}">${s.completed ? `Сдана ${fmtDate(s.completedAt)}.` : `Не сдана: заполнено разделов ${d.progress.complete.length} из ${d.progress.total}. Показан черновик. Уровни потенциала появятся после сдачи.`}</p>${d.potential ? window.potentialView.card(d.potential, esc) : ''}<h3 class="answers-heading">Ответы</h3>${window.formView.answers(d.form, s, esc)}`);
        } catch (err) {
            openDialog(`<p class="error">${esc(err.message)}</p><button class="secondary" data-close>Закрыть</button>`);
        }
    }

    async function preview() {
        openDialog('<h2>Загрузка опросника…</h2>');
        try {
            form = form || await api('/forms/' + FORM);
            openDialog(`<div class="section-title"><div><h2>${esc(form.title)}</h2><p>Опросник версии ${esc(form.version)}: разделы и вопросы без ответов. * — обязательные вопросы.</p></div><button class="secondary" data-close>Закрыть</button></div>${window.formView.questions(form, esc)}`);
        } catch (err) {
            openDialog(`<p class="error">${esc(err.message)}</p><button class="secondary" data-close>Закрыть</button>`);
        }
    }

    view.innerHTML = '<p role="status">Загрузка назначений и анкет…</p>';
    try {
        await reload();
    } catch (e) {
        view.innerHTML = `<p class="error">${esc(e.message)}</p><a href="/teacher">Войти в кабинет</a>`;
    }
};
