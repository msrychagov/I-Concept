// Only repeated tests belong here. The server checks comparability; platoon
// averages use the same students and the same complete readings at each date.
window.testDynamics = (() => {
    const valid = n => typeof n === 'number' && Number.isFinite(n);
    const fmt = n => Number(n).toLocaleString('ru-RU', {maximumFractionDigits: 2});
    const date = value => value && Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleDateString('ru-RU') : 'Без даты';
    const sectionKey = m => JSON.stringify([m.section, m.unit, m.min, m.max]);
    const colors = ['green', 'gold', 'blue'];

    function sections(slices) {
        const result = new Map();
        for (const slice of slices) for (const m of slice.metrics || []) {
            const key = sectionKey(m);
            if (!result.has(key)) result.set(key, {key, name: `${m.section} · ${m.unit} (${m.min}…${m.max})`, min: m.min, max: m.max, unit: m.unit});
        }
        return [...result.values()];
    }

    function paired(courses) {
        if (!courses.length) return {slices: [], people: 0};
        const maps = courses.map(c => new Map((c.students || []).filter(s => s.studentId).map(s => [s.studentId, s])));
        const people = [...maps[0].keys()].filter(id => maps.every(map => map.has(id)) && maps[0].get(id).compatibility && maps.every(map => map.get(id).compatibility === maps[0].get(id).compatibility));
        if (new Set(people.map(id => maps[0].get(id).compatibility)).size > 1) return {slices: [], people: 0, mixed: true};
        const definitions = new Map();
        for (const id of people) for (const m of maps[0].get(id).metrics || []) definitions.set(m.id, m);
        const metricMaps = maps.map(map => new Map(people.map(id => [id, new Map((map.get(id).metrics || []).map(m => [m.id, m]))])));
        const slices = courses.map(c => ({id: c.id, title: c.title, date: c.date, metrics: []}));
        for (const [id, definition] of definitions) {
            const complete = people.filter(person => metricMaps.every(map => {
                const m = map.get(person).get(id);
                return m && sectionKey(m) === sectionKey(definition) && valid(m.value);
            }));
            slices.forEach((slice, n) => slice.metrics.push({...definition, count: complete.length,
                value: complete.length ? complete.reduce((sum, person) => sum + metricMaps[n].get(person).get(id).value, 0) / complete.length : null}));
        }
        return {slices, people: people.length};
    }

    function render(slices, section, esc, platoon = false) {
        if (!section) return '<p class="muted">Нет числовых показателей для сравнения.</p>';
        const definitions = new Map();
        for (const slice of slices) for (const m of slice.metrics || []) if (sectionKey(m) === section.key) definitions.set(m.id, m);
        const maps = slices.map(s => new Map((s.metrics || []).map(m => [m.id, m])));
        const min = Math.min(0, section.min), max = section.max, range = max - min;
        if (!(range > 0)) return '';
        const zero = (0 - min) / range * 100;
        const legend = `<ul class="chart-legend">${slices.map((s, n) => `<li><i class="chart-color-${colors[n]}" aria-hidden="true"></i>${n + 1}. ${esc(date(s.date))} · ${esc(s.title)}</li>`).join('')}</ul>`;
        const groups = [...definitions.values()].map(m => {
            const values = maps.map(map => map.get(m.id)?.value);
            const delta = values.length > 1 && valid(values[values.length - 1]) && valid(values[values.length - 2]) ? values[values.length - 1] - values[values.length - 2] : null;
            return `<div class="dynamics-group"><div class="dynamics-bars" style="--zero:${zero}%"><span class="dynamics-ceiling" aria-hidden="true">${fmt(min)}…${fmt(max)}</span>${slices.map((s, n) => {
                const value = values[n], known = valid(value);
                const height = known ? Math.abs(value) / range * 100 : 0;
                const bottom = known && value < 0 ? zero - height : zero;
                return `<div class="dynamics-column"><div class="dynamics-stem" aria-hidden="true">${known ? `<span class="dynamics-fill chart-color-${colors[n]}" style="height:${height}%;bottom:${bottom}%"></span>` : '<span class="dynamics-absent">—</span>'}</div><span class="dynamics-value">${known ? fmt(value) : '—'}</span><span class="dynamics-index">${n + 1}</span><span class="dynamics-a11y">${esc(date(s.date))}: ${known ? esc(fmt(value) + ' ' + section.unit) : 'Нет данных'}</span></div>`;
            }).join('')}</div><div class="dynamics-name">${esc(m.name)}</div><p class="chart-counts">${delta == null ? 'Изменение не рассчитано' : `К предыдущему: ${delta > 0 ? '+' : ''}${fmt(delta)}`}${platoon ? ` · студентов: ${m.count || 0}` : ''}</p></div>`;
        }).join('');
        return `<figure class="diagnostic-chart"><figcaption><h4>${platoon ? 'Средние результаты одних и тех же студентов' : 'Результаты по датам'}</h4><p>${esc(section.unit)} · одинаковые вопросы и шкалы. Номера под столбцами соответствуют датам в легенде.${platoon ? ' Даты в легенде — даты назначения тестирований.' : ''}</p></figcaption>${legend}<div class="dynamics-grid">${groups}</div><p class="muted dynamics-note">${platoon ? 'Для каждого показателя учтены студенты с ответами во всех выбранных срезах; их число указано под группой. ' : ''}Показаны числовые изменения самоотчёта студента. Пропуски обозначены «—».</p></figure>`;
    }

    function defaultSelection(slices, platoon) {
        if (platoon) {
            for (let last = slices.length - 1; last > 0; last--) {
                for (let first = 0; first < last; first++) {
                    if (!paired([slices[first], slices[last]]).people) continue;
                    for (let middle = last - 1; middle > first; middle--) {
                        if (paired([slices[first], slices[middle], slices[last]]).people) return [slices[first].id, slices[middle].id, slices[last].id];
                    }
                    return [slices[first].id, slices[last].id];
                }
            }
        }
        return slices.length > 2 ? [slices[0].id, slices[slices.length - 2].id, slices[slices.length - 1].id] : slices.map(s => s.id);
    }

    function mount({element, history, esc, platoon = false}) {
        const slices = platoon ? history || [] : history?.slices || [];
        const title = platoon ? 'Динамика тестирования взвода' : 'Динамика тестирования';
        if (slices.length < 2) {
            element.innerHTML = `<h3>${title}</h3><p class="muted">${esc(platoon ? 'Выберите взвод, в котором есть хотя бы два завершённых назначения тестирования.' : history?.reason || 'Для сравнения нужны хотя бы два завершённых сопоставимых тестирования.')}</p>`;
            return;
        }
        let selected = new Set(defaultSelection(slices, platoon)), section = '';
        element.innerHTML = `<h3>${title}</h3><p class="muted">Выберите два или три среза. Автобиография и эссе показываются отдельно.</p><div class="dynamics-picks">${slices.map(s => `<label class="check"><input type="checkbox" data-dynamics-slice="${esc(s.id)}" ${selected.has(s.id) ? 'checked' : ''}>${esc(date(s.date))} · ${esc(s.title)}</label>`).join('')}</div><p data-dynamics-coverage class="muted" aria-live="polite"></p><label class="dynamics-section">Показатели<select data-dynamics-section></select></label><div data-dynamics-chart aria-live="polite"></div>${!platoon && history?.excluded ? `<p class="muted">Других завершённых прохождений с отличающимися вопросами или шкалами: ${history.excluded}. Они не включены в сравнение.</p>` : ''}`;
        const choice = element.querySelector('[data-dynamics-section]');
        const box = element.querySelector('[data-dynamics-chart]');
        const coverage = element.querySelector('[data-dynamics-coverage]');
        const draw = () => {
            const chosen = slices.filter(s => selected.has(s.id));
            const data = platoon ? paired(chosen) : {slices: chosen};
            const options = sections(data.slices);
            if (!options.some(o => o.key === section)) section = options[0]?.key || '';
            choice.innerHTML = options.map(o => `<option value="${esc(o.key)}">${esc(o.name)}</option>`).join('');
            choice.value = section;
            choice.disabled = !options.length || chosen.length < 2;
            coverage.textContent = platoon ? `Общих студентов с сопоставимыми прохождениями: ${data.people}.` : `Выбрано прохождений: ${chosen.length}.`;
            box.innerHTML = chosen.length < 2 ? '<p class="muted">Выберите минимум два среза.</p>' : data.mixed ? '<p class="muted">У участников разные наборы вопросов. Общее среднее не рассчитано; выберите назначения с одним и тем же составом вопросов.</p>' : platoon && !data.people ? '<p class="muted">В выбранных срезах нет общих студентов с одинаковыми вопросами и шкалами.</p>' : render(data.slices, options.find(o => o.key === section), esc, platoon);
            element.querySelectorAll('[data-dynamics-slice]').forEach(input => input.disabled = selected.size >= 3 && !input.checked);
        };
        element.querySelectorAll('[data-dynamics-slice]').forEach(input => input.onchange = () => {
            if (input.checked) selected.add(input.dataset.dynamicsSlice);
            else selected.delete(input.dataset.dynamicsSlice);
            draw();
        });
        choice.onchange = () => { section = choice.value; draw(); };
        draw();
    }

    return {mount, paired, sections, render, defaultSelection};
})();
