// The portrait by the test (task "test") and the conclusion by a platoon (task
// "test-platoon"). Above the text — the program's signature that it is the
// student's self-report; under it — the tables with the program's numbers.
// The portrait of v2 is the conclusion of a psychologist in seven sections; a
// portrait of v1 is still shown in its own format while v2 is being made.
window.portraitRenderers = window.portraitRenderers || {};
window.portraitRenderers.test = {
    text(output, esc, data) {
        if (output.components) return v2(output, esc, data || {});
        const paragraph = (title, text) => text ? `<h4>${esc(title)}</h4><p>${esc(text)}</p>` : '';
        const list = (title, items) => (items || []).length ? `<h4>${esc(title)}</h4><ul>${items.map(x => `<li>${esc(x)}</li>`).join('')}</ul>` : '';
        const core = output.core || {};
        return `<div class="test-portrait-text">
   ${paragraph('Портрет', output.portrait)}
   ${paragraph('Ценностно-смысловой компонент', core.cs)}
   ${paragraph('Эмоционально-волевой компонент', core.ev)}
   ${paragraph('Деятельностно-практический компонент', core.dp)}
   ${paragraph('Признаки: любовь, соборность, патриотизм', output.signs)}
   ${paragraph('Я-реальное и Я-идеальное', output.images)}
   ${paragraph('Условия среды ВУЦ', output.environment)}
   ${(output.activity || []).length ? `<h4>Виды деятельности</h4>${output.activity.map(a => `<p><b>${esc(a.kind ? a.kind[0].toUpperCase() + a.kind.slice(1) : '')}</b> — ${esc(a.level)}. ${esc(a.text)}</p>`).join('')}` : ''}
   ${paragraph('Связь условий, деятельности и Я-концепции', output.link)}
   ${list('На что опереться', output.support)}
   ${list('Над чем работать', output.growth)}
  </div>`;
    }
};

// recommendationList names the direction of the catalogue and the group of the
// image of every recommendation.
function testRecommendations(items, esc, data) {
    const directions = Object.fromEntries((data.catalogue || []).map(d => [d.id, d.name]));
    const groups = Object.fromEntries((data.defender || []).map((g, n) => [n + 1, g.name]));
    return (items || []).length ? `<h4>Рекомендации</h4><ul class="test-portrait-recommendations">${items.map(r => `<li><b>${esc(directions[r.code] || r.code)}</b>${groups[r.group] ? ` <span class="muted">(${esc(groups[r.group].toLowerCase())})</span>` : ''}. ${esc(r.text)}</li>`).join('')}</ul>` : '';
}

function v2(output, esc, data) {
    const paragraph = (title, text) => text ? `<h4>${esc(title)}</h4><p>${esc(text)}</p>` : '';
    const c = output.components || {}, levels = c.levels || {};
    const named = [['cs', 'ценностно-смысловой'], ['ev', 'эмоционально-волевой'], ['dp', 'деятельностно-практический']]
        .filter(([key]) => levels[key]).map(([key, name]) => `${name} — ${esc(levels[key])}`).join('; ');
    return `<div class="test-portrait-text">
   ${paragraph('Психологический портрет', output.portrait)}
   <h4>Компоненты Я-концепции</h4>${named ? `<p class="muted">Уровень сформированности: ${named}.</p>` : ''}<p>${esc(c.text || '')}</p>
   ${paragraph('Профессиональная Я-концепция', output.identity)}
   ${paragraph('Соответствие образу защитника Отечества', output.ideal)}
   ${paragraph('Деятельность и условия среды', output.activity)}
   ${testRecommendations(output.recommendations, esc, data)}
   ${paragraph('Общий вывод', output.conclusion)}
  </div>`;
}

window.portraitRenderers['test-platoon'] = {
    text(output, esc, data) {
        const paragraph = (title, text) => text ? `<h4>${esc(title)}</h4><p>${esc(text)}</p>` : '';
        return `<div class="test-portrait-text">
   ${paragraph('Состояние взвода', output.overview)}
   ${paragraph('Компоненты Я-концепции', output.components)}
   ${paragraph('Профессиональная Я-концепция', output.identity)}
   ${paragraph('Соответствие образу защитника Отечества', output.ideal)}
   ${paragraph('Деятельность и условия среды', output.activity)}
   ${testRecommendations(output.recommendations, esc, data || {})}
   ${(output.individual || []).length ? `<h4>Кому нужна отдельная работа</h4><ul>${output.individual.map(x => `<li>${esc(x)}</li>`).join('')}</ul>` : ''}
   ${paragraph('Общий вывод', output.conclusion)}
  </div>`;
    }
};

window.testPortrait = (() => {
    const COMPONENTS = {CS: 'Ценностно-смысловой', EV: 'Эмоционально-волевой', DP: 'Деятельностно-практический'};
    const SIGNATURE = 'По результатам тестирования: самооценка студента и методики';
    const PLATOON_SIGNATURE = 'По результатам тестирования взвода: самооценка студентов и методики';
    const number = v => v == null ? '—' : String(v).replace('.', ',');
    const capital = text => text ? text[0].toUpperCase() + text.slice(1) : '';

    function levels(m, esc) {
        const methods = Object.keys(COMPONENTS).map(code => {
            const rows = (m.methods || []).filter(x => x.component === code);
            return rows.length ? `<tr class="test-portrait-group"><th colspan="3">${esc(COMPONENTS[code])} компонент</th></tr>${rows.map(x => `<tr><td>${esc(x.name)}<small>${esc(x.method)}</small></td><td>${x.score == null ? '—' : `${x.score} <small>из ${x.min}–${x.max}</small>`}</td><td><span class="potential-level level-${esc(x.level || 'none')}">${esc(x.levelText)}</span></td></tr>`).join('')}` : '';
        }).join('');
        const kinds = (m.kinds || []).map(k => `<tr><td>${esc(k.name)}</td><td></td><td><span class="potential-level level-${esc(k.level || 'none')}">${esc(k.levelText)}</span></td></tr>`).join('');
        // The statuses of professional identity have ranges, not levels: a
        // marked status is not better or worse, so the ranges are not coloured.
        const identity = (m.identity || []).map(p => `<tr class="test-portrait-group"><th colspan="3">Профессиональная идентичность — ${esc(p.name.toLowerCase())}<small>${esc(p.method)}</small></th></tr>${p.statuses.map(x => `<tr><td>${esc(capital(x.name))}</td><td>${x.score == null ? '—' : `${x.score} <small>из 0–${x.max}</small>`}</td><td><span class="potential-level identity-range level-${x.range ? 'initial' : 'none'}">${esc(x.range || 'не определён')}</span></td></tr>`).join('')}<tr><td colspan="3">Ведущий статус: <b>${esc(p.leadingText)}</b></td></tr>`).join('');
        return `<div class="table-scroll"><table class="test-portrait-levels"><thead><tr><th>Показатель</th><th>Баллы</th><th>Уровень</th></tr></thead><tbody>${methods}<tr class="test-portrait-group"><th colspan="3">Виды деятельности (по анкете деятельностного компонента)</th></tr>${kinds}${identity}</tbody></table></div>
   <p class="muted">Уровни посчитаны программой по ключам методик из таблицы 8 руководителя${identity ? ', выраженность статусов профессиональной идентичности — по ключу методики А. А. Азбель, составленному на школьниках' : ''}; это самоотчёт студента, а не установленное качество личности.</p>`;
    }

    // The groups of the image of the defender: the real and the ideal self of
    // the student (0–10), the distance to the image and the situation the
    // program found, and the congruence of the images.
    function image(m, esc) {
        if (!(m.defender || []).length) return '';
        const rows = m.defender.map(g => `<tr><td>${esc(g.name)}<small>Я-реальное ${number(g.real)} · Я-идеальное ${number(g.ideal)} · до образа ${number(g.distance)}</small></td><td>${esc(g.situationText)}${g.note ? `<small>${esc(g.note)}</small>` : ''}</td></tr>`).join('');
        const c = m.congruence;
        return `<h4>Образ защитника Отечества</h4><div class="table-scroll"><table class="test-portrait-levels test-portrait-image"><thead><tr><th>Группа качеств</th><th>Ситуация</th></tr></thead><tbody>${rows}</tbody></table></div>
   ${c ? `<p>Согласованность Я-реального и Я-идеального: ${esc(c.text)}${c.value == null ? '' : ` <span class="muted">(коэффициент ${number(c.value)})</span>`}.</p>` : ''}
   <p class="muted">Образ — верх шкалы 0–10; ситуация — по расстоянию от Я-идеального до образа и от Я-реального до Я-идеального.</p>`;
    }

    function tables(m, esc) {
        return `<div class="test-portrait-numbers"><h4>Результаты методик</h4>${levels(m, esc)}${image(m, esc)}</div>`;
    }

    function signature(view, esc, text) {
        if (view.status !== 'ready') return '';
        const date = view.portrait.created ? new Date(view.portrait.created).toLocaleDateString('ru-RU') : '';
        return `<p class="portrait-signature muted">${esc(text)}${date ? ` · ${esc(date)}` : ''}</p>`;
    }

    // Mounts the block and returns the function that stops its polling.
    function mount({element, api, esc, id, methods, completed}) {
        if (methods && !completed) {
            element.innerHTML = `<h3>Портрет по результатам тестирования</h3><p class="muted">Текст портрета составится после завершения тестирования.</p>${tables(methods, esc)}`;
            return () => {};
        }
        if (!methods) {
            element.innerHTML = '<h3>Портрет по результатам тестирования</h3><p class="muted">В этом тестировании нет методик уровней Я-концепции, портрет не составляется. Включите «Методики уровней Я-концепции» при назначении.</p>';
            return () => {};
        }
        element.innerHTML = `<h3>Портрет по результатам тестирования</h3><div data-test-text aria-live="polite"></div>${tables(methods, esc)}`;
        const box = element.querySelector('[data-test-text]');
        let handle, alive = true;
        const run = (first = 'GET') => {
            handle?.stop();
            handle = window.portraitView.start(api, 'test', id, {
                alive: () => alive && box.isConnected,
                update: view => {
                    const ready = view.status === 'ready' ? window.portraitRenderers.test.text(view.portrait.output, esc, methods) : '';
                    box.innerHTML = window.portraitView.status(view, esc) + signature(view, esc, SIGNATURE) + ready;
                    box.querySelector('[data-portrait-retry]')?.addEventListener('click', () => run('POST'));
                }
            }, first);
        };
        run();
        return () => {
            alive = false;
            handle?.stop();
        };
    }

    // The platoon table "scale or kind × level" with counts; a count opens the
    // list of students, a name opens the result.
    function summary(data, esc, selected) {
        if (!data.sessions) return '<p class="muted">Завершённых тестирований с методиками пока нет.</p>';
        const head = `<tr><th>Показатель</th>${data.rows[0].levels.map(l => `<th>${esc(l.levelText)}</th>`).join('')}</tr>`;
        const group = (title, rows) => rows.length ? `<tr class="test-portrait-group"><th colspan="${rows[0].levels.length + 1}">${esc(title)}</th></tr>${rows.map(r => `<tr><th scope="row">${esc(r.name)}</th>${r.levels.map(l => {
            const on = selected && selected.row === r.kind + ':' + r.id && selected.level === l.level;
            return `<td>${l.students.length ? `<button class="secondary potential-count${on ? ' active' : ''}" data-row="${esc(r.kind + ':' + r.id)}" data-level="${esc(l.level)}" aria-pressed="${on}">${l.students.length}</button>` : '<span class="muted">0</span>'}</td>`;
        }).join('')}</tr>`).join('')}` : '';
        const body = Object.keys(COMPONENTS).map(code => group(COMPONENTS[code] + ' компонент', data.rows.filter(r => r.kind === 'method' && r.component === code))).join('')
            + group('Виды деятельности', data.rows.filter(r => r.kind === 'activity'));
        return `<p class="muted">Учтено тестирований: ${data.sessions} — последнее завершённое с методиками у каждого студента. Уровни посчитаны программой по ключам методик.</p><div class="table-scroll"><table class="potential-summary"><thead>${head}</thead><tbody>${body}</tbody></table></div>`;
    }

    function students(data, selected, esc) {
        const row = data?.rows?.find(r => r.kind + ':' + r.id === selected?.row);
        const level = row?.levels.find(l => l.level === selected.level);
        if (!level) return '';
        return `<div class="potential-students"><h3>${esc(row.name)} — ${esc(level.levelText)}: ${level.students.length}</h3><ul>${level.students.map(s => `<li><button class="text-button" data-session="${esc(s.sessionId)}">${esc(s.name)}</button> <span class="muted">взвод ${esc(s.group)}</span></li>`).join('')}</ul></div>`;
    }

    const SITUATIONS = [['achieved', 'Принят и достигнут'], ['far', 'Принят, до него далеко'], ['not_accepted', 'Не принят как свой'], ['below', 'Я-идеальное ниже Я-реального'], ['', 'Не определена']];

    // The groups of the image over the platoon: students in every situation; a
    // count opens the list, a name opens the result.
    function situations(platoon, esc, selected) {
        const shown = SITUATIONS.filter(([id]) => id || platoon.defender.some(g => g.situations.find(c => c.id === id)?.students.length));
        const rows = platoon.defender.map(g => `<tr><th scope="row">${esc(g.name)}</th>${shown.map(([id]) => {
            const c = g.situations.find(x => x.id === id), n = c ? c.students.length : 0;
            const on = selected && selected.group === g.id && selected.situation === id;
            return `<td>${n ? `<button class="secondary potential-count${on ? ' active' : ''}" data-group="${esc(g.id)}" data-situation="${esc(id)}" aria-pressed="${on}">${n}</button>` : '<span class="muted">0</span>'}</td>`;
        }).join('')}</tr>`).join('');
        const group = platoon.defender.find(g => g.id === selected?.group);
        const list = group?.situations.find(c => c.id === selected.situation);
        return `<h4>Образ защитника Отечества по взводу</h4><div class="table-scroll"><table class="potential-summary"><thead><tr><th>Группа качеств</th>${shown.map(([, name]) => `<th>${esc(name)}</th>`).join('')}</tr></thead><tbody>${rows}</tbody></table></div>
   ${list ? `<div class="potential-students"><h3>${esc(group.name)} — ${esc(list.text.toLowerCase())}: ${list.students.length}</h3><ul>${list.students.map(s => `<li><button class="text-button" data-session="${esc(s.sessionId)}">${esc(s.name)}</button></li>`).join('')}</ul></div>` : ''}`;
    }

    // The conclusion by a platoon: made by the button of three students at
    // least; above the table of levels. Returns the function that stops it.
    function mountPlatoon({element, api, esc, group, platoon, open}) {
        if (!group || !platoon) {
            element.innerHTML = '';
            return () => {};
        }
        const enough = platoon.students >= platoon.minimum;
        element.innerHTML = `<h3>Вывод по взводу ${esc(group)}</h3>${enough ? '<div data-platoon-text aria-live="polite"></div>' : `<p class="muted">Вывод по взводу составляется не меньше чем по ${platoon.minimum} студентам; завершили тестирование с методиками: ${platoon.students}.</p>`}<div data-platoon-groups></div>`;
        let selected = null;
        const groups = element.querySelector('[data-platoon-groups]');
        const drawGroups = () => {
            groups.innerHTML = platoon.students ? situations(platoon, esc, selected) : '';
            groups.querySelectorAll('[data-group]').forEach(b => b.onclick = () => {
                const next = {group: b.dataset.group, situation: b.dataset.situation};
                selected = selected && selected.group === next.group && selected.situation === next.situation ? null : next;
                drawGroups();
            });
            groups.querySelectorAll('[data-session]').forEach(b => b.onclick = () => open(b.dataset.session));
        };
        drawGroups();
        if (!enough) return () => {};
        const box = element.querySelector('[data-platoon-text]');
        let handle, alive = true;
        const run = (first = 'GET') => {
            handle?.stop();
            handle = window.portraitView.start(api, 'test-platoon', group, {
                alive: () => alive && box.isConnected,
                update: view => {
                    if (view.status === 'none') {
                        box.innerHTML = '<p><button type="button" data-platoon-make>Составить вывод по взводу</button></p><p class="muted">Вывод составляется по последнему завершённому тестированию с методиками каждого студента; в модель уходят только числа по взводу, без имён.</p>';
                    } else {
                        const ready = view.status === 'ready' ? window.portraitRenderers['test-platoon'].text(view.portrait.output, esc, platoon) : '';
                        box.innerHTML = window.portraitView.status(view, esc) + signature(view, esc, PLATOON_SIGNATURE) + ready;
                    }
                    box.querySelector('[data-platoon-make]')?.addEventListener('click', () => run('POST'));
                    box.querySelector('[data-portrait-retry]')?.addEventListener('click', () => run('POST'));
                }
            }, first);
        };
        run();
        return () => {
            alive = false;
            handle?.stop();
        };
    }

    return {mount, summary, students, mountPlatoon};
})();
