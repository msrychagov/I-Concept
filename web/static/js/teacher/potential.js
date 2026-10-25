// Potential by the autobiography: levels of one student and the platoon summary.
// Levels come ready from the server; here they are only drawn.
window.potentialView = (() => {
    const NOTE = 'Уровни посчитаны программой по шкале потенциала из того, что студент сообщил о себе сам. Это не проверенная оценка способностей.';
    const lower = text => text ? text[0].toLowerCase() + text.slice(1) : '';
    const badge = (level, text, esc) => `<span class="potential-level level-${esc(level)}">${esc(text)}</span>`;

    // Levels of one submitted autobiography, for the answers dialog. The levels
    // are the program's; with a ready text of the model (output) the table also
    // shows what the student reported and the conclusion for each area.
    function card(potential, esc, output) {
        const texts = {};
        for (const item of output?.potential || []) texts[item.area] = item;
        const extra = !!output;
        const rows = potential.areas.map(a => {
            const text = texts[lower(a.title)];
            return `<tr><th scope="row">${esc(a.title)}</th><td>${badge(a.level, a.levelText, esc)}</td><td>${a.reasons.length ? `<ul class="potential-reasons">${a.reasons.map(r => `<li>${esc(r)}</li>`).join('')}</ul>` : '<span class="muted">Сведений нет</span>'}</td>${extra ? `<td>${text ? esc(text.facts) : ''}</td><td>${text ? esc(text.conclusion) : ''}</td>` : ''}</tr>`;
        }).join('');
        const high = potential.high.length ? `<p><b>Высокий уровень:</b> ${esc(potential.high.map(lower).join(', '))}.</p>` : '<p class="muted">Направлений с высоким уровнем нет.</p>';
        const awards = potential.awards.length ? `<p><b>Государственные награды:</b> ${esc(potential.awards.map(a => a.year ? `${a.name} (${a.year})` : a.name).join('; '))}. На уровни не влияют.</p>` : '';
        return `<section class="potential-card"><h3>Потенциал</h3><p class="muted">${esc(NOTE)}</p>${high}${awards}<div class="table-scroll"><table class="potential-table"><thead><tr><th>Направление</th><th>Уровень</th><th>Из каких ответов</th>${extra ? '<th>Что указал студент</th><th>Вывод</th>' : ''}</tr></thead><tbody>${rows}</tbody></table></div></section>`;
    }

    // Levels of one submitted autobiography beside the conclusion of the
    // instruction v4 (decision Б2, 25 October 2026): only the areas the student
    // has experience in; the six areas are under «Все направления».
    function brief(potential, esc) {
        const table = areas => `<div class="table-scroll"><table class="potential-table"><thead><tr><th>Направление</th><th>Уровень</th><th>Из каких ответов</th></tr></thead><tbody>${areas.map(a => `<tr><th scope="row">${esc(a.title)}</th><td>${badge(a.level, a.levelText, esc)}</td><td>${a.reasons.length ? `<ul class="potential-reasons">${a.reasons.map(r => `<li>${esc(r)}</li>`).join('')}</ul>` : '<span class="muted">Сведений нет</span>'}</td></tr>`).join('')}</tbody></table></div>`;
        const shown = potential.areas.filter(a => a.level !== 'none');
        const high = potential.high.length ? `<p><b>Высокий уровень:</b> ${esc(potential.high.map(lower).join(', '))}.</p>` : '';
        const awards = potential.awards.length ? `<p><b>Государственные награды:</b> ${esc(potential.awards.map(a => a.year ? `${a.name} (${a.year})` : a.name).join('; '))}. На уровни не влияют.</p>` : '';
        return `<section class="potential-card"><h3>Потенциал</h3><p class="muted">${esc(NOTE)}</p>${high}${awards}${shown.length ? table(shown) : '<p class="muted">Направлений с опытом нет.</p>'}<details class="potential-all"><summary>Все направления</summary>${table(potential.areas)}</details></section>`;
    }

    // The table "area × level" with counts; a count opens the list of students.
    function summary(data, esc, selected) {
        if (!data.submitted) return '<p class="muted">Сданных автобиографий пока нет.</p>';
        const levels = data.areas[0].levels;
        const head = `<tr><th>Направление</th>${levels.map(l => `<th>${badge(l.level, l.levelText, esc)}</th>`).join('')}</tr>`;
        const rows = data.areas.map(a => `<tr><th scope="row">${esc(a.title)}</th>${a.levels.map(l => {
            const on = selected && selected.area === a.id && selected.level === l.level;
            return `<td>${l.students.length ? `<button class="secondary potential-count${on ? ' active' : ''}" data-area="${esc(a.id)}" data-level="${esc(l.level)}" aria-pressed="${on}">${l.students.length}</button>` : '<span class="muted">0</span>'}</td>`;
        }).join('')}</tr>`).join('');
        return `<p class="muted">Учтено сданных автобиографий: ${data.submitted}. ${esc(NOTE)}</p><div class="table-scroll"><table class="potential-summary"><thead>${head}</thead><tbody>${rows}</tbody></table></div>`;
    }

    // Students of one cell; each name opens the conclusion about the student.
    function students(data, selected, esc) {
        const area = data.areas.find(a => a.id === selected?.area);
        const group = area?.levels.find(l => l.level === selected.level);
        if (!group) return '';
        return `<div class="potential-students"><h3>${esc(area.title)} — ${esc(group.levelText)}: ${group.students.length}</h3><ul>${group.students.map(s => `<li><button class="text-button" data-answers="${esc(s.submissionId)}">${esc(s.name)}</button> <span class="muted">· взвод ${esc(s.platoon)}</span></li>`).join('')}</ul></div>`;
    }

    return {card, brief, summary, students};
})();
