// The portrait by the test (task "test"): the levels of the methods and the
// kinds of activity are the program's, under them the text of the model by the
// sections of the first chapter. Mounted in the result of a session.
window.portraitRenderers = window.portraitRenderers || {};
window.portraitRenderers.test = {
    text(output, esc) {
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

window.testPortrait = (() => {
    const COMPONENTS = {CS: 'Ценностно-смысловой', EV: 'Эмоционально-волевой', DP: 'Деятельностно-практический'};
    const SIGNATURE = 'Составлено автоматически по самоотчёту студента и результатам методик';

    function levels(m, esc) {
        const methods = Object.keys(COMPONENTS).map(code => {
            const rows = (m.methods || []).filter(x => x.component === code);
            return rows.length ? `<tr class="test-portrait-group"><th colspan="3">${esc(COMPONENTS[code])} компонент</th></tr>${rows.map(x => `<tr><td>${esc(x.name)}<small>${esc(x.method)}</small></td><td>${x.score == null ? '—' : `${x.score} <small>из ${x.min}–${x.max}</small>`}</td><td><span class="potential-level level-${esc(x.level || 'none')}">${esc(x.levelText)}</span></td></tr>`).join('')}` : '';
        }).join('');
        const kinds = (m.kinds || []).map(k => `<tr><td>${esc(k.name)}</td><td></td><td><span class="potential-level level-${esc(k.level || 'none')}">${esc(k.levelText)}</span></td></tr>`).join('');
        // The statuses of professional identity have ranges, not levels: a
        // marked status is not better or worse, so the ranges are not coloured.
        const capital = text => text ? text[0].toUpperCase() + text.slice(1) : '';
        const identity = (m.identity || []).map(p => `<tr class="test-portrait-group"><th colspan="3">Профессиональная идентичность — ${esc(p.name.toLowerCase())}<small>${esc(p.method)}</small></th></tr>${p.statuses.map(x => `<tr><td>${esc(capital(x.name))}</td><td>${x.score == null ? '—' : `${x.score} <small>из 0–${x.max}</small>`}</td><td><span class="potential-level identity-range level-${x.range ? 'initial' : 'none'}">${esc(x.range || 'не определён')}</span></td></tr>`).join('')}<tr><td colspan="3">Ведущий статус: <b>${esc(p.leadingText)}</b></td></tr>`).join('');
        return `<div class="table-scroll"><table class="test-portrait-levels"><thead><tr><th>Показатель</th><th>Баллы</th><th>Уровень</th></tr></thead><tbody>${methods}<tr class="test-portrait-group"><th colspan="3">Виды деятельности (по анкете деятельностного компонента)</th></tr>${kinds}${identity}</tbody></table></div>
   <p class="muted">Уровни посчитаны программой по ключам методик из таблицы 8 руководителя${identity ? ', выраженность статусов профессиональной идентичности — по ключу методики А. А. Азбель, составленному на школьниках' : ''}; это самоотчёт студента, а не установленное качество личности.</p>`;
    }

    // Mounts the block and returns the function that stops its polling.
    function mount({element, api, esc, id, methods, completed}) {
        if (methods && !completed) {
            element.innerHTML = `<h3>Портрет по результатам тестирования</h3>${levels(methods, esc)}<p class="muted">Текст портрета составится после завершения тестирования.</p>`;
            return () => {};
        }
        if (!methods) {
            element.innerHTML = '<h3>Портрет по результатам тестирования</h3><p class="muted">В этом тестировании нет методик уровней Я-концепции, портрет не составляется. Включите «Методики уровней Я-концепции» при назначении.</p>';
            return () => {};
        }
        element.innerHTML = `<h3>Портрет по результатам тестирования</h3>${levels(methods, esc)}<div data-test-text aria-live="polite"></div>`;
        const box = element.querySelector('[data-test-text]');
        let handle, alive = true;
        const signature = view => {
            if (view.status !== 'ready') return '';
            const date = view.portrait.created ? new Date(view.portrait.created).toLocaleDateString('ru-RU') : '';
            return `<p class="portrait-signature muted">${esc(SIGNATURE)}${date ? ` · ${esc(date)}` : ''}</p>`;
        };
        const run = (first = 'GET') => {
            handle?.stop();
            handle = window.portraitView.start(api, 'test', id, {
                alive: () => alive && box.isConnected,
                update: view => {
                    const ready = view.status === 'ready' ? window.portraitRenderers.test.text(view.portrait.output, esc) : '';
                    box.innerHTML = window.portraitView.status(view, esc) + ready + signature(view);
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

    return {mount, summary, students};
})();
