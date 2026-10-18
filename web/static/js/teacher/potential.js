// Potential by the autobiography: levels of one student.
// Levels come ready from the server; here they are only drawn.
window.potentialView = (() => {
    const NOTE = 'Уровни посчитаны программой по шкале потенциала из того, что студент сообщил о себе сам. Это не проверенная оценка способностей.';
    const lower = text => text ? text[0].toLowerCase() + text.slice(1) : '';
    const badge = (level, text, esc) => `<span class="potential-level level-${esc(level)}">${esc(text)}</span>`;

    // Levels of one submitted autobiography, for the answers dialog.
    function card(potential, esc) {
        const rows = potential.areas.map(a => `<tr><th scope="row">${esc(a.title)}</th><td>${badge(a.level, a.levelText, esc)}</td><td>${a.reasons.length ? `<ul class="potential-reasons">${a.reasons.map(r => `<li>${esc(r)}</li>`).join('')}</ul>` : '<span class="muted">Сведений нет</span>'}</td></tr>`).join('');
        const high = potential.high.length ? `<p><b>Высокий уровень:</b> ${esc(potential.high.map(lower).join(', '))}.</p>` : '<p class="muted">Направлений с высоким уровнем нет.</p>';
        const awards = potential.awards.length ? `<p><b>Государственные награды:</b> ${esc(potential.awards.map(a => a.year ? `${a.name} (${a.year})` : a.name).join('; '))}. На уровни не влияют.</p>` : '';
        return `<section class="potential-card"><h3>Потенциал</h3><p class="muted">${esc(NOTE)}</p>${high}${awards}<div class="table-scroll"><table class="potential-table"><thead><tr><th>Направление</th><th>Уровень</th><th>Из каких ответов</th></tr></thead><tbody>${rows}</tbody></table></div></section>`;
    }

    return {card};
})();
