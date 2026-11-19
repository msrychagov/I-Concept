// The home page of the cabinet and «Прохождение» (decided 18 November 2026,
// К2 and К4): the home page is cards only — four big ones (the platoons and the
// three kinds of materials) and four tiles (the students, the management of the
// platoons, the passing statistics, the model); the statistics that used to be
// under the cards live behind the tile «Прохождение».
(() => {
    const percent = (n, total) => total ? Math.round(n / total * 100) : 0;
    const stats = rows => ({
        total: rows.length,
        done: rows.filter(r => r.record?.completed).length,
        started: rows.filter(r => r.record && !r.record.completed).length,
        waiting: rows.filter(r => !r.record).length
    });

    // Pairs «student — assignment» of one kind of material.
    function rowsFor(data, scope = 'all', platoon = '') {
        const records = new Map(data.records.map(r => [r.assignmentId + ':' + r.studentId, r]));
        const rows = [];
        for (const assignment of data.assignments) {
            if (scope === 'active' && !assignment.active) continue;
            for (const student of data.students) {
                if (platoon && student.platoon !== platoon) continue;
                const record = records.get(assignment.id + ':' + student.id);
                // Preserve existing attempts, but do not count inaccessible new assignments.
                if (!record && (student.disabled || (student.vuc && !student.platoonId) || (assignment.platoon && assignment.platoon !== student.platoon))) continue;
                rows.push({student, record});
            }
        }
        return rows;
    }

    // The test and the autobiography, their students and platoons.
    async function loadDatasets(api) {
        const [concept, autobiography] = await Promise.all([api('/dashboard'), api('/forms/autobiography/overview')]);
        const datasets = [{...concept, name: 'Диагностическое тестирование', records: concept.sessions}, {
            ...autobiography,
            name: 'Автобиография',
            records: autobiography.submissions
        }];
        const students = [...new Map(datasets.flatMap(d => d.students).map(s => [s.id, s])).values()];
        const platoons = [...new Set([...students.map(s => s.platoon), ...datasets.flatMap(d => d.assignments.map(a => a.platoon).filter(Boolean))])].sort((a, b) => a.localeCompare(b, 'ru', {numeric: true}));
        return {datasets, students, platoons};
    }

    const icon = name => `<img class="ui-icon" src="/assets/images/icons/${name}.svg" alt="" width="26" height="26">`;

    window.teachingDashboard = async function ({view, api, esc}) {
        view.innerHTML = '<div class="testing-overview home-page"><p role="status">Загрузка…</p></div>';
        const root = view.firstElementChild;
        const go = (tab, options) => window.teacherGo(tab, options);
        const routes = {
            platoons: () => go('people', {start: 'platoons'}),
            students: () => go('people', {start: 'students'}),
            autobiography: () => go('autobiography'),
            testing: () => go('profiles'),
            essay: () => go('essay'),
            management: () => go('platoons'),
            progress: () => go('progress'),
            model: () => go('model')
        };
        const card = (to, picture, emblem, value, title, note) => `<button type="button" class="metric-link" data-go="${to}"><img class="metric-picture" src="/assets/images/reference/${picture}.jpg" alt=""><span class="metric-emblem">${icon(emblem)}</span><strong>${esc(value)}</strong><span class="metric-title">${esc(title)}</span><small class="metric-note">${esc(note)}</small></button>`;
        const tile = (to, emblem, title, note) => `<button type="button" class="path-tile" data-go="${to}"><span class="metric-emblem">${icon(emblem)}</span><b>${esc(title)}</b><small>${esc(note)}</small></button>`;

        function draw({datasets, students, platoons}, essays) {
            const [test, autobiography] = datasets.map(d => stats(rowsFor(d)));
            const all = stats(datasets.flatMap(d => rowsFor(d)));
            const essay = essays ? essays.assignments.reduce((sum, a) => ({done: sum.done + a.counts.submitted, total: sum.total + a.counts.students}), {done: 0, total: 0}) : null;
            root.innerHTML = `<header class="page-heading"><div><div class="eyebrow">Кабинет преподавателя</div><h1>Главная</h1><p>Выберите карточку: внутри — следующий шаг, назад — по строке пути вверху.</p><p class="home-motto">«Знание · Ответственность · Служение»</p></div></header>
   <div id="overview-error" role="alert"></div>
   <div class="dashboard-metrics home-cards">${card('platoons', 'card-upbringing', 'users', platoons.filter(Boolean).length, 'Взвода', 'Состав и общее по взводу')}${card('autobiography', 'card-study', 'file', `${autobiography.done} / ${autobiography.total}`, 'Автобиография', 'Сдано анкет')}${card('testing', 'card-daily', 'target', `${test.done} / ${test.total}`, 'Диагностическое тестирование', 'Пройдено тестирований')}${card('essay', 'card-knowledge', 'book', essay ? `${essay.done} / ${essay.total}` : '—', 'Эссе', 'Сдано эссе · автобиографическая память')}</div>
   <div class="path-tiles home-tiles">${tile('students', 'graduation', 'Студенты', `${students.length} · поиск по фамилии`)}${tile('management', 'shield', 'Управление взводом', 'Взводы, приглашения, выпуски')}${tile('progress', 'check-circle', 'Прохождение', `Пройдено назначений ${all.done} из ${all.total}`)}${tile('model', 'book-open', 'Модель и методика', 'Назначение и расчёт диагностики')}</div>`;
            root.querySelectorAll('[data-go]').forEach(b => b.onclick = () => routes[b.dataset.go]());
        }

        try {
            const [data, essays] = await Promise.all([loadDatasets(api), api('/essays').catch(() => null)]);
            if (root.isConnected) draw(data, essays);
        } catch (err) {
            if (root.isConnected) {
                root.innerHTML = `<h1>Главная</h1><p class="error" role="alert">${esc(err.message)}</p><button id="retry-overview">Повторить</button>`;
                root.querySelector('#retry-overview').onclick = () => window.teachingDashboard({view, api, esc})
            }
        }
    };

    // «Прохождение»: the statistics that were under the cards of the home page, unchanged.
    window.teachingProgress = async function ({view, api, esc}) {
        view.innerHTML = '<header class="page-heading"><div><div class="eyebrow">Кабинет преподавателя</div><h1>Прохождение</h1><p class="muted">Общая статистика по вашим назначениям и студентам</p></div></header><div class="testing-overview"><p role="status">Загрузка общей статистики…</p></div>';
        const root = view.querySelector('.testing-overview');
        let datasets = [], students = [], platoons = [], scope = 'all', platoon = '';

        function bar(s, label) {
            if (!s.total) return '<div class="testing-bar testing-bar-empty"><span>Нет назначений</span></div>';
            return `<div class="testing-bar" role="img" aria-label="${esc(label)}: завершено ${s.done}, в процессе ${s.started}, не начато ${s.waiting}, всего ${s.total}"><span class="testing-done" style="width:${s.done / s.total * 100}%"></span><span class="testing-started" style="width:${s.started / s.total * 100}%"></span><span class="testing-waiting" style="width:${s.waiting / s.total * 100}%"></span></div>`;
        }

        function draw() {
            const rows = datasets.map(d => rowsFor(d, scope, platoon)), totals = rows.map(stats);
            const shownPlatoons = platoons.filter(p => !platoon || p === platoon);
            root.innerHTML = `<div id="overview-error" role="alert"></div>
   <section class="surface testing-summary"><div class="section-title"><div><h2>Общая статистика</h2></div></div>
   <div class="toolbar testing-filters"><label>Назначения<select id="overview-scope"><option value="all" ${scope === 'all' ? 'selected' : ''}>Все, включая закрытые</option><option value="active" ${scope === 'active' ? 'selected' : ''}>Только активные</option></select></label><label>Взвод<select id="overview-platoon"><option value="">Все взводы</option>${platoons.filter(Boolean).map(p => `<option value="${esc(p)}" ${p === platoon ? 'selected' : ''}>${esc(p)}</option>`).join('')}</select></label></div>
   <div class="testing-legend"><span><i class="testing-done"></i>Завершено</span><span><i class="testing-started"></i>В процессе</span><span><i class="testing-waiting"></i>Не начато</span></div>
   <div class="testing-type-grid">${datasets.map((d, i) => {
                const s = totals[i], done = s.total ? s.done / s.total * 100 : 0,
                    started = s.total ? (s.done + s.started) / s.total * 100 : 0;
                return `<article class="testing-type-card"><div class="testing-type-heading"><span class="testing-type-number">0${i + 1}</span><h3>${esc(d.name)}</h3></div><div class="testing-type-body"><div class="testing-ring" style="--done:${done}%;--started:${started}%" role="img" aria-label="${esc(d.name)}: завершено ${s.done} из ${s.total}"><div><strong>${s.total ? percent(s.done, s.total) + '%' : '—'}</strong><span>${s.total ? 'завершено' : 'нет назначений'}</span></div></div><dl class="testing-counts"><div><dt>Завершено</dt><dd>${s.done}</dd></div><div><dt>В процессе</dt><dd>${s.started}</dd></div><div><dt>Не начато</dt><dd>${s.waiting}</dd></div><div><dt>Всего назначено</dt><dd>${s.total}</dd></div></dl></div></article>`;
            }).join('')}</div>
   <p class="muted testing-caption">Считаются пары «студент — назначение»: у одного студента может быть несколько тестирований. Проценты показывают прохождение, а не результаты диагностики. Назначить можно в карточках «Автобиография» и «Диагностическое тестирование» на главной.</p></section>
   <section class="surface"><div class="section-title"><div><div class="eyebrow">Сравнение взводов</div><h2>Прохождение по взводам</h2></div></div>
   <div class="testing-platoon-list">${shownPlatoons.map(p => `<article class="testing-platoon"><header><h3>${p ? 'Взвод ' + esc(p) : 'Без взвода'}</h3><span>${students.filter(s => s.platoon === p).length} студентов</span></header><div class="testing-platoon-types">${datasets.map((d, i) => {
                const s = stats(rows[i].filter(r => r.student.platoon === p));
                return `<div class="testing-platoon-type"><div class="testing-bar-heading"><b>${esc(d.name)}</b><strong>${s.total ? percent(s.done, s.total) + '%' : '—'}</strong></div>${bar(s, d.name)}<p>${s.total ? `${s.done} из ${s.total} завершено <span>· ${s.started} в процессе · ${s.waiting} не начато</span>` : 'Тестирование ещё не назначено'}</p></div>`
            }).join('')}</div></article>`).join('') || '<p class="muted">Пока нет данных по взводам. Добавьте студентов в карточке «Управление взводом» на главной.</p>'}</div></section>`;
            root.querySelector('#overview-scope').onchange = e => {
                scope = e.target.value;
                draw()
            };
            root.querySelector('#overview-platoon').onchange = e => {
                platoon = e.target.value;
                draw()
            };
        }

        try {
            ({datasets, students, platoons} = await loadDatasets(api));
            if (platoon && !platoons.includes(platoon)) platoon = '';
            if (root.isConnected) draw();
        } catch (err) {
            if (root.isConnected) {
                root.innerHTML = `<p class="error" role="alert">${esc(err.message)}</p><button id="retry-progress">Повторить</button>`;
                root.querySelector('#retry-progress').onclick = () => window.teachingProgress({view, api, esc})
            }
        }
    };
})();
