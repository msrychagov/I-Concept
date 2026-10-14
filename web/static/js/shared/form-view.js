// Read-only views of a questionnaire for teachers: a student's answers with
// human-readable labels, and the list of questions without answers.
window.formView = (() => {
    const MONTHS = ['январь', 'февраль', 'март', 'апрель', 'май', 'июнь', 'июль', 'август', 'сентябрь', 'октябрь', 'ноябрь', 'декабрь'];
    const optionsOf = (form, field) => field.ref ? (form.refs?.[field.ref] || []) : (field.options || []);
    const optionText = (form, field, id) => optionsOf(form, field).find(o => o.id === id)?.text ?? id;
    const nl = (esc, text) => esc(text).replace(/\n/g, '<br>');

    function valueText(form, field, value, esc) {
        switch (field.type) {
            case 'choice':
                return esc(optionText(form, field, value));
            case 'multi':
                return esc(value.map(v => optionText(form, field, v)).join(', '));
            case 'date':
                return esc(value.split('-').reverse().join('.'));
            case 'month': {
                const [y, m] = value.split('-');
                return esc(`${MONTHS[Number(m) - 1] || m} ${y}`);
            }
            case 'textarea':
                return nl(esc, value);
            default:
                return esc(value);
        }
    }

    function rows(form, fields, store, esc) {
        let html = '';
        for (const f of fields) {
            const v = store?.[f.id];
            if (v === undefined || v === null) continue;
            if (f.type === 'group') {
                if (!v.length) continue;
                html += `<dt>${esc(f.label)}</dt><dd>${v.map((rec, n) => `<div class="answer-record"><b>${n + 1}.</b><dl class="answers-list">${rows(form, f.fields, rec, esc) || '<dt></dt><dd class="muted">Запись пуста</dd>'}</dl></div>`).join('')}</dd>`;
            } else {
                html += `<dt>${esc(f.label)}</dt><dd>${valueText(form, f, v, esc)}</dd>`;
            }
        }
        return html;
    }

    // Answers grouped by section. savedSections tells which sections the student has saved.
    function answers(form, submission, esc) {
        return form.sections.map((s, i) => {
            const body = rows(form, s.fields, submission.answers, esc);
            const saved = (submission.savedSections || []).includes(s.id);
            return `<section class="answers-section"><h3>${i + 1}. ${esc(s.title)}</h3>${body ? `<dl class="answers-list">${body}</dl>` : `<p class="muted">${saved ? 'Ответов нет.' : 'Раздел не заполнялся.'}</p>`}</section>`;
        }).join('');
    }

    function condition(form, fields, outer, field, esc) {
        if (!field.showIf) return '';
        const source = [...fields, ...(outer || [])].find(f => f.id === field.showIf.field);
        if (!source) return '';
        const values = field.showIf.in.map(v => optionText(form, source, v));
        return `<small class="muted">Если «${esc(source.label)}»: ${esc(values.join(' / '))}</small>`;
    }

    function questionList(form, fields, outer, esc) {
        return `<ul class="question-list">${fields.map(f => {
            const options = optionsOf(form, f);
            const kind = {
                choice: 'один вариант',
                multi: 'несколько вариантов',
                text: 'текст',
                textarea: 'текст',
                date: 'дата',
                month: 'месяц и год',
                year: 'год',
                group: 'повторяемые записи'
            }[f.type];
            const choices = options.length ? (f.ref ? ` — список (${options.length})` : `: ${esc(options.map(o => o.text).join(' · '))}`) : '';
            return `<li><b>${esc(f.label)}</b>${f.required ? ' <span class="req">*</span>' : ''} <span class="muted">(${kind}${choices})</span>${f.hint ? `<br><small class="muted">${esc(f.hint)}</small>` : ''}${condition(form, fields, outer, f, esc) ? '<br>' + condition(form, fields, outer, f, esc) : ''}${f.type === 'group' ? questionList(form, f.fields, fields, esc) : ''}</li>`;
        }).join('')}</ul>`;
    }

    // The questions of every section, without answers.
    function questions(form, esc) {
        return form.sections.map((s, i) => `<section class="answers-section"><h3>${i + 1}. ${esc(s.title)}</h3>${questionList(form, s.fields, null, esc)}</section>`).join('');
    }

    return {answers, questions};
})();
