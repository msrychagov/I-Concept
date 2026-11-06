// The portrait by the test (task "test") and the conclusion by a platoon (task
// "test-platoon"). Above the text — the program's signature that it is the
// student's self-report; under it — the tables with the program's numbers.
// The portrait of v5 is four paragraphs with no headings, the recommendations
// and the conclusion; that of v2–v4 is the conclusion of a psychologist in
// seven sections; a portrait of v1 is still shown in its own format.
window.portraitRenderers = window.portraitRenderers || {};
window.portraitRenderers.test = {
    text(output, esc, data, version) {
        if (output.paragraphs) return v5(output, esc, data || {}, versionNumber(version) >= 7);
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

// versionNumber is the number of a version of an instruction: «v7» is 7.
function versionNumber(version) {
    return parseInt(String(version || '').replace(/^v/, ''), 10) || 0;
}

// recommendationList names the direction of the catalogue and the group of the
// image of every recommendation; an action of test/v6 or test-platoon/v4 also
// says who does it, in which activity, and what the program suggests to look
// at by the next testing: how the student rates himself in the group and its
// scales of the methods on the low level. An action of test/v7 or
// test-platoon/v5 (human) is for everyone: no who and no activity.
function testRecommendations(items, esc, data, human) {
    const lowerFirst = t => t ? t[0].toLowerCase() + t.slice(1) : t;
    const directions = Object.fromEntries((data.catalogue || []).map(d => [d.id, d.name]));
    const groups = Object.fromEntries((data.defender || []).map((g, n) => [n + 1, g]));
    const look = r => {
        const low = ((groups[r.group] || {}).methods || []).filter(m => m.level === 'low').map(m => lowerFirst(m.name));
        return ['как студент оценивает себя в этих качествах', ...low].join('; ');
    };
    return (items || []).length ? `<h4>Рекомендации</h4><ul class="test-portrait-recommendations">${items.map(r => `<li><b>${esc(directions[r.code] || r.code)}</b>${groups[r.group] ? ` <span class="muted">(${esc(groups[r.group].name.toLowerCase())})</span>` : ''}. ${esc(r.text)}${testAction(r, esc, look(r), human)}</li>`).join('')}</ul>` : '';
}

// testAction is the line under an action: who does it and in which activity
// (test/v6, test-platoon/v4) and what to look at by the next testing.
function testAction(r, esc, look, human) {
    if (!r.who && !human) return '';
    const parts = [r.who ? `Кто: ${r.who}` : '', r.kind ? `${r.kind} деятельность` : '', look ? `при следующем тестировании посмотреть: ${look}` : ''].filter(Boolean);
    return `<small class="test-portrait-action muted">${esc(parts.join(' · '))}</small>`;
}

// testWork is «Кому нужна отдельная работа» of test-platoon/v4: the groups the
// program put together, by the label of the answer, with the names, the bases,
// the action of the model and what to look at; the groups past the first five
// have no action and go under «Ещё нужна отдельная работа».
function testWork(items, esc, data, human) {
    const lowerFirst = t => t ? t[0].toLowerCase() + t.slice(1) : t;
    const groups = data.workGroups || [];
    const byLabel = Object.fromEntries(groups.map(g => [g.label, g]));
    const directions = Object.fromEntries((data.catalogue || []).map(d => [d.id, d.name]));
    const names = g => g.students.map(s => `<button type="button" class="text-button" data-session="${esc(s.sessionId)}">${esc(s.name)}</button>`).join(', ');
    // The bases in plain words (Инструкция по взводу v5.md, Т5) for every version.
    const bases = g => `<span class="muted">По тесту:</span> ${esc(((g.plain || []).length ? g.plain : g.bases).join('; '))}.`;
    const shown = (items || []).map(r => {
        const g = byLabel[r.label];
        const head = g ? `<b>${names(g)}</b><br>${bases(g)}<br>` : '';
        const direction = directions[r.code] ? ` <span class="muted">(${esc(lowerFirst(directions[r.code]))})</span>` : '';
        return `<li>${head}<span class="muted">Что сделать:</span> ${esc(r.text)}${direction}${testAction(r, esc, g ? g.look.map(lowerFirst).join('; ') : '', human)}</li>`;
    }).join('');
    const rest = groups.filter(g => !g.model).map(g => `<li><b>${names(g)}</b><br>${bases(g)}</li>`).join('');
    return (shown ? `<h4>Кому нужна отдельная работа</h4><ul class="test-portrait-work">${shown}</ul>` : '')
        + (rest ? `<h4>Ещё нужна отдельная работа</h4><ul class="test-portrait-work">${rest}</ul>` : '');
}

// paragraphs are the four paragraphs of the portrait of v5 or of the
// conclusion by a platoon of v3, in their order, with no headings.
function testParagraphs(output, esc) {
    const p = output.paragraphs || {};
    return ['values', 'will', 'image', 'environment'].filter(key => p[key]).map(key => `<p>${esc(p[key])}</p>`).join('');
}

function v5(output, esc, data, human) {
    return `<div class="test-portrait-text">
   ${testParagraphs(output, esc)}
   ${testRecommendations(output.recommendations, esc, data, human)}
   ${output.conclusion ? `<h4>Общий вывод</h4><p>${esc(output.conclusion)}</p>` : ''}
  </div>`;
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
    text(output, esc, data, version) {
        const paragraph = (title, text) => text ? `<h4>${esc(title)}</h4><p>${esc(text)}</p>` : '';
        const human = versionNumber(version) >= 5;
        const work = output.individual || [];
        const individual = work.length && typeof work[0] === 'object' ? testWork(work, esc, data || {}, human)
            : work.length ? `<h4>Кому нужна отдельная работа</h4><ul>${work.map(x => `<li>${esc(x)}</li>`).join('')}</ul>` : '';
        if (output.paragraphs) {
            return `<div class="test-portrait-text">
   ${testParagraphs(output, esc)}
   ${testRecommendations(output.recommendations, esc, data || {}, human)}
   ${individual}
   ${paragraph('Общий вывод', output.conclusion)}
  </div>`;
        }
        return `<div class="test-portrait-text">
   ${paragraph('Состояние взвода', output.overview)}
   ${paragraph('Компоненты Я-концепции', output.components)}
   ${paragraph('Профессиональная Я-концепция', output.identity)}
   ${paragraph('Соответствие образу защитника Отечества', output.ideal)}
   ${paragraph('Деятельность и условия среды', output.activity)}
   ${testRecommendations(output.recommendations, esc, data || {})}
   ${individual}
   ${paragraph('Общий вывод', output.conclusion)}
  </div>`;
    }
};

window.testPortrait = (() => {
    const COMPONENTS = {CS: 'Ценностно-смысловой', EV: 'Эмоционально-волевой', DP: 'Деятельностно-практический'};
    const SIGNATURE = 'Составлено программой по ответам тестирования.';
    const PLATOON_SIGNATURE = 'Составлено программой по ответам тестирования студентов взвода.';
    const number = v => v == null ? '—' : String(v).replace('.', ',');
    // A rating of the climate, from +3 to −3, with its sign.
    const signed = v => v == null ? '—' : v > 0 ? '+' + number(v) : v < 0 ? '−' + number(-v) : '0';
    const capital = text => text ? text[0].toUpperCase() + text.slice(1) : '';

    function levels(m, esc) {
        const methods = Object.keys(COMPONENTS).map(code => {
            const rows = (m.methods || []).filter(x => x.component === code);
            // The level of the component by its scales, counted by the program:
            // since v5 the text does not name the components.
            const level = (m.componentLevels || {})[code];
            return rows.length ? `<tr class="test-portrait-group"><th colspan="3">${esc(COMPONENTS[code])} компонент${level ? ` — ${esc(level)}` : ''}</th></tr>${rows.map(x => `<tr><td>${esc(x.name)}<small>${esc(x.method)}</small></td><td>${x.score == null ? '—' : `${x.score} <small>из ${x.min}–${x.max}</small>`}</td><td><span class="potential-level level-${esc(x.level || 'none')}">${esc(x.levelText)}</span></td></tr>`).join('')}` : '';
        }).join('');
        const kinds = (m.kinds || []).map(k => `<tr><td>${esc(k.name)}</td><td></td><td><span class="potential-level level-${esc(k.level || 'none')}">${esc(k.levelText)}</span></td></tr>`).join('');
        // The statuses of professional identity have ranges, not levels: a
        // marked status is not better or worse, so the ranges are not coloured.
        const identity = (m.identity || []).map(p => `<tr class="test-portrait-group"><th colspan="3">Профессиональная идентичность — ${esc(p.name.toLowerCase())}<small>${esc(p.method)}</small></th></tr>${p.statuses.map(x => `<tr><td>${esc(capital(x.name))}</td><td>${x.score == null ? '—' : `${x.score} <small>из 0–${x.max}</small>`}</td><td><span class="potential-level identity-range level-${x.range ? 'initial' : 'none'}">${esc(x.range || 'не определён')}</span></td></tr>`).join('')}<tr><td colspan="3">Ведущий статус: <b>${esc(p.leadingText)}</b></td></tr>`).join('');
        const c = m.climate;
        const climate = c ? `<tr class="test-portrait-group"><th colspan="3">Климат взвода — по оценке самого студента<small>${esc(c.method)}</small></th></tr><tr><td>Сумма по ${c.total} парам</td><td>${c.sum == null ? `— <small>отвечено ${c.answered} из ${c.total}</small>` : `${signed(c.sum)} <small class="climate-range">из ${signed(c.min)}…${signed(c.max)}</small>`}</td><td><span class="muted">уровней нет</span></td></tr>${c.sum == null ? '' : `<tr><td colspan="3">${c.equal ? 'Все стороны климата студент оценил одинаково.' : `Сильнее всего: ${esc(c.strongest.join('; '))}.<br>Слабее всего: ${esc(c.weakest.join('; '))}.`}</td></tr>`}` : '';
        return `<div class="table-scroll"><table class="test-portrait-levels"><thead><tr><th>Показатель</th><th>Баллы</th><th>Уровень</th></tr></thead><tbody>${methods}<tr class="test-portrait-group"><th colspan="3">Виды деятельности (по анкете деятельностного компонента)</th></tr>${kinds}${identity}${climate}</tbody></table></div>
   ${c ? `<details class="climate-pairs"><summary>Оценки климата взвода по парам</summary><div class="table-scroll"><table class="test-portrait-levels"><thead><tr><th>Сторона климата</th><th>Оценка</th></tr></thead><tbody>${c.pairs.map(p => `<tr><td>${esc(capital(p.side))}<small>${esc(p.answer || 'нет ответа')}</small></td><td>${signed(p.score)}</td></tr>`).join('')}</tbody></table></div></details>` : ''}
   <p class="muted">Уровни посчитаны программой по ключам методик из таблицы 8 руководителя, уровень компонента — по уровням его шкал${identity ? ', выраженность статусов профессиональной идентичности — по ключу методики А. А. Азбель, составленному на школьниках' : ''}; это самоотчёт студента, а не установленное качество личности.${c ? ' Климат взвода — оценки от +3 до −3 по парам противоположных утверждений; уровней климата нет: единого ключа методики не принято.' : ''}</p>`;
    }

    // The groups of the image of the defender: the real and the ideal self of
    // the student (0–10), the distance to the image and the situation the
    // program found, and the congruence of the images.
    function image(m, esc) {
        if (!(m.defender || []).length) return '';
        const others = g => (g.norm == null ? '' : ` · чего ждут другие ${number(g.norm)}`) + (g.mirror == null ? '' : ` · как видят другие ${number(g.mirror)}`);
        const rows = m.defender.map(g => `<tr><td>${esc(g.name)}<small>Я-реальное ${number(g.real)} · Я-идеальное ${number(g.ideal)} · до образа ${number(g.distance)}${esc(others(g))}</small></td><td>${esc(g.situationText)}${danilova(g, esc)}${g.note ? `<small>${esc(g.note)}</small>` : ''}</td></tr>`).join('');
        const c = m.congruence;
        const marked = (m.components || []).filter(x => x.pressure === 'yes' || x.adequacy === 'better' || x.adequacy === 'worse');
        const counted = (m.components || []).some(x => x.pressure || x.adequacy) || m.defender.some(g => g.pressure || g.adequacy);
        const components = marked.length ? `<p>По компонентам Я-концепции:</p><ul class="danilova-components">${marked.map(x => `<li>${esc(capital(x.name))}: ${esc([x.pressure === 'yes' ? 'внешнее давление' : '', x.adequacy === 'better' || x.adequacy === 'worse' ? 'студент ' + x.adequacyText : ''].filter(Boolean).join('; '))}</li>`).join('')}</ul>` : '';
        return `<h4>Образ защитника Отечества</h4><div class="table-scroll"><table class="test-portrait-levels test-portrait-image"><thead><tr><th>Группа качеств</th><th>Ситуация</th></tr></thead><tbody>${rows}</tbody></table></div>
   ${c ? `<p>Согласованность Я-реального и Я-идеального: ${esc(c.text.replace(/\.$/, ''))}${c.value == null ? '' : ` <span class="muted">(коэффициент ${number(c.value)})</span>`}.</p>` : ''}${components}
   <p class="muted">Образ — верх шкалы 0–10; ситуация — по расстоянию от Я-идеального до образа и от Я-реального до Я-идеального.${counted ? ' Внешнее давление и взгляд других — по правилам О. Ю. Даниловой: «чего ждут другие» или «как видят другие» отличается от Я-реального на 3 балла и больше; это мнение самого студента, а не оценка товарищей.' : ''}</p>`;
    }

    // The rules of Danilova in a group, only when the program marked them.
    function danilova(g, esc) {
        const lines = [];
        if (g.pressure === 'yes') lines.push('Внешнее давление: ' + g.pressureText);
        if (g.adequacy === 'better' || g.adequacy === 'worse') lines.push('Самооценка: студент ' + g.adequacyText);
        return lines.map(line => `<small>${esc(line)}</small>`).join('');
    }

    function tables(m, esc) {
        return `<div class="test-portrait-numbers"><h4>Результаты методик</h4>${levels(m, esc)}${image(m, esc)}</div>`;
    }

    // A switch of the panes «Вывод», «Подробно», «Рекомендации», like the
    // window of the autobiography.
    function tabs(names) {
        const buttons = names.map(([id, title], n) => `<button type="button" data-show="${id}" aria-pressed="${n === 0}"${n ? ' class="secondary"' : ''}>${title}</button>`).join('');
        return `<div class="toolbar autobiography-switch" role="group" aria-label="Что показать">${buttons}</div>`;
    }

    function bindTabs(element) {
        element.querySelectorAll('[data-show]').forEach(b => b.onclick = () => {
            element.querySelectorAll('[data-show]').forEach(x => {
                const on = x === b;
                x.classList.toggle('secondary', !on);
                x.setAttribute('aria-pressed', String(on));
            });
            element.querySelectorAll('[data-pane]').forEach(pane => pane.hidden = pane.dataset.pane !== b.dataset.show);
        });
    }

    // Mounts the conclusion on a test the program wrote (test_conclusion.go):
    // the summary, the details with the tables of the methods and the
    // recommendations. Returns the function that stops it: nothing to stop,
    // the conclusion comes ready in the report.
    function mount({element, esc, methods, completed, conclusion}) {
        const title = '<h3>Вывод по тестированию</h3>';
        if (!methods) {
            element.innerHTML = `${title}<p class="muted">В этом тестировании нет методик уровней Я-концепции, вывод не составляется. Включите «Методики уровней Я-концепции» при назначении.</p>`;
            return () => {};
        }
        if (!completed || !conclusion) {
            element.innerHTML = `${title}<p class="muted">Вывод составится после завершения тестирования.</p>${tables(methods, esc)}`;
            return () => {};
        }
        const advice = (conclusion.recommendations || []).map(r => `<li>${esc(r.text)}</li>`).join('');
        element.innerHTML = `${title}${tabs([['summary', 'Вывод'], ['details', 'Подробно'], ['advice', 'Рекомендации']])}
   <div data-pane="summary" class="test-portrait-text"><p>${esc(conclusion.summary)}</p></div>
   <div data-pane="details" class="test-portrait-text" hidden><p class="muted">По ответам студента.</p>${(conclusion.details || []).map(p => `<h4>${esc(p.title)}</h4><p>${esc(p.text)}</p>`).join('')}${tables(methods, esc)}</div>
   <div data-pane="advice" class="test-portrait-text" hidden>${advice ? `<ul class="test-portrait-recommendations">${advice}</ul>` : '<p class="muted">Рекомендаций нет.</p>'}</div>
   <p class="portrait-signature muted">${esc(SIGNATURE)}</p>`;
        bindTabs(element);
        return () => {};
    }

    // The platoon table "scale or kind × level" with counts; a count opens the
    // list of students, a name opens the result.
    // titled: with its own heading and a line about what it is — inside
    // «Подробно» of the conclusion by a platoon.
    function summary(data, esc, selected, titled) {
        if (!data.sessions) return '<p class="muted">Завершённых тестирований с методиками пока нет.</p>';
        const head = `<tr><th>Показатель</th>${data.rows[0].levels.map(l => `<th>${esc(l.levelText)}</th>`).join('')}</tr>`;
        const group = (title, rows) => rows.length ? `<tr class="test-portrait-group"><th colspan="${rows[0].levels.length + 1}">${esc(title)}</th></tr>${rows.map(r => `<tr><th scope="row">${esc(r.name)}</th>${r.levels.map(l => {
            const on = selected && selected.row === r.kind + ':' + r.id && selected.level === l.level;
            return `<td>${l.students.length ? `<button class="secondary potential-count${on ? ' active' : ''}" data-row="${esc(r.kind + ':' + r.id)}" data-level="${esc(l.level)}" aria-pressed="${on}">${l.students.length}</button>` : '<span class="muted">0</span>'}</td>`;
        }).join('')}</tr>`).join('')}` : '';
        const body = Object.keys(COMPONENTS).map(code => group(COMPONENTS[code] + ' компонент', data.rows.filter(r => r.kind === 'method' && r.component === code))).join('')
            + group('Виды деятельности', data.rows.filter(r => r.kind === 'activity'));
        const head2 = titled ? '<h4>Уровни методик по взводу</h4><p>Сколько студентов взвода оказались на высоком, среднем и низком уровне по каждой методике теста. Нажмите на число — откроется список этих студентов.</p>' : '';
        return `${head2}<p class="muted">Учтено тестирований: ${data.sessions} — последнее завершённое с методиками у каждого студента. Уровни посчитаны программой по ключам методик.</p><div class="table-scroll"><table class="potential-summary"><thead>${head}</thead><tbody>${body}</tbody></table></div>`;
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

    // The climate of the platoon by the students who answered all the pairs:
    // the mean sum and the mean of every pair, the sides the program found
    // strongest and weakest; no level.
    function climate(platoon, esc) {
        const c = platoon.climate;
        if (!c) return '';
        // The marks are not coloured: the sides are not levels.
        const mark = side => c.strongest.includes(side) ? ' <span class="muted">· сильнее всего</span>' : c.weakest.includes(side) ? ' <span class="muted">· слабее всего</span>' : '';
        return `<h4>Климат взвода</h4><p class="muted">${esc(c.method)}. Оценили все пары: ${c.rated.length} — ${esc(c.rated.map(s => s.name).join(', '))}. Средняя сумма: <b>${signed(c.sum)}</b> из ${signed(c.min)}…${signed(c.max)}. Уровней климата нет.</p>
   <div class="table-scroll"><table class="potential-summary climate-summary"><thead><tr><th>Сторона климата</th><th>Средняя оценка</th></tr></thead><tbody>${c.pairs.map(p => `<tr><th scope="row">${esc(capital(p.side))}${mark(p.side)}</th><td>${signed(p.mean)}</td></tr>`).join('')}</tbody></table></div>
   ${c.equal ? '<p class="muted">Все стороны климата оценены одинаково.</p>' : ''}`;
    }

    // The conclusion by a platoon the program wrote (test_platoon_conclusion.go):
    // the summary, the recommendations to the platoon and the groups for
    // separate work with their action; below — the groups of the image and the
    // climate. Returns the function that stops it: nothing to stop.
    function mountPlatoon({element, esc, group, platoon, open}) {
        if (!group || !platoon) {
            element.innerHTML = '';
            return () => {};
        }
        const c = platoon.conclusion;
        if (!c) {
            element.innerHTML = `<h3>Вывод по взводу ${esc(group)}</h3><p class="muted">Вывод по взводу составляется, когда тестирование с методиками завершили хотя бы двое; сейчас: ${platoon.students}.</p><div data-platoon-groups></div>`;
        } else {
            // The same switch as the conclusion on a student: «Вывод»,
            // «Рекомендации», «Отдельная работа», «Подробно» (the groups of the
            // image and the climate of the platoon).
            const advice = c.recommendations.length ? `<ul class="test-portrait-recommendations">${c.recommendations.map(r => `<li>${esc(r.text)}</li>`).join('')}</ul>` : '<p class="muted">Рекомендаций нет.</p>';
            const work = platoonWork(platoon, c, esc) || '<p class="muted">Отдельная работа сейчас никому не нужна.</p>';
            element.innerHTML = `<h3>Вывод по взводу ${esc(group)}</h3>${tabs([['summary', 'Вывод'], ['advice', 'Рекомендации'], ['work', 'Отдельная работа'], ['details', 'Подробно']])}
   <div data-pane="summary" class="test-portrait-text"><p>${esc(c.summary)}</p></div>
   <div data-pane="advice" class="test-portrait-text" hidden>${advice}</div>
   <div data-pane="work" class="test-portrait-text" hidden><p class="muted">Кому нужна отдельная работа и что сделать.</p>${work}</div>
   <div data-pane="details" hidden><div data-platoon-groups></div></div>
   <p class="portrait-signature muted">${esc(PLATOON_SIGNATURE)}</p>`;
            bindTabs(element);
        }
        element.querySelectorAll('[data-pane="work"] [data-session]').forEach(b => b.onclick = () => open(b.dataset.session));
        let selected = null;
        const groups = element.querySelector('[data-platoon-groups]');
        const drawGroups = () => {
            groups.innerHTML = platoon.students ? situations(platoon, esc, selected) + climate(platoon, esc) : '';
            groups.querySelectorAll('[data-group]').forEach(b => b.onclick = () => {
                const next = {group: b.dataset.group, situation: b.dataset.situation};
                selected = selected && selected.group === next.group && selected.situation === next.situation ? null : next;
                drawGroups();
            });
            groups.querySelectorAll('[data-session]').forEach(b => b.onclick = () => open(b.dataset.session));
        };
        drawGroups();
        return () => {};
    }

    // «Кому нужна отдельная работа»: the groups of the program with the names,
    // the bases in plain words and the action of the phrase book.
    function platoonWork(platoon, c, esc) {
        const actions = Object.fromEntries((c.work || []).map(w => [w.label, w.text]));
        const items = (platoon.workGroups || []).map(g => {
            const names = g.students.map(s => `<button type="button" class="text-button" data-session="${esc(s.sessionId)}">${esc(s.name)}</button>`).join(', ');
            const bases = ((g.plain || []).length ? g.plain : g.bases).join('; ');
            return `<li><b>${names}</b><br><span class="muted">По тесту:</span> ${esc(bases)}.${actions[g.label] ? `<br><span class="muted">Что сделать:</span> ${esc(actions[g.label])}` : ''}</li>`;
        }).join('');
        return items ? `<ul class="test-portrait-work">${items}</ul>` : '';
    }

    return {mount, summary, students, mountPlatoon};
})();
