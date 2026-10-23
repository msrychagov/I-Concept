// Student questionnaire (autobiography, essay): intro and consent, one screen per section, review and submit.
(() => {
    'use strict';
    const app = document.querySelector('#app');
    const submissionId = new URLSearchParams(location.search).get('id');
    const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[c]));
    const MONTHS = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь', 'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
    const thisYear = new Date().getFullYear();
    const MAX_BUTTONS = 5;

    let data = null;             // {submission, form, progress}
    let values = {};             // working copy of the answers, top-level field id -> value
    let view = 'intro';          // 'intro' | section id | 'review' | 'done'
    let dirty = false, busy = false;
    let fieldError = null;       // {path, message} from the last failed save
    let focusPath = '';
    const partial = {};          // month pickers with only one half chosen, by path
    const visited = new Set();   // sections whose gaps are shown

    const api = (path = '', method = 'GET', body) => window.apiRequest('/students/me/submissions/' + encodeURIComponent(submissionId) + path, {
        method,
        body
    });
    const sectionById = id => data.form.sections.find(s => s.id === id);
    const sectionIndex = id => data.form.sections.findIndex(s => s.id === id);
    const optionsOf = field => field.ref ? (data.form.refs?.[field.ref] || []) : (field.options || []);
    const asList = v => Array.isArray(v) ? v : [v];
    // The browser's own calendar day (toISOString would give the UTC day).
    const localToday = () => {
        const d = new Date();
        return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    };
    const isEmpty = v => v === undefined || v === null || v === '' || (Array.isArray(v) && !v.length);

    // ---- visibility: the same rules as on the server -------------------------
    function holds(cond, record, outer) {
        if (!cond) return true;
        const current = record[cond.field] ?? outer?.[cond.field];
        return !!current && current.some(v => cond.in.includes(v));
    }

    // Calls fn for each visible field in order; returns the visible values.
    function visit(fields, store, outer, fn) {
        const seen = {};
        for (const f of fields) {
            if (!holds(f.showIf, seen, outer)) continue;
            fn(f, seen);
            const v = store[f.id];
            if (f.type !== 'group' && !isEmpty(v)) seen[f.id] = asList(v);
        }
        return seen;
    }

    // Copy of the answers that keeps only visible, non-empty fields.
    function prune(fields, store, outer) {
        const out = {};
        visit(fields, store, outer, (f, seen) => {
            if (f.type === 'group') {
                out[f.id] = (store[f.id] || []).map(rec => prune(f.fields, rec, seen));
            } else if (!isEmpty(store[f.id])) {
                out[f.id] = typeof store[f.id] === 'string' ? store[f.id].trim() : store[f.id];
            }
        });
        return out;
    }

    // ---- rendering of fields -------------------------------------------------
    function control(f, store, path) {
        const v = store[f.id];
        const attr = `data-path="${esc(path)}"`;
        const single = `${attr} id="f-${esc(path)}"`;
        const options = optionsOf(f);
        switch (f.type) {
            case 'choice':
                if (options.length > MAX_BUTTONS) {
                    return `<select ${single} data-kind="choice"><option value="">Выберите…</option>${options.map(o => `<option value="${esc(o.id)}" ${v === o.id ? 'selected' : ''}>${esc(o.text)}</option>`).join('')}</select>`;
                }
                return `<div class="choice-list ${options.length <= 3 && options.every(o => o.text.length < 14) ? 'compact' : ''}" role="radiogroup">${options.map(o => `<label class="option ${v === o.id ? 'selected' : ''}"><input type="radio" name="${esc(path)}" ${attr} data-kind="choice" value="${esc(o.id)}" ${v === o.id ? 'checked' : ''}><span>${esc(o.text)}</span></label>`).join('')}</div>`;
            case 'multi': {
                const picked = v || [];
                return `<div class="choice-list">${options.map(o => `<label class="option ${picked.includes(o.id) ? 'selected' : ''}"><input type="checkbox" ${attr} data-kind="multi" value="${esc(o.id)}" ${picked.includes(o.id) ? 'checked' : ''}><span>${esc(o.text)}</span></label>`).join('')}</div>`;
            }
            case 'text':
                return `<input type="text" ${single} data-kind="text" maxlength="${f.maxLength || 300}" value="${esc(v)}" ${f.suggest ? `list="dl-${esc(path)}" autocomplete="off"` : ''}>${f.suggest ? `<datalist id="dl-${esc(path)}"></datalist>` : ''}`;
            case 'textarea': {
                const limit = f.maxLength || 2000, long = limit >= 1000;
                return `<textarea ${single} data-kind="text" rows="${long ? 6 : 3}" maxlength="${limit}">${esc(v)}</textarea>${long ? `<small class="char-count" aria-hidden="true">${[...(v || '')].length} / ${limit}</small>` : ''}`;
            }
            case 'date':
                return `<input type="date" ${single} data-kind="text" min="1900-01-01" max="${localToday()}" value="${esc(v)}">`;
            case 'month': {
                const [year, month] = (partial[path] || (v ? v.split('-') : ['', ''])).concat('').slice(0, 2);
                const years = [];
                for (let y = thisYear + 10; y >= 1950; y--) years.push(y);
                return `<div class="month-pick"><select ${single} data-kind="month" data-part="m" aria-label="Месяц"><option value="">Месяц</option>${MONTHS.map((m, i) => `<option value="${String(i + 1).padStart(2, '0')}" ${month === String(i + 1).padStart(2, '0') ? 'selected' : ''}>${m}</option>`).join('')}</select><select ${attr} data-kind="month" data-part="y" aria-label="Год"><option value="">Год</option>${years.map(y => `<option value="${y}" ${year === String(y) ? 'selected' : ''}>${y}</option>`).join('')}</select></div>`;
            }
            case 'year': {
                const years = [];
                for (let y = thisYear + 10; y >= 1930; y--) years.push(y);
                return `<select ${single} data-kind="text"><option value="">Год</option>${years.map(y => `<option value="${y}" ${v === String(y) ? 'selected' : ''}>${y}</option>`).join('')}</select>`;
            }
        }
        return '';
    }

    function fieldHtml(f, store, outer, prefix, missing) {
        const path = prefix + f.id;
        let html = '';
        if (f.type === 'group') {
            if (!store[f.id]) store[f.id] = [];
            if (f.min >= 1 && !store[f.id].length) store[f.id].push({});
            const records = store[f.id];
            html = `<div class="record-list">${records.map((rec, n) => {
                const inner = recordFields(f, rec, outer, `${path}.${n}.`, missing);
                return `<div class="record" data-record="${esc(path)}.${n}"><div class="record-head"><b>${esc(f.label)}${records.length > 1 || f.max > 1 ? ' · ' + (n + 1) : ''}</b><button type="button" class="secondary" data-remove="${esc(path)}" data-n="${n}">Удалить</button></div>${inner}</div>`;
            }).join('')}</div>${records.length < f.max ? `<button type="button" class="secondary add-record" data-add="${esc(path)}">＋ ${esc(f.addLabel || 'Добавить')}</button>` : ''}`;
            const gap = missing.has(f.id);
            return `<div class="form-field ${gap ? 'missing' : ''}" data-field="${esc(path)}"><span class="field-label">${esc(f.label)}${f.min >= 1 ? '<span class="req" aria-hidden="true">*</span>' : ''}</span>${f.hint ? `<p class="hint">${esc(f.hint)}</p>` : ''}${gap ? '<p class="field-error">Добавьте хотя бы одну запись.</p>' : ''}${html}${errorFor(path)}</div>`;
        }
        const gap = missing.has(path);
        const wrap = ['choice', 'multi'].includes(f.type) && (optionsOf(f).length <= MAX_BUTTONS || f.type === 'multi');
        const title = (f.number ? `<span class="q-number">${esc(f.number)}</span> ` : '') + esc(f.label) + (f.required ? '<span class="req" aria-hidden="true">*</span>' : '');
        const body = control(f, store, path);
        const hint = f.hint ? `<p class="hint">${esc(f.hint)}</p>` : '';
        const note = gap ? '<p class="field-error">Обязательное поле.</p>' : '';
        if (wrap) return `<fieldset class="form-field ${gap ? 'missing' : ''}" data-field="${esc(path)}"><legend>${title}</legend>${hint}${body}${note}${errorFor(path)}</fieldset>`;
        return `<div class="form-field ${gap ? 'missing' : ''}" data-field="${esc(path)}"><label class="field-label" for="f-${esc(path)}">${title}</label>${hint}${body}${note}${errorFor(path)}</div>`;
    }

    const errorFor = path => fieldError && fieldError.path === path ? `<p class="field-error" role="alert">${esc(fieldError.message)}</p>` : '';

    function renderFields(fields, store, outer, prefix, missing) {
        let html = '';
        visit(fields, store, outer, (f, seen) => {
            if (f.type === 'group') store[f.id] = store[f.id] || [];
            html += fieldHtml(f, store, seen, prefix, missing);
        });
        return html;
    }

    // A record is rendered like a level of fields whose conditions may also refer to the section.
    function recordFields(group, rec, sectionSeen, prefix, missing) {
        let html = '';
        visit(group.fields, rec, sectionSeen, f => {
            html += fieldHtml(f, rec, sectionSeen, prefix, missing);
        });
        return html;
    }

    // ---- screens -------------------------------------------------------------
    function header() {
        const done = data.progress.complete.length;
        return `<header class="student-header"><a href="/student">← Личный кабинет</a><span>${esc(data.form.title)}</span><span class="muted">Заполнено разделов: ${done} из ${data.progress.total}</span></header><div class="progress-track"><div style="width:${done / data.progress.total * 100}%"></div></div>`;
    }

    function sectionState(id) {
        if (data.progress.complete.includes(id)) return 'done';
        if (data.submission.savedSections.includes(id)) return 'gaps';
        return '';
    }

    function nav() {
        const mark = {done: '✓', gaps: '!', '': ''};
        return `<aside class="module-nav form-nav"><div class="eyebrow">Разделы</div>${data.form.sections.map((s, i) => {
            const state = sectionState(s.id);
            return `<button type="button" data-go="${esc(s.id)}" class="${view === s.id ? 'active' : ''} ${state}"><span class="section-name">${i + 1}. ${esc(s.title)}</span><span class="section-mark" aria-label="${state === 'done' ? 'заполнен' : state === 'gaps' ? 'есть пропуски' : ''}">${mark[state]}</span></button>`;
        }).join('')}<button type="button" data-go="review" class="review-link ${view === 'review' ? 'active' : ''}"><span class="section-name">Проверка и сдача</span></button></aside>`;
    }

    function render() {
        if (!data) return;
        const s = data.submission;
        if (view === 'done' || s.completed) {
            const texts = data.form.texts || {};
            app.innerHTML = `<main class="consent"><div class="eyebrow">${esc(data.form.title)}</div><h1>${esc(texts.doneTitle || 'Автобиография сдана')}</h1><p>${esc(texts.doneText || 'Спасибо. Сведения переданы преподавателю. Изменить ответы после сдачи нельзя.')}</p><a class="button" href="/student">Вернуться в кабинет</a></main>`;
            return;
        }
        if (!s.consent) {
            app.innerHTML = `<div class="student-layout"><header class="student-header"><a href="/student">← Личный кабинет</a><span>${esc(data.form.title)}</span></header><main class="consent"><div class="eyebrow">Перед началом</div><h1>${esc(data.form.title)}</h1>${data.form.intro.split('\n\n').map(p => `<p>${esc(p)}</p>`).join('')}<button id="consent" type="button">${esc(data.form.texts?.consent || 'Согласен(на) заполнять автобиографию и сохранять сведения')}</button><div id="err"></div></main></div>`;
            app.querySelector('#consent').onclick = giveConsent;
            return;
        }
        const body = view === 'review' ? reviewHtml() : sectionHtml(sectionById(view));
        app.innerHTML = `<div class="student-layout">${header()}<div class="survey-workspace">${nav()}<main class="question-surface">${body}</main></div></div>`;
        bind();
        if (focusPath) {
            const el = app.querySelector(`[data-path="${CSS.escape(focusPath)}"]:checked`) || app.querySelector(`[data-path="${CSS.escape(focusPath)}"]`);
            if (el) el.focus({preventScroll: true});
            focusPath = '';
        }
    }

    function sectionHtml(section) {
        const i = sectionIndex(section.id), last = i === data.form.sections.length - 1;
        const missing = new Set(visited.has(section.id) ? data.progress.missing[section.id] || [] : []);
        const fields = renderFields(section.fields, values, {}, '', missing);
        return `<div class="question-meta"><span>Раздел ${i + 1} из ${data.form.sections.length}</span><span>* — обязательно</span></div><h1 class="form-section-title">${esc(section.title)}</h1>${section.hint ? `<p class="section-hint">${esc(section.hint)}</p>` : ''}<form id="section-form" novalidate>${fields}</form><div id="err"></div><div class="question-footer"><button type="button" class="secondary" id="prev" ${i === 0 ? 'disabled' : ''}>Назад</button><span id="status" class="form-status" role="status"></span><button type="button" id="next">${last ? 'К проверке' : 'Далее'}</button></div>`;
    }

    function labelFor(sectionId, path) {
        const section = sectionById(sectionId);
        const [id, n, nested] = path.split('.');
        const f = section.fields.find(x => x.id === id);
        if (!f) return path;
        if (nested === undefined) return (f.number ? f.number + ' ' : '') + f.label;
        const inner = f.fields.find(x => x.id === nested);
        return `${f.label} № ${Number(n) + 1}: ${inner ? inner.label : nested}`;
    }

    function reviewHtml() {
        const p = data.progress, ready = p.complete.length === p.total;
        return `<div class="eyebrow">Проверка</div><h1 class="form-section-title">Проверка и сдача</h1><p class="muted">Заполнено разделов: ${p.complete.length} из ${p.total}. ${esc(data.form.texts?.review || 'Сдать автобиографию можно, когда заполнено всё обязательное. После сдачи ответы изменить нельзя.')}</p><ul class="review-list">${data.form.sections.map((s, i) => {
            const state = sectionState(s.id), gaps = p.missing[s.id] || [];
            return `<li class="${state}"><span class="review-mark">${state === 'done' ? '✓' : '!'}</span><div><button type="button" class="link" data-go="${esc(s.id)}">${i + 1}. ${esc(s.title)}</button>${state === 'done' ? '' : gaps.length ? `<ul>${gaps.slice(0, 6).map(g => `<li>${esc(labelFor(s.id, g))}</li>`).join('')}${gaps.length > 6 ? `<li>…и ещё ${gaps.length - 6}</li>` : ''}</ul>` : '<ul><li>Раздел нужно открыть и нажать «Далее»</li></ul>'}</div></li>`;
        }).join('')}</ul><label class="confirm-line"><input type="checkbox" id="confirm" ${ready ? '' : 'disabled'}> Подтверждаю, что сведения верны</label><div id="err"></div><div class="question-footer"><button type="button" class="secondary" id="prev">Назад</button><span></span><button type="button" id="submit" disabled>Сдать</button></div>`;
    }

    // ---- events --------------------------------------------------------------
    // Path "group.N.field" resolves to the record store; a plain id to the section store.
    function locate(path) {
        const [id, n, nested] = path.split('.');
        if (nested === undefined) return {store: values, key: id};
        return {store: values[id][Number(n)], key: nested};
    }

    function fieldDef(path) {
        const [id, , nested] = path.split('.');
        const top = sectionById(view).fields.find(f => f.id === id);
        return nested === undefined ? top : top.fields.find(f => f.id === nested);
    }

    function change(path, value, rerender) {
        const {store, key} = locate(path);
        if (isEmpty(value)) delete store[key]; else store[key] = value;
        dirty = true;
        fieldError = null;
        if (rerender) {
            focusPath = path;
            render();
        }
    }

    let suggestTimer = 0;

    function suggest(input) {
        clearTimeout(suggestTimer);
        suggestTimer = setTimeout(async () => {
            const f = fieldDef(input.dataset.path), list = app.querySelector(`#dl-${CSS.escape(input.dataset.path)}`);
            if (!f || !list) return;
            try {
                const r = await api('/suggestions?field=' + encodeURIComponent(f.id) + '&q=' + encodeURIComponent(input.value));
                list.innerHTML = r.values.map(v => `<option value="${esc(v)}">`).join('');
            } catch (e) { /* suggestions are optional */
            }
        }, 250);
    }

    function bind() {
        app.querySelectorAll('[data-go]').forEach(b => b.onclick = () => go(b.dataset.go));
        const prev = app.querySelector('#prev'), next = app.querySelector('#next');
        if (prev) prev.onclick = () => go(view === 'review' ? data.form.sections.at(-1).id : data.form.sections[sectionIndex(view) - 1].id);
        if (next) next.onclick = () => go(sectionIndex(view) === data.form.sections.length - 1 ? 'review' : data.form.sections[sectionIndex(view) + 1].id, true);
        if (view === 'review') {
            const confirm = app.querySelector('#confirm'), submit = app.querySelector('#submit');
            confirm.onchange = () => {
                submit.disabled = !confirm.checked;
            };
            submit.onclick = finish;
            return;
        }
        const form = app.querySelector('#section-form');
        form.addEventListener('input', e => {
            const el = e.target, path = el.dataset.path, kind = el.dataset.kind;
            if (!path) return;
            if (kind === 'text') {
                change(path, el.value, false);
                if (el.tagName === 'INPUT' && el.list) suggest(el);
                const counter = el.tagName === 'TEXTAREA' && el.nextElementSibling;
                if (counter && counter.classList.contains('char-count')) counter.textContent = `${[...el.value].length} / ${el.maxLength}`;
            }
        });
        form.addEventListener('change', e => {
            const el = e.target, path = el.dataset.path, kind = el.dataset.kind;
            if (!path) return;
            if (kind === 'choice') {
                change(path, el.value, true);
            } else if (kind === 'multi') {
                const picked = [...form.querySelectorAll(`[data-path="${CSS.escape(path)}"]:checked`)].map(x => x.value);
                change(path, picked, true);
            } else if (kind === 'month') {
                const box = el.closest('.month-pick');
                const m = box.querySelector('[data-part=m]').value, y = box.querySelector('[data-part=y]').value;
                if (m && y) {
                    delete partial[path];
                    change(path, `${y}-${m}`, false);
                } else {
                    partial[path] = [y, m];
                    change(path, '', false);
                }
            } else if (kind === 'text') {
                change(path, el.value, false);
            }
            if (el.type === 'radio' || el.type === 'checkbox') {
                el.closest('.choice-list')?.querySelectorAll('.option').forEach(o => o.classList.toggle('selected', o.querySelector('input').checked));
            }
        });
        form.addEventListener('focusin', e => {
            const el = e.target;
            if (el.tagName === 'INPUT' && el.list && !el.value) suggest(el);
        });
        form.addEventListener('click', e => {
            const add = e.target.closest('[data-add]'), remove = e.target.closest('[data-remove]');
            if (add) {
                const {store, key} = locate(add.dataset.add);
                (store[key] = store[key] || []).push({});
                dirty = true;
                render();
            } else if (remove) {
                const {store, key} = locate(remove.dataset.remove);
                store[key].splice(Number(remove.dataset.n), 1);
                dirty = true;
                render();
            }
        });
    }

    // ---- server calls --------------------------------------------------------
    function showError(text, extra = '') {
        const el = app.querySelector('#err');
        if (el) el.innerHTML = `<p class="error" role="alert">${esc(text)}${extra}</p>`;
    }

    function apply(result) {
        data.submission.revision = result.revision;
        data.submission.updated = result.updated;
        if (result.progress) data.progress = result.progress;
    }

    function conflict(e) {
        showError(e.message, ' <button type="button" id="reload">Обновить страницу</button>');
        app.querySelector('#reload')?.addEventListener('click', () => location.reload());
    }

    async function giveConsent() {
        try {
            apply(await api('/consent', 'PUT', {value: true, revision: data.submission.revision}));
            data.submission.consent = true;
            view = data.form.sections[0].id;
            render();
        } catch (e) {
            e.details?.conflict ? conflict(e) : showError(e.message);
        }
    }

    // Saves the open section; returns false and shows the error if the server rejects it.
    async function save() {
        const section = sectionById(view);
        const answers = prune(section.fields, values, {});
        const status = app.querySelector('#status');
        if (status) status.textContent = 'Сохранение…';
        try {
            apply(await api('/sections/' + encodeURIComponent(section.id), 'PUT', {
                answers,
                revision: data.submission.revision
            }));
            for (const f of section.fields) {
                if (f.id in answers) values[f.id] = answers[f.id]; else delete values[f.id];
                data.submission.answers[f.id] = values[f.id];
                if (!(f.id in answers)) delete data.submission.answers[f.id];
            }
            if (!data.submission.savedSections.includes(section.id)) data.submission.savedSections.push(section.id);
            visited.add(section.id);
            dirty = false;
            fieldError = null;
            return true;
        } catch (e) {
            if (status) status.textContent = '';
            if (e.details?.conflict) {
                conflict(e);
                return false;
            }
            fieldError = e.details?.field ? {path: e.details.field, message: e.message} : null;
            if (fieldError) {
                render();
                app.querySelector('[data-field] .field-error[role=alert]')?.scrollIntoView({block: 'center'});
            } else showError(e.message);
            return false;
        }
    }

    async function go(target, advance = false) {
        if (busy || target === view) return;
        busy = true;
        try {
            const from = view;
            const neverSaved = from !== 'review' && advance && !data.submission.savedSections.includes(from);
            if (from !== 'review' && (dirty || neverSaved) && !await save()) return;
            if (from === 'review' && target !== 'review') visited.add(target);
            view = target;
            scrollTo({top: 0});
            render();
        } finally {
            busy = false;
        }
    }

    async function finish() {
        if (busy) return;
        busy = true;
        app.querySelector('#submit').disabled = true;
        try {
            await api('/completion', 'POST', {revision: data.submission.revision});
            dirty = false;
            view = 'done';
            render();
        } catch (e) {
            if (e.details?.conflict) conflict(e);
            else {
                if (e.details?.missing) {
                    data.progress.missing = e.details.missing;
                    render();
                }
                showError(e.message);
            }
        } finally {
            busy = false;
        }
    }

    window.addEventListener('beforeunload', e => {
        if (dirty) {
            e.preventDefault();
            e.returnValue = '';
        }
    });

    async function load() {
        if (!submissionId) {
            location.replace('/student');
            return;
        }
        try {
            data = await api();
            document.title = data.form.title + ' · ВУЦ';
            values = structuredClone(data.submission.answers || {});
            // Gaps of sections saved earlier are shown right away, also after a reload.
            data.submission.savedSections.forEach(id => visited.add(id));
            view = data.submission.consent ? firstUnfinished() : 'intro';
            render();
        } catch (e) {
            app.innerHTML = `<main class="consent"><h1>Анкета недоступна</h1><p class="error">${esc(e.message)}</p><a class="button" href="/student">Вернуться в кабинет</a></main>`;
        } finally {
            window.pageLoader.hide();
        }
    }

    function firstUnfinished() {
        const open = data.form.sections.find(s => !data.progress.complete.includes(s.id));
        return open ? open.id : 'review';
    }

    load();
})();
