// Teacher section "Эссе": give the essay to a cohort, follow who has written it,
// read an essay with the analysis of the model, and the summary of the cohort.
// Counts and groupings come from the server; the model's words are drawn as given.
window.teacherEssays = async function ({view, api, esc}) {
    const FORM = 'essay';
    const MOMENTS = {beliefs: 'Убеждения', emotions: 'Эмоции', values: 'Ценности', abilities: 'Восприятие своих возможностей'};
    let data, current = '', summary = null, search = '', platoon = '', status = '', form = null, timer = 0;
    const fmtDate = v => v ? new Date(v).toLocaleDateString('ru-RU') : '';
    const alive = () => view.isConnected;
    const waiting = s => s === 'queued' || s === 'working';
    const error = e => {
        const el = view.querySelector('#essay-error');
        if (el) el.innerHTML = `<p class="error" role="alert">${esc(e.message || e)}</p>`;
    };
    const nl = text => esc(text).replace(/\n/g, '<br>');

    async function reload() {
        data = await api('/essays');
        if (!data.assignments.some(a => a.id === current)) current = data.assignments[0]?.id || '';
        summary = current ? await api('/essays/assignments/' + encodeURIComponent(current) + '/summary') : null;
        if (alive()) draw();
        schedule();
    }

    // While an analysis or the summary is being made the page asks again.
    function schedule() {
        clearTimeout(timer);
        const as = data?.assignments.find(a => a.id === current);
        const busy = as?.students.some(s => waiting(s.analysis)) || waiting(summary?.status);
        if (busy && alive()) timer = setTimeout(() => alive() && reload().catch(error), 4000);
    }

    // ---- the page --------------------------------------------------------
    function metrics(c) {
        return `<div class="dashboard-metrics"><article><strong>${c.students}</strong>Студентов в выпуске</article><article><strong>${c.submitted}</strong>Сдали эссе</article><article><strong>${c.notSubmitted}</strong>Не сдали</article><article><strong>${c.analysed}</strong>С готовым разбором</article></div>`;
    }

    function draw() {
        const as = data.assignments.find(a => a.id === current);
        view.innerHTML = `<header class="page-heading"><div><div class="eyebrow">Кабинет преподавателя</div><h1>Эссе</h1></div><div class="toolbar"><button id="preview" class="secondary">Посмотреть вопросы</button></div></header><div id="essay-error" role="alert"></div>
  <section class="surface"><h2>Выдать эссе выпуску</h2>${data.cohorts.length ? `<form id="essay-assign"><div class="form-grid"><label>Название<input name="title" required maxlength="200" value="Эссе"></label><label>Выпуск<select name="cohort" required>${data.cohorts.map(c => `<option value="${esc(c.id)}">${esc(c.name)} · ${esc(c.year)}${c.platoons.length ? ' · взводы ' + esc(c.platoons.join(', ')) : ' · без взводов'}</option>`).join('')}</select></label></div><button type="submit">Выдать эссе</button><p class="muted" style="margin-top:1rem">Эссе получат все студенты взводов выпуска, кто бы их ни пригласил. Когда писать — во время учёбы или по окончании — решаете вы: выдайте эссе тогда, когда оно нужно. Эссе одно и не редактируется.</p></form>` : '<p class="muted">Сначала создайте выпуск и добавьте в него взводы: страница «Управление взводом», раздел «Выпуски».</p>'}</section>
  <section class="surface"><h2>Выданные эссе</h2>${data.assignments.map(a => `<div class="assignment-row ${a.id === current ? 'is-current' : ''}"><div><b>${esc(a.title)}</b><small>${esc(a.cohortName)} · ${esc(a.year)} · ${a.active ? 'Открыто' : 'Закрыто'} · сдано ${a.counts.submitted} из ${a.counts.students}</small></div><div class="platoon-actions"><button class="secondary" data-open="${esc(a.id)}" ${a.id === current ? 'aria-pressed="true"' : ''}>${a.id === current ? 'Выбрано' : 'Открыть'}</button><button class="secondary" data-toggle="${esc(a.id)}" data-active="${!a.active}">${a.active ? 'Закрыть доступ' : 'Возобновить'}</button></div></div>`).join('') || '<p class="muted">Эссе пока никому не выдавали.</p>'}</section>
  ${as ? `<section class="surface"><h2>${esc(as.title)} — ${esc(as.cohortName)}</h2>${metrics(as.counts)}${as.counts.withoutAnalysis ? `<p class="muted">Сдано эссе без готового разбора: ${as.counts.withoutAnalysis}. В сводку они не входят.</p>` : ''}<div class="dashboard-filters"><label>ФИО<input id="essay-search" placeholder="Поиск по фамилии или имени" value="${esc(search)}"></label><label>Взвод<select id="essay-platoon"><option value="">Все взводы</option>${[...new Set(as.students.map(s => s.platoon))].map(p => `<option value="${esc(p)}">${esc(p)}</option>`).join('')}</select></label><label>Статус<select id="essay-status"><option value="">Любой</option><option value="submitted">Сдано</option><option value="writing">Пишет</option><option value="not_started">Не начато</option></select></label></div><div id="essay-table"></div></section>
  <section class="surface" id="essay-summary"></section>` : ''}<dialog id="essay-dialog" class="result-dialog"></dialog>`;
        view.querySelector('#preview').onclick = preview;
        const assign = view.querySelector('#essay-assign');
        if (assign) assign.onsubmit = async e => {
            e.preventDefault();
            const button = assign.querySelector('button');
            button.disabled = true;
            try {
                await api('/assignments', {method: 'POST', body: {title: assign.elements.title.value, formId: FORM, cohortId: assign.elements.cohort.value}});
                current = '';
                await reload();
            } catch (err) {
                error(err);
                button.disabled = false;
            }
        };
        view.querySelectorAll('[data-open]').forEach(b => b.onclick = async () => {
            current = b.dataset.open;
            platoon = status = search = '';
            try {
                await reload();
            } catch (err) {
                error(err);
            }
        });
        view.querySelectorAll('[data-toggle]').forEach(b => b.onclick = async () => {
            try {
                await api('/assignments/' + encodeURIComponent(b.dataset.toggle), {method: 'PATCH', body: {active: b.dataset.active === 'true'}});
                await reload();
            } catch (err) {
                error(err);
            }
        });
        if (!as) return;
        view.querySelector('#essay-platoon').value = platoon;
        view.querySelector('#essay-status').value = status;
        view.querySelector('#essay-search').oninput = e => {
            search = e.target.value;
            table();
        };
        view.querySelector('#essay-platoon').onchange = e => {
            platoon = e.target.value;
            table();
        };
        view.querySelector('#essay-status').onchange = e => {
            status = e.target.value;
            table();
        };
        table();
        drawSummary();
    }

    const stateText = s => s.status === 'submitted' ? `Сдано ${fmtDate(s.completedAt)}` : s.status === 'writing' ? `Пишет: ${s.answered} из ${s.total}` : 'Не начато';
    const analysisText = s => s.status !== 'submitted' ? '—' : {ready: 'Готов', queued: 'Составляется…', working: 'Составляется…', failed: 'Не удалось', none: 'Не составлен', off: 'Модель не подключена', earlier: 'В прежнем виде'}[s.analysis] || '—';

    function table() {
        const as = data.assignments.find(a => a.id === current);
        const rows = as.students.filter(s => (!platoon || s.platoon === platoon) && (!status || s.status === status) && (!search || s.name.toLowerCase().includes(search.toLowerCase())));
        view.querySelector('#essay-table').innerHTML = `<p class="muted">Найдено: ${rows.length}</p><div class="table-scroll"><table><thead><tr><th>ФИО</th><th>Взвод</th><th>Эссе</th><th>Разбор</th><th></th></tr></thead><tbody>${rows.map(s => `<tr><td><b>${esc(s.name)}</b></td><td>${esc(s.platoon)}</td><td>${esc(stateText(s))}</td><td>${esc(analysisText(s))}${s.hasPrevious ? '<small class="muted"> · есть прежнее эссе</small>' : ''}</td><td>${s.submissionId ? `<button class="secondary" data-essay="${esc(s.submissionId)}">Эссе</button>` : '—'}</td></tr>`).join('') || '<tr><td colspan="5">Студенты не найдены. Измените поиск или фильтры.</td></tr>'}</tbody></table></div>`;
        view.querySelectorAll('[data-essay]').forEach(b => b.onclick = () => openEssay(b.dataset.essay));
    }

    // ---- dialogs -----------------------------------------------------------
    function openDialog(html) {
        const dialog = view.querySelector('#essay-dialog');
        dialog.innerHTML = html;
        if (!dialog.open) dialog.showModal();
        dialog.querySelector('[data-close]')?.addEventListener('click', () => dialog.close());
        return dialog;
    }

    async function preview() {
        openDialog('<h2>Загрузка вопросов…</h2>');
        try {
            form = form || await api('/forms/' + FORM);
            openDialog(`<div class="section-title"><div><h2>${esc(form.title)}</h2><p>Вопросы одинаковы для эссе во время учёбы и по окончании. * — обязательные.</p></div><button class="secondary" data-close>Закрыть</button></div>${window.formView.questions(form, esc)}`);
        } catch (err) {
            openDialog(`<p class="error">${esc(err.message)}</p><button class="secondary" data-close>Закрыть</button>`);
        }
    }

    // ---- the analysis of one essay ------------------------------------------
    let portrait = null, instructors = null;
    const answerMap = (form, submission) => {
        const result = {};
        form.sections.forEach(s => s.fields.forEach(f => {
            if (f.number && submission.answers?.[f.id] !== undefined) result[f.number] = submission.answers[f.id];
        }));
        return result;
    };
    // The full answer under a quote is headed by its question, not by its number.
    const questionMap = form => {
        const result = {};
        form.sections.forEach(s => s.fields.forEach(f => {
            if (f.number) result[f.number] = f.label;
        }));
        return result;
    };
    // A quote shows the number of the question; a click opens the whole answer.
    const quote = (q, source, esc) => `<button type="button" class="quote-chip" data-quote="${esc(q.question)}" data-source="${source}"><span>«${esc(q.quote)}»</span><small>вопрос ${esc(q.question)}</small></button>`;
    const quotes = (list, esc, source = 'current') => `<div class="quote-list">${(list || []).map(q => quote(q, source, esc)).join('')}</div>`;
    const tags = (p, esc) => `${p.about ? `<span class="essay-tag">${esc(p.about)}</span>` : ''}${p.basis === 'вывод по тексту' ? '<span class="essay-tag inference">вывод по тексту</span>' : ''}`;

    // Two instructors fit a name they share; rarely more.
    const fitWord = list => list.length === 2 ? 'подходят двое' : 'подходят несколько';

    function who(label) {
        const found = instructors?.find(i => `[преподаватель ${i.number}]` === label);
        if (!found) return {text: label, short: label, listed: false};
        const initials = [found.name, found.patronymic].filter(Boolean).map(n => n[0] + '.').join(' ');
        return {
            text: [found.surname, found.name, found.patronymic].filter(Boolean).join(' ') + (found.rank ? ', ' + found.rank : ''),
            short: found.surname + (initials ? ' ' + initials : '') + (found.rank ? ', ' + found.rank : ''),
            listed: true
        };
    }

    // v2 (decided 25 October 2026): paragraphs without lists and question numbers;
    // the quotes of each paragraph and recommendation are behind «Из чего вывод».
    const plainQuote = (q, source, esc, label = '') => `<button type="button" class="quote-chip" data-quote="${esc(q.question)}" data-source="${source}"><span>${label ? `<small>${esc(label)}</small> ` : ''}«${esc(q.quote)}»</span></button>`;
    const why = (inner, label = 'Из чего вывод') => `<details class="essay-why"><summary>${esc(label)}</summary><div class="essay-why-body">${inner}</div></details>`;
    const quotesWhy = (list, esc, before = []) => why(`<div class="quote-list">${before.map(q => plainQuote(q, 'previous', esc, 'из прежнего эссе')).join('')}${(list || []).map(q => plainQuote(q, 'current', esc)).join('')}</div>`);

    function teacherLines(list, esc) {
        if (!list.length) return '';
        return `<section class="essay-findings-teachers"><h3>Преподаватели</h3><ul class="essay-points">${list.map(t => {
            const w = who(t.who);
            const fit = (t.candidates || []).map(l => who(l).short).join('; ');
            return `<li><b>${esc(w.text)}</b>${t.candidates ? ` <span class="essay-tag">${fitWord(t.candidates)}: ${esc(fit)}</span>` : w.listed ? '' : ' <small class="muted">(как назвал автор)</small>'} <span class="essay-tag">${esc(t.kind)}</span> ${esc(t.text)}${quotesWhy(t.quotes, esc)}</li>`;
        }).join('')}</ul></section>`;
    }

    function analysisV2(o, esc) {
        const paragraph = (p, before) => `<div class="essay-paragraph"><p>${esc(p.text)}</p>${quotesWhy(p.quotes, esc, before)}</div>`;
        return `<section class="essay-findings"><h3>Выводы по эссе</h3><p class="muted essay-note">Выводы по тексту эссе. Из чего сделан вывод — по кнопке под абзацем.</p>
${paragraph(o.formation, o.formation.before || [])}${paragraph(o.moments)}${paragraph(o.environment)}
<h3>Рекомендации</h3><ol class="essay-advice">${o.recommendations.map(r => `<li>${esc(r.text)}${quotesWhy(r.quotes, esc)}</li>`).join('')}</ol></section>${teacherLines(o.teachers, esc)}`;
    }

    window.portraitRenderers.essay = {
        analysis: (o, esc) => {
            if (o.formation) return analysisV2(o, esc);
            const points = (title, list) => `<div><h3>${title}</h3>${list.length ? `<ul class="essay-points">${list.map(p => `<li>${tags(p, esc)} ${esc(p.text)}${quotes(p.quotes, esc)}</li>`).join('')}</ul>` : '<p class="muted">Нет.</p>'}</div>`;
            const teachers = o.teachers.length ? `<section><h3>Преподаватели</h3><ul class="essay-points">${o.teachers.map(t => {
                const w = who(t.who);
                const fit = (t.candidates || []).map(l => who(l).short).join('; ');
                return `<li><b>${esc(w.text)}</b>${t.candidates ? ` <span class="essay-tag">${fitWord(t.candidates)}: ${esc(fit)}</span>` : w.listed ? '' : ' <small class="muted">(как назвал автор)</small>'} <span class="essay-tag">${esc(t.kind)}</span> ${esc(t.text)}${quotes(t.quotes, esc)}</li>`;
            }).join('')}</ul></section>` : '';
            const change = o.change.length ? `<section><h3>Что изменилось</h3>${o.change.map(c => `<div class="change-pair"><b>${esc(c.moment)}</b><p>${esc(c.text)}</p><div class="change-quotes"><div><small class="muted">Было</small>${quotes([c.before], esc, 'previous')}</div><div><small class="muted">Стало</small>${quotes([c.after], esc)}</div></div></div>`).join('')}</section>` : '';
            return `<section class="portrait-summary"><h3>Путь автора</h3><p>${esc(o.summary)}</p></section>
<section class="portrait-summary"><h3>Итог</h3><p>${esc(o.conclusion.text)}</p>${quotes(o.conclusion.quotes, esc)}</section>
<section><h3>Четыре момента</h3>${Object.keys(MOMENTS).map(k => `<div class="essay-moment"><b>${MOMENTS[k]}</b><p>${esc(o.moments[k].text)}</p>${quotes(o.moments[k].quotes, esc)}</div>`).join('')}</section>
<section class="portrait-lists">${points('Положительное', o.positive)}${points('Отрицательное', o.negative)}${points('На что обратить внимание', o.attention)}</section>${teachers}${change}`;
        }
    };

    async function openEssay(id) {
        portrait?.stop();
        openDialog('<h2>Загрузка эссе…</h2>');
        try {
            const [d, list] = await Promise.all([api('/submissions/' + encodeURIComponent(id)), instructors ? Promise.resolve(null) : api('/teachers/me/instructors').catch(() => null)]);
            if (list) instructors = list.instructors;
            const s = d.submission, student = d.student, context = d.essay || {};
            const maps = {current: answerMap(d.form, s), previous: context.previous ? answerMap(d.form, context.previous) : {}};
            const questions = questionMap(d.form);
            const dialog = openDialog(`<div class="section-title"><div><h2>${esc(student?.name || 'Студент')}</h2><p>Взвод ${esc(student?.platoon || '')} · ${esc(context.assignmentTitle || d.form.title)}${s.completed ? ' · сдано ' + fmtDate(s.completedAt) : ''}</p></div><button class="secondary" data-close>Закрыть</button></div>
   ${s.completed ? '' : `<p class="draft-note">Не сдано: заполнено разделов ${d.progress.complete.length} из ${d.progress.total}. Показан черновик, разбора пока нет.</p>`}
   <div id="essay-analysis"></div><div id="quote-view" aria-live="polite"></div>
   ${context.previous ? `<details class="essay-previous"><summary>Прежнее эссе (${esc(context.previous.assignmentTitle || 'без названия')}, сдано ${fmtDate(context.previous.completedAt)})</summary>${window.formView.answers(d.form, context.previous, esc)}</details>` : ''}
   <h3 class="answers-heading">Ответы</h3>${window.formView.answers(d.form, s, esc)}`);
            dialog.onclick = e => {
                const chip = e.target.closest('[data-quote]');
                if (!chip) return;
                const text = maps[chip.dataset.source]?.[chip.dataset.quote];
                const box = dialog.querySelector('#quote-view');
                box.innerHTML = text === undefined ? '' : `<blockquote class="quote-full"><small class="muted">${chip.dataset.source === 'previous' ? 'Прежнее эссе' : 'Эссе'}: ${esc(questions[chip.dataset.quote] || '')}</small><p>${nl(text)}</p><button type="button" class="secondary" data-quote-close>Свернуть</button></blockquote>`;
                box.querySelector('[data-quote-close]')?.addEventListener('click', () => box.innerHTML = '');
                box.scrollIntoView({block: 'nearest', behavior: 'smooth'});
            };
            if (!s.completed) return;
            const box = dialog.querySelector('#essay-analysis');
            const run = (first = 'GET') => {
                portrait?.stop();
                portrait = window.portraitView.start(api, FORM, id, {
                    alive: () => dialog.open && box.isConnected,
                    update: v => {
                        box.innerHTML = (v.status === 'ready' ? window.portraitRenderers.essay.analysis(v.portrait.output, esc) : '') + window.portraitView.status(v, esc) + window.portraitView.signature(v, esc);
                        box.querySelector('[data-portrait-retry]')?.addEventListener('click', () => run('POST'));
                    }
                }, first);
            };
            run();
        } catch (err) {
            openDialog(`<p class="error">${esc(err.message)}</p><button class="secondary" data-close>Закрыть</button>`);
        }
    }

    // ---- the summary -------------------------------------------------------
    // Sources of a common point: the points of the analyses it stands on.
    const authorsWord = n => n % 10 === 1 && n % 100 !== 11 ? 'автор' : [2, 3, 4].includes(n % 10) && ![12, 13, 14].includes(n % 100) ? 'автора' : 'авторов';

    function sources(group, esc, label = `у ${group.authors} из ${group.of}`) {
        return `<details class="essay-sources"><summary>${label}</summary><ul>${group.sources.map(s => `<li><button type="button" class="link" data-essay="${esc(s.submissionId)}">${esc(s.student)}</button> <small class="muted">взвод ${esc(s.platoon)}</small> ${tags(s, esc)} ${esc(s.text)}${s.quotes.length ? `<div class="quote-list">${s.quotes.map(q => `<span class="quote-chip static"><span>${q.label ? esc(q.label) + ': ' : ''}«${esc(q.quote)}»</span><small>вопрос ${esc(q.question)}</small></span>`).join('')}</div>` : ''}</li>`).join('')}</ul></details>`;
    }

    // A source of v2 is a paragraph or a recommendation of an analysis: its text, no quotes.
    const sourcesWhy = group => why(`<ul class="essay-source-list">${group.sources.map(s => `<li><button type="button" class="link" data-essay="${esc(s.submissionId)}">${esc(s.student)}</button> <small class="muted">взвод ${esc(s.platoon)}</small> ${esc(s.text)}</li>`).join('')}</ul>`, `Из чего вывод · ${group.authors} ${authorsWord(group.authors)}`);

    function summaryBodyV2(sm) {
        const o = sm.output, c = sm.common;
        const paragraph = key => `<div class="essay-paragraph"><p>${esc(o[key].text)}</p>${sourcesWhy(c[key])}</div>`;
        const left = c.notIncluded.length ? `<details class="essay-left"><summary>Не вошло в общие рекомендации: ${c.notIncluded.length}</summary><p class="muted">Рекомендации из разборов, на которые не опирается ни одна рекомендация сводки.</p><ul class="essay-source-list">${c.notIncluded.map(s => `<li><button type="button" class="link" data-essay="${esc(s.submissionId)}">${esc(s.student)}</button> <small class="muted">взвод ${esc(s.platoon)}</small> ${esc(s.text)}</li>`).join('')}</ul></details>` : '';
        return `<section class="essay-findings"><h3>Выводы по выпуску</h3><p class="muted essay-note">Выводы по разборам эссе. Из чего сделан вывод — по кнопке под абзацем.</p>
${['formation', 'moments', 'environment'].map(paragraph).join('')}
<h3>Рекомендации</h3>${o.recommendations.length ? `<ol class="essay-advice">${o.recommendations.map((r, i) => `<li>${esc(r.text)}${sourcesWhy(c.recommendations[i])}</li>`).join('')}</ol>` : '<p class="muted">Общих рекомендаций не нашлось.</p>'}${left}</section>`;
    }

    const byAuthors = (items, groups) => items.map((item, i) => ({item, group: groups[i]})).sort((a, b) => (b.group?.authors || 0) - (a.group?.authors || 0));

    function summaryBody(sm) {
        if (sm.output.formation) return summaryBodyV2(sm);
        const o = sm.output, c = sm.common;
        const list = (title, items, groups) => `<div><h3>${title}</h3>${items.length ? `<ul class="essay-points">${byAuthors(items, groups).map(({item, group}) => `<li><span class="essay-tag">${esc(item.about)}</span> ${esc(item.text)} ${sources(group, esc)}</li>`).join('')}</ul>` : '<p class="muted">Общего не нашлось.</p>'}</div>`;
        const left = c.notIncluded.length ? `<div class="essay-left"><h3>Не вошло в общие</h3><p class="muted">Пункты «на что обратить внимание», которые встретились только у одного автора: они не потерялись.</p><ul class="essay-points">${c.notIncluded.map(s => `<li><button type="button" class="link" data-essay="${esc(s.submissionId)}">${esc(s.student)}</button> <small class="muted">взвод ${esc(s.platoon)}</small> ${tags(s, esc)} ${esc(s.text)}${s.quotes.length ? `<div class="quote-list">${s.quotes.map(q => `<span class="quote-chip static"><span>«${esc(q.quote)}»</span><small>вопрос ${esc(q.question)}</small></span>`).join('')}</div>` : ''}</li>`).join('')}</ul></div>` : '';
        return `<div class="portrait-summary"><h3>Общая картина</h3><p>${esc(o.overview)}</p></div>
<h3>Типичные пути</h3>${o.paths.length ? `<ul class="essay-points">${o.paths.map((p, i) => `<li>${esc(p.text)} ${sources(c.paths[i], esc, c.paths[i].authors + ' ' + authorsWord(c.paths[i].authors))}</li>`).join('')}</ul>` : '<p class="muted">Общих путей не нашлось.</p>'}
<h3>Четыре момента</h3>${Object.keys(MOMENTS).map(k => `<div class="essay-moment"><b>${MOMENTS[k]}</b><p>${esc(o.moments[k].text)}</p>${sources(c.moments[k], esc)}</div>`).join('')}
<h3>Связь со средой</h3>${o.environment.length ? `<ul class="essay-points">${o.environment.map((p, i) => `<li><span class="essay-tag">${esc(p.moment)}</span> ${esc(p.text)} ${sources(c.environment[i], esc)}</li>`).join('')}</ul>` : '<p class="muted">Общего не нашлось.</p>'}
<div class="portrait-lists">${list('Положительное', o.positive, c.positive)}${list('Отрицательное', o.negative, c.negative)}${list('На что обратить внимание', o.attention, c.attention)}</div>${left}
${o.change.length ? `<h3>Что обычно меняется</h3>${o.change.map((p, i) => `<div class="essay-moment"><b>${esc(p.moment)}</b><p>${esc(p.text)}</p>${sources(c.change[i], esc)}</div>`).join('')}` : ''}
<div class="portrait-summary"><h3>Вывод по выпуску</h3><p>${esc(o.conclusion.text)}</p>${sources(c.conclusion, esc)}</div>`;
    }

    function teachersBlock(block) {
        const listed = block.listed || [], unlisted = block.unlisted || [];
        if (!listed.length && !unlisted.length) return '<p class="muted">Преподавателей в эссе не называли.</p>';
        // Without the numbers of questions (decided 25 October 2026): the quotes are behind the button.
        const line = (l, name = '') => `<li>${name}<span class="essay-tag">${esc(l.kind)}</span> <button type="button" class="link" data-essay="${esc(l.submissionId)}">${esc(l.student)}</button> <small class="muted">взвод ${esc(l.platoon)}</small>: ${esc(l.text)}${l.quotes.length ? why(`<div class="quote-list">${l.quotes.map(q => `<span class="quote-chip static"><span>«${esc(q.quote)}»</span></span>`).join('')}</div>`) : ''}</li>`;
        return `${listed.map(e => `<div class="essay-teacher"><b>${esc(e.who)}</b> <small class="muted">благодарностей ${e.thanks}, проблем ${e.problems}, упоминаний ${e.mentions}</small><ul class="essay-points">${e.lines.map(line).join('')}</ul></div>`).join('')}
${unlisted.length ? `<div class="essay-teacher"><b>Не сопоставлены со списком</b> <small class="muted">как написал автор</small><ul class="essay-points">${unlisted.map(l => `${line(l, `<b>${esc(l.who)}</b>${l.candidates ? ` <span class="essay-tag">${fitWord(l.candidates)}: ${esc(l.candidates.join('; '))}</span>` : ''} `)}`).join('')}</ul></div>` : ''}`;
    }

    function drawSummary() {
        const box = view.querySelector('#essay-summary');
        if (!box || !summary) return;
        const c = summary.counts, sm = summary.summary;
        const busy = waiting(summary.status);
        let note = '';
        if (!summary.canCompose) note = `<span class="muted">Нужно хотя бы два эссе с разбором (сейчас ${c.analysed}).</span>`;
        else if (summary.status === 'off' && !sm) note = '<span class="muted">Языковая модель не подключена.</span>';
        else if (busy) note = '<span role="status">Сводка готовится…</span>';
        else if (summary.status === 'failed') note = `<span role="alert"><b>Не удалось составить сводку.</b> <span class="muted">${esc(summary.reason || '')}</span></span>`;
        else if (sm) note = `<span class="muted">Составлена ${fmtDate(sm.created)} по ${sm.essays} эссе.</span>${sm.stale ? ' <b>Есть новые эссе — обновить.</b>' : sm.older ? ' <b>Сводка в прежнем виде — обновить.</b>' : ''}`;
        const label = summary.status === 'failed' ? 'Повторить' : sm ? 'Обновить сводку' : 'Составить сводку';
        box.innerHTML = `<h2>Сводка по выпуску</h2><p class="muted">В сводке эссе с готовым разбором: ${c.analysed}. Сдано ${c.submitted}, не сдано ${c.notSubmitted}, у ${c.withPrevious} есть прежнее эссе${c.withoutAnalysis ? `, без разбора ${c.withoutAnalysis} (в сводку не вошли)` : ''}.</p>
  <div class="toolbar"><button id="compose" ${!summary.canCompose || busy || summary.status === 'off' ? 'disabled' : ''}>${label}</button> ${note}</div><div id="summary-error" role="alert"></div>
  ${sm ? summaryBody(sm) : ''}<h2 style="margin-top:1.5rem">Преподаватели</h2><p class="muted">Что сказали о преподавателях из списка ВУЦ. Считает программа по разборам; сводка модели для этого не нужна.</p>${teachersBlock(summary.teachers)}`;
        box.querySelector('#compose').onclick = async e => {
            e.target.disabled = true;
            try {
                summary = await api('/essays/assignments/' + encodeURIComponent(current) + '/summary', {method: 'POST'});
                drawSummary();
                schedule();
            } catch (err) {
                box.querySelector('#summary-error').innerHTML = `<p class="error">${esc(err.message)}</p>`;
                e.target.disabled = false;
            }
        };
        box.querySelectorAll('[data-essay]').forEach(b => b.onclick = () => openEssay(b.dataset.essay));
    }

    view.innerHTML = '<p role="status">Загрузка эссе…</p>';
    try {
        await reload();
    } catch (e) {
        view.innerHTML = `<p class="error">${esc(e.message)}</p><a href="/teacher">Войти в кабинет</a>`;
    }
};
