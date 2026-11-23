// Charts display existing report fields. They do not introduce scores,
// thresholds or a shared scale for different diagnostic methods.
window.diagnosticCharts = (() => {
    const fmt = value => Number(value).toLocaleString('ru-RU', {maximumFractionDigits: 2});
    const valid = value => typeof value === 'number' && Number.isFinite(value);
    const clamp = value => Math.max(0, Math.min(100, value));
    const colors = {high: 'green', medium: 'gold', initial: 'blue', low: 'clay', none: 'gray', '': 'gray'};
    const tone = level => colors[level] || 'gray';
    const card = (title, note, body, esc) => `<figure class="diagnostic-chart"><figcaption><span class="chart-eyebrow">Диаграмма</span><h4>${esc(title)}</h4><p>${esc(note)}</p></figcaption>${body}</figure>`;
    const legend = (items, esc) => `<ul class="chart-legend">${items.map(i => `<li><i class="chart-color-${i.color}" aria-hidden="true"></i>${esc(i.label)}</li>`).join('')}</ul>`;
    const axis = (min, max, layout = '') => `<div class="chart-axis${layout ? ' chart-axis-' + layout : ''}" aria-hidden="true"><span>${fmt(min)}</span><span>${Number.isInteger((min + max) / 2) ? fmt((min + max) / 2) : ''}</span><span>${fmt(max)}</span></div>`;

    // One numerical range per row. Missing data stays missing; a measured zero
    // has a visible origin marker and a numerical label.
    function bars(rows, min, max, esc, series = [{label: '', color: 'green'}]) {
        if (!rows.length) return '<p class="muted">Данных для диаграммы пока нет.</p>';
        return `${axis(min, max, series.some(s => s.label) ? 'grouped' : '')}<ul class="chart-rows">${rows.map(row => `<li><div class="chart-row-label">${esc(row.name)}${row.note ? `<small>${esc(row.note)}</small>` : ''}</div><div class="chart-series">${row.values.map((value, n) => {
            const known = valid(value) && max > min;
            const width = known ? clamp((value - min) / (max - min) * 100) : 0;
            return `<div class="chart-bar-line">${series[n].label ? `<span class="chart-series-label">${esc(series[n].label)}</span>` : ''}<div class="chart-track${known ? '' : ' chart-missing'}" aria-hidden="true">${known ? `<span class="chart-fill chart-color-${series[n].color}" style="width:${width}%"></span>` : ''}</div><strong class="chart-number">${known ? fmt(value) : 'Нет данных'}</strong></div>`;
        }).join('')}</div></li>`).join('')}</ul>`;
    }

    // Absolute numbers, with the same denominator for every row. The buttons
    // reuse the tables' attributes and open the same lists of students.
    function distribution(rows, total, esc, selected, kind) {
        if (!total || !rows.length) return '';
        const categories = [...new Map(rows.flatMap(r => r.levels).map(l => [l.level, l])).values()];
        const key = row => kind === 'potential' ? row.id : row.kind + ':' + row.id;
        const body = `${legend(categories.map(l => ({label: l.levelText, color: tone(l.level)})), esc)}${axis(0, total, 'full')}<ul class="chart-rows chart-distribution">${rows.map(row => {
            const rowKey = key(row);
            const segments = row.levels.map(level => {
                const count = level.students.length;
                if (!count) return '';
                const on = selected?.level === level.level && (kind === 'potential' ? selected?.area : selected?.row) === rowKey;
                const attrs = kind === 'potential' ? `data-area="${esc(row.id)}"` : `data-row="${esc(rowKey)}"`;
                return `<button type="button" class="chart-segment chart-color-${tone(level.level)}${on ? ' active' : ''}" style="width:${clamp(count / total * 100)}%" ${attrs} data-level="${esc(level.level)}" aria-pressed="${on}" aria-label="${esc(row.title || row.name)}: ${esc(level.levelText)} — ${count} из ${total}" title="${esc(level.levelText)}: ${count}"><span aria-hidden="true">${count}</span></button>`;
            }).join('');
            const counts = row.levels.map(l => `${l.levelText}: ${l.students.length}`).join(' · ');
            return `<li><div class="chart-row-label">${esc(row.title || row.name)}${row.note ? `<small>${esc(row.note)}</small>` : ''}</div><div><div class="chart-stacked">${segments}</div><p class="chart-counts">${esc(counts)}</p></div></li>`;
        }).join('')}</ul>`;
        return card('Распределение по уровням', `Число студентов · всего ${total}. Нажмите на сегмент, чтобы открыть список.`, body, esc);
    }

    function potential(data, esc) {
        const categories = [{id: 'none', text: 'Не выражен'}, {id: 'initial', text: 'Начальный'}, {id: 'medium', text: 'Средний'}, {id: 'high', text: 'Высокий'}];
        const rows = (data.areas || []).filter(a => a.level !== 'none');
        if (!rows.length) return card('Потенциал по направлениям', 'По ответам автобиографии и действующей шкале потенциала.', '<p class="muted">Направлений с опытом пока нет.</p>', esc);
        const body = `<div class="chart-potential"><div class="chart-potential-head" aria-hidden="true"><span></span>${categories.map(c => `<span>${c.text}</span>`).join('')}</div>${rows.map(a => `<div class="chart-potential-row"><div class="chart-row-label">${esc(a.title)}<small>${esc(a.levelText)}</small></div>${categories.map(c => `<span class="chart-step${a.level === c.id ? ' chart-step-active chart-color-' + tone(c.id) : ''}" aria-hidden="true">${a.level === c.id ? '●' : '·'}</span>`).join('')}</div>`).join('')}</div>`;
        return card('Потенциал по направлениям', 'Показаны направления с опытом. Категории отражают утверждённые уровни, а не числовой балл способностей.', body, esc);
    }

    function climate(pairs, esc, average = false) {
        if (!pairs?.length) return '';
        const known = pairs.some(p => valid(average ? p.mean : p.score));
        const body = `${axis(-3, 3)}<ul class="chart-rows">${pairs.map(p => {
            const value = average ? p.mean : p.score;
            const hasValue = valid(value);
            const left = hasValue ? 50 + Math.min(0, value) / 6 * 100 : 50;
            const width = hasValue ? Math.abs(value) / 6 * 100 : 0;
            return `<li><div class="chart-row-label">${esc(p.side)}</div><div class="chart-bar-line"><div class="chart-track chart-diverging${hasValue ? '' : ' chart-missing'}" aria-hidden="true">${hasValue ? `<span class="chart-fill chart-color-${value < 0 ? 'clay' : 'green'}" style="left:${clamp(left)}%;width:${clamp(width)}%"></span>` : ''}</div><strong class="chart-number">${hasValue ? (value > 0 ? '+' : '') + fmt(value) : 'Нет данных'}</strong></div></li>`;
        }).join('')}</ul>`;
        return card(average ? 'Климат взвода — средние оценки' : 'Климат взвода — ответы студента', `Шкала от −3 до +3. Названа положительная сторона каждой пары. Уровни климата не определяются.${known ? '' : ' Ответов пока нет.'}`, body, esc);
    }

    function test(methods, esc) {
        if (!methods) return '<p class="muted">В этом тестировании нет данных методик для диаграмм.</p>';
        const images = (methods.defender || []).map(g => ({name: g.name, values: [g.real, g.ideal]}));
        const imageSeries = [{label: 'Я-реальное', color: 'green'}, {label: 'Я-идеальное', color: 'gold'}];
        const image = images.length ? card('Я-реальное и Я-идеальное', 'Самооценка по группам качеств образа защитника Отечества. Шкала 0–10; 10 — верх шкалы.', `${legend(imageSeries, esc)}${bars(images, 0, 10, esc, imageSeries)}`, esc) : '';
        const scales = (methods.methods || []).filter(m => m.max > m.min).map(m => `<div class="chart-method"><h5>${esc(m.name)}</h5><p class="chart-method-note">${esc(m.method)} · шкала ${fmt(m.min)}–${fmt(m.max)} · ${esc(m.levelText)}</p>${bars([{name: 'Баллы', values: [m.score]}], m.min, m.max, esc)}</div>`).join('');
        const scores = scales ? card('Баллы по методикам', 'У каждой методики своя шкала. Положение столбца показывает балл в пределах этой шкалы; пропуски не заменяются нулём.', scales, esc) : '';
        return `<div class="diagnostic-charts">${image}${scores}${climate(methods.climate?.pairs, esc)}</div>`;
    }

    function evidence(rows, esc) {
        if (!rows?.length) return '';
        return card('Показатели по утверждениям', 'Средний балл по шкале 1–5 с учётом обратных утверждений. Самоотчёт студента; при неполном наборе ответов балл не рассчитывается.', bars(rows.map(r => ({name: r.name, note: valid(r.coverage) && valid(r.total) ? `Ответов ${r.coverage} из ${r.total}` : '', values: [r.statements]})), 1, 5, esc), esc);
    }

    function values(count, esc) {
        const ideas = count.ideas || [];
        const series = [{key: 'accept', label: 'Принимает', color: 'green'}, {key: 'reject', label: 'Отвергает', color: 'clay'}, {key: 'everydayAccept', label: 'Принимает в повседневной жизни', color: 'blue'}, {key: 'everydayReject', label: 'Отвергает в повседневной жизни', color: 'gold'}];
        const total = i => series.reduce((sum, s) => sum + (i[s.key] || 0), 0);
        const max = Math.max(0, ...ideas.map(total));
        const body = max ? `${legend(series, esc)}${axis(0, max, 'full')}<ul class="chart-rows">${ideas.map(i => `<li><div class="chart-row-label">${esc(i.label)}</div><div><div class="chart-stacked" aria-hidden="true">${series.map(s => i[s.key] ? `<span class="chart-value-segment chart-color-${s.color}" style="width:${clamp(i[s.key] / max * 100)}%">${i[s.key]}</span>` : '').join('')}</div><p class="chart-counts">${series.map(s => `${s.label.toLowerCase()}: ${i[s.key] || 0}`).map(esc).join(' · ')}</p></div></li>`).join('')}</ul>` : '<p class="muted">В собственных словах автора ценностей не отмечено. Это не означает отсутствия ценностей у человека.</p>';
        return card('Упоминания ценностей по идеям', 'Число упоминаний по разметке текста. Одна фраза может относиться к нескольким ценностям. Антиценности и чужие слова без отношения автора показаны отдельно ниже. Это не оценка личности.', body, esc);
    }

    function valueAuthors(data, esc) {
        if (!data?.marked) return '';
        return card('Ценности по авторам', `Число авторов, принимающих ценности каждой идеи, из ${data.marked} с готовой разметкой. Один автор может входить в несколько строк; строки не складываются. Упоминания только в повседневной жизни учитываются отдельно ниже.`, bars((data.ideas || []).map(i => ({name: i.label, values: [i.authors]})), 0, data.marked, esc), esc);
    }

    function essayProgress(counts, esc) {
        const total = counts.submitted + counts.notSubmitted;
        if (!total) return '';
        const rows = [{name: 'Сдано, разбор готов', values: [counts.analysed]}, {name: 'Сдано, без готового разбора', values: [counts.submitted - counts.analysed]}, {name: 'Ещё не сдано', values: [counts.notSubmitted]}];
        return card('Готовность эссе', `Количество эссе · назначено ${total}. Статус разбора показывает готовность обработки текста.`, bars(rows, 0, total, esc), esc);
    }

    function essayFeedback(title, items, groups, esc, sourceList) {
        const rows = (items || []).map((item, n) => ({item, group: groups?.[n]}));
        const total = Math.max(0, ...rows.map(r => r.group?.of || 0));
        rows.sort((a, b) => (b.group?.authors || 0) - (a.group?.authors || 0));
        const body = rows.length ? `<div class="feedback-rows">${rows.map(({item, group}) => {
            const known = valid(group?.authors) && total > 0;
            const width = known ? clamp(group.authors / total * 100) : 0;
            const label = known ? `${group.authors} из ${total} эссе` : 'Нет данных об источниках';
            return `<details class="feedback-row"><summary><span class="feedback-label">${esc(item.text)}</span><span class="feedback-measure"><span class="chart-track${known ? '' : ' chart-missing'}" aria-hidden="true">${known ? `<span class="chart-fill chart-color-green" style="width:${width}%"></span>` : ''}</span><strong>${esc(label)}</strong><small>Исходные эссе</small></span></summary><div class="essay-why-body">${group ? sourceList(group) : '<p class="muted">Ссылок на исходные эссе пока нет.</p>'}</div></details>`;
        }).join('')}</div>` : '<p class="muted">Общих пунктов в сводке нет.</p>';
        return card(title, 'Число разных эссе, на которых основан каждый пункт сводки. Один автор может входить в несколько пунктов. Нажмите на полосу, чтобы открыть источники.', body, esc);
    }

    return {potential, distribution, test, evidence, climate, values, valueAuthors, essayProgress, essayFeedback};
})();
