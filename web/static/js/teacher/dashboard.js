window.teachingDashboard = async function ({view, api, esc}) {
    view.innerHTML = '<div class="testing-overview"><p role="status">Загрузка общей статистики…</p></div>';
    const root = view.firstElementChild;
    let datasets = [], students = [], platoons = [], scope = 'all', platoon = '';
    const percent = (n, total) => total ? Math.round(n / total * 100) : 0;
    const stats = rows => ({
        total: rows.length,
        done: rows.filter(r => r.record?.completed).length,
        started: rows.filter(r => r.record && !r.record.completed).length,
        waiting: rows.filter(r => !r.record).length
    });
    const rowsFor = data => {
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
    };

    function bar(s, label) {
        if (!s.total) return '<div class="testing-bar testing-bar-empty"><span>Нет назначений</span></div>';
        return `<div class="testing-bar" role="img" aria-label="${esc(label)}: завершено ${s.done}, в процессе ${s.started}, не начато ${s.waiting}, всего ${s.total}"><span class="testing-done" style="width:${s.done / s.total * 100}%"></span><span class="testing-started" style="width:${s.started / s.total * 100}%"></span><span class="testing-waiting" style="width:${s.waiting / s.total * 100}%"></span></div>`;
    }

    function draw() {
        const rows = datasets.map(rowsFor), totals = rows.map(stats), all = stats(rows.flat());
        const shownStudents = students.filter(s => !platoon || s.platoon === platoon);
        const shownPlatoons = platoons.filter(p => !platoon || p === platoon);
        const assignments = datasets.flatMap(d => d.assignments).filter(a => a.active && (!platoon || !a.platoon || a.platoon === platoon));
        root.innerHTML = `<header class="page-heading"><div><div class="eyebrow">Кабинет преподавателя</div><h1>Главная</h1><p class="muted">Общая статистика по вашим назначениям и студентам</p></div><button id="refresh-overview" class="secondary">Обновить</button></header>
   <div id="overview-error" role="alert"></div>
   <div class="dashboard-metrics"><article><strong>${shownStudents.length}</strong>Студентов</article><article><strong>${shownPlatoons.filter(Boolean).length}</strong>Взводов</article><article><strong>${assignments.length}</strong>Активных тестирований</article><article><strong>${all.done} / ${all.total}</strong>Пройдено назначений</article></div>
   <section class="surface testing-summary"><div class="section-title"><div><div class="eyebrow">Два вида тестирования</div><h2>Общая картина прохождения</h2></div></div>
   <div class="toolbar testing-filters"><label>Назначения<select id="overview-scope"><option value="all" ${scope === 'all' ? 'selected' : ''}>Все, включая закрытые</option><option value="active" ${scope === 'active' ? 'selected' : ''}>Только активные</option></select></label><label>Взвод<select id="overview-platoon"><option value="">Все взводы</option>${platoons.filter(Boolean).map(p => `<option value="${esc(p)}" ${p === platoon ? 'selected' : ''}>${esc(p)}</option>`).join('')}</select></label></div>
   <div class="testing-legend"><span><i class="testing-done"></i>Завершено</span><span><i class="testing-started"></i>В процессе</span><span><i class="testing-waiting"></i>Не начато</span></div>
   <div class="testing-type-grid">${datasets.map((d, i) => {
            const s = totals[i], done = s.total ? s.done / s.total * 100 : 0,
                started = s.total ? (s.done + s.started) / s.total * 100 : 0;
            return `<article class="testing-type-card"><div class="testing-type-heading"><span class="testing-type-number">0${i + 1}</span><h3>${esc(d.name)}</h3></div><div class="testing-type-body"><div class="testing-ring" style="--done:${done}%;--started:${started}%" role="img" aria-label="${esc(d.name)}: завершено ${s.done} из ${s.total}"><div><strong>${s.total ? percent(s.done, s.total) + '%' : '—'}</strong><span>${s.total ? 'завершено' : 'нет назначений'}</span></div></div><dl class="testing-counts"><div><dt>Завершено</dt><dd>${s.done}</dd></div><div><dt>В процессе</dt><dd>${s.started}</dd></div><div><dt>Не начато</dt><dd>${s.waiting}</dd></div><div><dt>Всего назначено</dt><dd>${s.total}</dd></div></dl></div></article>`;
        }).join('')}</div>
   <p class="muted testing-caption">Считаются пары «студент — назначение»: у одного студента может быть несколько тестирований. Проценты показывают прохождение, а не результаты диагностики. Назначить тестирование можно в разделах «Я концепция» и «Автобиография».</p></section>
   <section class="surface"><div class="section-title"><div><div class="eyebrow">Сравнение взводов</div><h2>Прохождение по взводам</h2></div></div>
   <div class="testing-platoon-list">${shownPlatoons.map(p => `<article class="testing-platoon"><header><h3>${p ? 'Взвод ' + esc(p) : 'Без взвода'}</h3><span>${students.filter(s => s.platoon === p).length} студентов</span></header><div class="testing-platoon-types">${datasets.map((d, i) => {
            const s = stats(rows[i].filter(r => r.student.platoon === p));
            return `<div class="testing-platoon-type"><div class="testing-bar-heading"><b>${esc(d.name)}</b><strong>${s.total ? percent(s.done, s.total) + '%' : '—'}</strong></div>${bar(s, d.name)}<p>${s.total ? `${s.done} из ${s.total} завершено <span>· ${s.started} в процессе · ${s.waiting} не начато</span>` : 'Тестирование ещё не назначено'}</p></div>`
        }).join('')}</div></article>`).join('') || '<p class="muted">Пока нет данных по взводам. Добавьте студентов в разделе «Управление взводом».</p>'}</div></section>`;
        root.querySelector('#overview-scope').onchange = e => {
            scope = e.target.value;
            draw()
        };
        root.querySelector('#overview-platoon').onchange = e => {
            platoon = e.target.value;
            draw()
        };
        root.querySelector('#refresh-overview').onclick = async e => {
            const button = e.currentTarget;
            button.disabled = true;
            try {
                await load()
            } catch (err) {
                if (root.isConnected) root.querySelector('#overview-error').innerHTML = `<p class="error">${esc(err.message)}</p>`
            } finally {
                button.disabled = false
            }
        };
    }

    async function load() {
        const [concept, autobiography] = await Promise.all([api('/dashboard'), api('/forms/autobiography/overview')]);
        if (!root.isConnected) return;
        datasets = [{...concept, name: 'Я концепция', records: concept.sessions}, {
            ...autobiography,
            name: 'Автобиография',
            records: autobiography.submissions
        }];
        students = [...new Map(datasets.flatMap(d => d.students).map(s => [s.id, s])).values()];
        platoons = [...new Set([...students.map(s => s.platoon), ...datasets.flatMap(d => d.assignments.map(a => a.platoon).filter(Boolean))])].sort((a, b) => a.localeCompare(b, 'ru', {numeric: true}));
        if (platoon && !platoons.includes(platoon)) platoon = '';
        draw();
    }

    try {
        await load()
    } catch (err) {
        if (root.isConnected) {
            root.innerHTML = `<h1>Главная</h1><p class="error" role="alert">${esc(err.message)}</p><button id="retry-overview">Повторить</button>`;
            root.querySelector('#retry-overview').onclick = () => window.teachingDashboard({view, api, esc})
        }
    }
};
