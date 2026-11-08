// The path «взвод → студент» from the home page (decided 6 November 2026 by the
// sketch of the supervisor): the platoons → a platoon with its students and
// what concerns the whole platoon → a student with three icons: the
// autobiography, the results of testing and the autobiographical memory (the
// essay). Every icon opens the window of its section; the data are those of
// the sections, nothing is counted here.
window.teacherPlatoonView = async function ({view, api, esc, start = 'platoons', home}) {
    const fmtDate = v => v ? new Date(v).toLocaleDateString('ru-RU') : '';
    const byNumber = (a, b) => a.localeCompare(b, 'ru', {numeric: true});
    const icon = (name, size = 26) => `<img class="ui-icon" src="/assets/images/icons/${name}.svg" alt="" width="${size}" height="${size}">`;
    const studentsWord = n => n % 10 === 1 && n % 100 !== 11 ? 'студент' : [2, 3, 4].includes(n % 10) && ![12, 13, 14].includes(n % 100) ? 'студента' : 'студентов';
    const KINDS = {
        autobiography: {title: 'Автобиография', short: 'автобиография', icon: 'file', done: 'Сдана', going: 'Заполняет', none: 'Не заполнял'},
        test: {title: 'Результаты тестирования', short: 'тест', icon: 'target', done: 'Пройдено', going: 'Проходит', none: 'Не проходил'},
        essay: {title: 'Автобиографическая память', short: 'эссе', icon: 'book', done: 'Сдано', going: 'Пишет', none: 'Не писал'}
    };
    let data, place = {screen: start}, search = '', generation = 0, stops = [];

    // Everything the sections already give; a section that failed leaves its icon empty.
    async function load() {
        const [platoons, tests, forms, essays] = await Promise.all([
            api('/teachers/me/platoons'),
            api('/dashboard').catch(() => null),
            api('/forms/autobiography/overview').catch(() => null),
            api('/essays').catch(() => null)
        ]);
        data = {platoons, tests, forms, essays};
    }

    const students = () => [...data.platoons.students].sort((a, b) => a.name.localeCompare(b.name, 'ru'));
    const platoonNames = () => {
        const names = data.platoons.platoons.map(p => p.name);
        if (data.platoons.students.some(s => !s.platoon)) names.push('');
        return [...new Set(names)].sort(byNumber);
    };
    const platoonTitle = p => p ? 'Взвод ' + p : 'Без взвода';

    // The passings of one student, newest first; the newest finished one opens.
    function passings(kind, studentId) {
        const titles = list => new Map((list || []).map(a => [a.id, a.title]));
        let rows = [];
        if (kind === 'autobiography' && data.forms) {
            const t = titles(data.forms.assignments);
            rows = data.forms.submissions.filter(s => s.studentId === studentId).map(s => ({id: s.id, date: s.completedAt || s.updated, completed: s.completed, title: t.get(s.assignmentId) || 'Автобиография'}));
        } else if (kind === 'test' && data.tests) {
            const t = titles(data.tests.assignments);
            rows = data.tests.sessions.filter(s => s.studentId === studentId).map(s => ({id: s.id, date: s.updated, completed: s.completed, title: t.get(s.assignmentId) || s.title || 'Тестирование'}));
        } else if (kind === 'essay' && data.essays) {
            rows = data.essays.assignments.flatMap(a => a.students.filter(s => s.studentId === studentId && s.submissionId).map(s => ({id: s.submissionId, date: s.completedAt || a.created, completed: s.status === 'submitted', title: a.title})));
        }
        rows.sort((a, b) => String(b.date).localeCompare(String(a.date)));
        const open = rows.find(r => r.completed) || rows[0];
        return {rows, open};
    }

    function stateLine(kind, list) {
        const k = KINDS[kind];
        if (!list.open) return k.none;
        const more = list.rows.length > 1 ? ` · прохождений: ${list.rows.length}` : '';
        return (list.open.completed ? `${k.done} ${fmtDate(list.open.date)}` : k.going) + more;
    }

    const slicesOf = list => list.rows.map(r => ({id: r.id, label: `${fmtDate(r.date)} · ${r.title}${r.completed ? '' : ' · не завершено'}`}));

    // ---- drawing -----------------------------------------------------------
    function crumbs(items) {
        const parts = [{label: 'Главная', go: 'home'}, ...items];
        return `<nav class="path-crumbs" aria-label="Путь">${parts.map((p, i) => i === parts.length - 1 ? `<span aria-current="page">${esc(p.label)}</span>` : `<button type="button" class="text-button" data-crumb="${i}">${esc(p.label)}</button><span class="path-sep" aria-hidden="true">›</span>`).join('')}</nav>`;
    }

    function tile({icon: name, title, note, attrs = '', pressed, disabled}) {
        return `<button type="button" class="path-tile${pressed ? ' active' : ''}" ${attrs} ${pressed !== undefined ? `aria-pressed="${pressed}"` : ''} ${disabled ? 'disabled' : ''}><span class="metric-emblem">${icon(name)}</span><b>${esc(title)}</b>${note ? `<small>${esc(note)}</small>` : ''}</button>`;
    }

    function frame(trail, heading, body) {
        view.innerHTML = `<div class="path-view">${crumbs(trail)}<header class="page-heading"><div>${heading}</div></header>${body}</div><dialog id="pv-dialog" class="result-dialog"></dialog><dialog id="result-dialog" class="result-dialog"></dialog>`;
        view.querySelectorAll('[data-crumb]').forEach(b => b.onclick = () => {
            const target = [{go: 'home'}, ...trail][Number(b.dataset.crumb)];
            if (target.go === 'home') return home();
            go(target.place);
        });
    }

    // A new screen scrolls to the top; a tile inside the same screen does not.
    function go(next, scroll = true) {
        stops.forEach(stop => stop());
        stops = [];
        generation++;
        place = next;
        draw();
        if (scroll) scrollTo({top: 0, behavior: 'smooth'});
    }

    function draw() {
        if (place.screen === 'students') return drawStudents();
        if (place.screen === 'platoon') return drawPlatoon();
        if (place.screen === 'student') return drawStudent();
        drawPlatoons();
    }

    function drawPlatoons() {
        const names = platoonNames();
        frame([{label: 'Взвода'}], '<div class="eyebrow">Кабинет преподавателя</div><h1>Взвода</h1><p class="muted">Выберите взвод: в нём — состав и общее по взводу.</p>',
            names.length ? `<div class="path-tiles">${names.map(p => {
                const n = data.platoons.students.filter(s => s.platoon === p).length;
                return tile({icon: 'users', title: platoonTitle(p), note: `${n} ${studentsWord(n)}`, attrs: `data-platoon="${esc(p)}"`});
            }).join('')}</div>` : '<p class="muted">Взводов пока нет. Создайте их на странице «Управление взводом».</p>');
        view.querySelectorAll('[data-platoon]').forEach(b => b.onclick = () => go({screen: 'platoon', platoon: b.dataset.platoon, part: 'list'}));
    }

    // One line under a name: what the student has handed in.
    function shortState(s) {
        const done = Object.keys(KINDS).filter(kind => passings(kind, s.id).rows.some(r => r.completed)).map(kind => KINDS[kind].short);
        return done.length ? 'Сдано: ' + done.join(' · ') : 'Пока ничего не сдал';
    }

    function studentList(list, from) {
        if (!list.length) return '<p class="muted">Студентов не найдено.</p>';
        return `<ul class="path-people">${list.map(s => `<li><button type="button" class="path-person" data-student="${esc(s.id)}"><span class="path-person-icon">${icon('user', 20)}</span><span><b>${esc(s.name)}</b><small>${from === 'students' ? esc(platoonTitle(s.platoon)) + ' · ' : ''}${esc(shortState(s))}${s.disabled ? ' · доступ закрыт' : ''}</small></span></button></li>`).join('')}</ul>`;
    }

    function bindStudents(from) {
        view.querySelectorAll('[data-student]').forEach(b => b.onclick = () => go({screen: 'student', id: b.dataset.student, from}));
    }

    function drawStudents() {
        frame([{label: 'Студенты'}], '<div class="eyebrow">Кабинет преподавателя</div><h1>Студенты</h1>',
            `<section class="surface"><div class="dashboard-filters"><label>ФИО<input id="pv-search" placeholder="Поиск по фамилии или имени" value="${esc(search)}"></label></div><div id="pv-list"></div></section>`);
        const list = () => {
            const q = search.trim().toLowerCase();
            view.querySelector('#pv-list').innerHTML = studentList(students().filter(s => !q || s.name.toLowerCase().includes(q)), 'students');
            bindStudents({screen: 'students'});
        };
        view.querySelector('#pv-search').oninput = e => {
            search = e.target.value;
            list();
        };
        list();
    }

    function drawPlatoon() {
        const p = place.platoon, members = students().filter(s => s.platoon === p);
        frame([{label: 'Взвода', place: {screen: 'platoons'}}, {label: platoonTitle(p)}], `<div class="eyebrow">Взвод</div><h1>${esc(platoonTitle(p))}</h1>`,
            `<div class="path-tiles path-tiles-two">${tile({icon: 'users', title: 'Состав взвода', note: `${members.length} ${studentsWord(members.length)}`, attrs: 'data-part="list"', pressed: place.part === 'list'})}${tile({icon: 'chart', title: 'Общее по взводу', note: 'Потенциал, тестирование, эссе выпуска', attrs: 'data-part="common"', pressed: place.part === 'common'})}</div><div id="pv-part"></div>`);
        view.querySelectorAll('[data-part]').forEach(b => b.onclick = () => go({...place, part: b.dataset.part, topic: undefined}, false));
        const box = view.querySelector('#pv-part');
        if (place.part === 'list') {
            box.innerHTML = `<section class="surface"><h2>Состав взвода</h2>${studentList(members, 'platoon')}</section>`;
            return bindStudents({screen: 'platoon', platoon: p, part: 'list'});
        }
        box.innerHTML = `<div class="path-tiles">${Object.entries(KINDS).map(([kind, k]) => tile({icon: k.icon, title: k.title, note: {autobiography: 'Потенциал по взводу', test: 'Уровни методик и вывод по взводу', essay: 'Сводка по выпуску'}[kind], attrs: `data-topic="${kind}"`, pressed: place.topic === kind})).join('')}</div><section class="surface" id="pv-topic" ${place.topic ? '' : 'hidden'}></section>`;
        box.querySelectorAll('[data-topic]').forEach(b => b.onclick = () => go({...place, topic: place.topic === b.dataset.topic ? undefined : b.dataset.topic}, false));
        const topic = box.querySelector('#pv-topic');
        if (place.topic === 'autobiography') commonAutobiography(topic, p);
        if (place.topic === 'test') commonTest(topic, p);
        if (place.topic === 'essay') commonEssay(topic, p);
    }

    function drawStudent() {
        const s = data.platoons.students.find(x => x.id === place.id);
        if (!s) return go({screen: 'platoons'});
        const trail = place.from.screen === 'students'
            ? [{label: 'Студенты', place: {screen: 'students'}}, {label: s.name}]
            : [{label: 'Взвода', place: {screen: 'platoons'}}, {label: platoonTitle(s.platoon), place: place.from}, {label: s.name}];
        const lists = Object.fromEntries(Object.keys(KINDS).map(kind => [kind, passings(kind, s.id)]));
        frame(trail, `<div class="eyebrow">${esc(platoonTitle(s.platoon))}</div><h1>${esc(s.name)}</h1>`,
            `<div class="path-tiles">${Object.entries(KINDS).map(([kind, k]) => tile({icon: k.icon, title: k.title, note: stateLine(kind, lists[kind]), attrs: `data-kind="${kind}"`, disabled: !lists[kind].open})).join('')}</div><p class="muted path-hint">Открывается последнее прохождение; другие — по датам вверху окна.</p>`);
        view.querySelectorAll('[data-kind]').forEach(b => b.onclick = () => openKind(b.dataset.kind, lists[b.dataset.kind]));
    }

    function openKind(kind, list) {
        const slices = slicesOf(list), id = list.open.id;
        if (kind === 'test') return window.teacherResult({view, api, esc, id, slices});
        const dialog = view.querySelector('#pv-dialog');
        if (kind === 'autobiography') return window.autobiographyWindow.open({dialog, api, esc, id, slices});
        return window.essayWindow.open({dialog, api, id, slices});
    }

    // ---- what concerns the whole platoon -----------------------------------
    const current = () => {
        const mine = generation;
        return () => mine === generation;
    };
    const failed = (box, err) => box.innerHTML = `<p class="error" role="alert">${esc(err.message || err)}</p>`;
    const openAutobiography = id => window.autobiographyWindow.open({dialog: view.querySelector('#pv-dialog'), api, esc, id});
    const openEssay = id => window.essayWindow.open({dialog: view.querySelector('#pv-dialog'), api, id});
    const openResult = id => window.teacherResult({view, api, esc, id});

    async function commonAutobiography(box, p) {
        const alive = current();
        box.innerHTML = '<h2>Потенциал по взводу</h2><p class="muted">Загрузка…</p>';
        try {
            const potential = await api('/forms/autobiography/potential?platoon=' + encodeURIComponent(p));
            if (!alive()) return;
            let selected = null;
            const drawPotential = () => {
                box.innerHTML = `<h2>Потенциал по взводу</h2>${window.potentialView.summary(potential, esc, selected)}${window.potentialView.students(potential, selected, esc)}`;
                box.querySelectorAll('[data-level]').forEach(b => b.onclick = () => {
                    const same = selected && selected.area === b.dataset.area && selected.level === b.dataset.level;
                    selected = same ? null : {area: b.dataset.area, level: b.dataset.level};
                    drawPotential();
                });
                box.querySelectorAll('[data-answers]').forEach(b => b.onclick = () => openAutobiography(b.dataset.answers));
            };
            drawPotential();
        } catch (err) {
            if (alive()) failed(box, err);
        }
    }

    async function commonTest(box, p) {
        const alive = current();
        box.innerHTML = '<h2>Результаты тестирования взвода</h2><div id="pv-conclusion" class="test-portrait"></div><div id="pv-values" class="values-block"></div><div id="pv-methods" aria-live="polite"><p class="muted">Загрузка…</p></div>';
        try {
            const summary = await api('/reports/methods?group=' + encodeURIComponent(p));
            if (!alive()) return;
            stops.push(window.testPortrait.mountPlatoon({element: box.querySelector('#pv-conclusion'), api, esc, group: p, platoon: summary.platoon, open: openResult}));
            let selected = null;
            const methods = box.querySelector('#pv-methods');
            // The table of the levels goes inside «Подробно» of the conclusion, not under every tab.
            box.querySelector('#pv-conclusion [data-pane="details"]')?.prepend(methods);
            const drawMethods = () => {
                methods.innerHTML = window.testPortrait.summary(summary, esc, selected, true) + window.testPortrait.students(summary, selected, esc);
                methods.querySelectorAll('[data-row]').forEach(b => b.onclick = () => {
                    const next = {row: b.dataset.row, level: b.dataset.level};
                    selected = selected && selected.row === next.row && selected.level === next.level ? null : next;
                    drawMethods();
                });
                methods.querySelectorAll('[data-session]').forEach(b => b.onclick = () => openResult(b.dataset.session));
            };
            drawMethods();
            const values = await api('/values/platoons/' + encodeURIComponent(p)).catch(() => null);
            if (!alive() || !values?.texts) return;
            const valuesBox = box.querySelector('#pv-values');
            // The values go inside «Подробно» of the conclusion, not under every tab.
            box.querySelector('#pv-conclusion [data-pane="details"]')?.append(valuesBox);
            valuesBox.innerHTML = `<h3>Ценности в ответах о деятельности</h3>${window.valuesBlock.summary(values, esc, {noun: 'студентов', of: 'студентов с рассказанными ответами'})}`;
            valuesBox.querySelectorAll('[data-session]').forEach(b => b.onclick = () => openResult(b.dataset.session));
        } catch (err) {
            if (alive()) failed(box, err);
        }
    }

    // The essay is summed up by the cohort, not by the platoon: the summary of
    // the cohort the platoon belongs to, the newest essay assignment first.
    async function commonEssay(box, p) {
        const alive = current();
        const heading = '<h2>Автобиографическая память</h2>';
        if (!data.essays) return failed(box, 'Не удалось загрузить эссе.');
        const cohorts = data.essays.cohorts.filter(c => c.platoons.includes(p));
        if (!cohorts.length) {
            box.innerHTML = `${heading}<p class="muted">Взвод не входит ни в один выпуск. Выпуск создаётся на странице «Управление взводом», раздел «Выпуски».</p>`;
            return;
        }
        const ids = new Set(cohorts.map(c => c.id));
        const given = data.essays.assignments.filter(a => ids.has(a.cohortId)).sort((a, b) => String(b.created).localeCompare(String(a.created)));
        if (!given.length) {
            box.innerHTML = `${heading}<p class="muted">Эссе выпуску «${esc(cohorts.map(c => c.name).join('», «'))}» ещё не выдавали. Выдать его можно в разделе «Эссе».</p>`;
            return;
        }
        const chosen = given.find(a => a.id === place.essay) || given[0];
        const cohort = cohorts.find(c => c.id === chosen.cohortId);
        const choice = given.length > 1 ? `<nav class="slice-bar" aria-label="Выданные эссе"><span class="muted">Выдачи:</span>${given.map(a => `<button type="button" class="${a.id === chosen.id ? '' : 'secondary'}" data-given="${esc(a.id)}" ${a.id === chosen.id ? 'aria-current="true" disabled' : ''}>${esc(a.title)} · ${fmtDate(a.created)}</button>`).join('')}</nav>` : '';
        box.innerHTML = `${heading}${choice}<p class="muted">${esc(chosen.title)}, выпуск «${esc(cohort.name)}» (${cohort.platoons.length > 1 ? 'взводы ' : 'взвод '}${esc(cohort.platoons.join(', '))}). Эссе сводится по выпуску: отдельной сводки по взводу нет.</p><div id="pv-essay-summary"><p class="muted">Загрузка…</p></div>`;
        box.querySelectorAll('[data-given]').forEach(b => b.onclick = () => go({...place, essay: b.dataset.given}, false));
        const slot = box.querySelector('#pv-essay-summary');
        let timer = 0;
        stops.push(() => clearTimeout(timer));
        const show = async () => {
            try {
                const [summary, values] = await Promise.all([api('/essays/assignments/' + encodeURIComponent(chosen.id) + '/summary'), api('/values/assignments/' + encodeURIComponent(chosen.id)).catch(() => null)]);
                if (!alive()) return;
                // While the summary is being made the block asks again.
                const poll = s => {
                    clearTimeout(timer);
                    if (s.status === 'queued' || s.status === 'working') timer = setTimeout(() => alive() && show(), 4000);
                };
                window.essayWindow.summary(slot, {summary, values, api, assignmentId: chosen.id, onEssay: openEssay, onChange: poll});
                poll(summary);
            } catch (err) {
                if (alive()) failed(slot, err);
            }
        };
        show();
    }

    view.innerHTML = '<p role="status">Загрузка взводов и студентов…</p>';
    try {
        await load();
        if (view.isConnected) draw();
    } catch (err) {
        view.innerHTML = `<h1>Взвода</h1><p class="error" role="alert">${esc(err.message)}</p><button id="pv-retry">Повторить</button>`;
        view.querySelector('#pv-retry').onclick = () => window.teacherPlatoonView({view, api, esc, start, home});
    }
};
