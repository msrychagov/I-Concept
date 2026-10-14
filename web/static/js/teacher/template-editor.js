window.templateEditor = function (ctx) {
    if (!document.querySelector('#template-editor-style')) {
        const link = document.createElement('link');
        link.id = 'template-editor-style';
        link.rel = 'stylesheet';
        link.href = '/assets/css/template-editor.css';
        document.head.append(link)
    }
    const {view, state, esc, api} = ctx;
    const p = ctx.profile;
    if (!p.questions) p.questions = structuredClone(state.bank.items);
    let search = '', section = '', indicator = '', mode = 'questions', dirty = false, undo = null, page = 1,
        pageSize = 10, indicatorPage = 1, indicatorPageSize = 10;
    const included = i => !state.bank.indicators.some(x => x.id === i.indicator) || Object.hasOwn(p.values, i.indicator);
    const mark = () => {
        dirty = true;
        ctx.onDraft(p);
        const el = view.querySelector('#draft-status');
        if (el) el.textContent = 'Есть несохранённые изменения'
    };
    const moduleOptions = value => Object.entries(state.bank.modules).map(([id, name]) => `<option value="${id}" ${id === value ? 'selected' : ''}>${esc(name)}</option>`).join('');
    const indicatorOptions = value => state.bank.indicators.filter(i => Object.hasOwn(p.values, i.id)).map(i => `<option value="${i.id}" ${i.id === value ? 'selected' : ''}>${i.id} · ${esc(i.name)}</option>`).join('');
    const scored = i => i.scored || state.bank.indicators.some(ind => ind.items.includes(i.id));
    const title = i => state.bank.indicators.find(x => x.id === i.indicator)?.name || i.indicator;

    function draw() {
        view.innerHTML = `<header class="page-heading"><div><div class="eyebrow">Конструктор тестирования</div><h1>${ctx.editingProfileId ? 'Редактирование шаблона' : 'Новый шаблон'}</h1></div><button id="save-template">Сохранить шаблон</button></header>
 <div class="editor-status"><span id="draft-status">${dirty ? 'Есть несохранённые изменения' : 'Все изменения применяются после сохранения'}</span><span id="editor-count"></span></div>
 <section class="surface"><div class="form-grid"><label>Название шаблона<input id="template-name" maxlength="150" value="${esc(p.name)}"></label><label>Автор<input id="template-author" value="${esc(p.author)}"></label><label>Этап (необязательно)<input id="template-wave" value="${esc(p.wave)}"></label></div></section>
 <div class="toolbar"><button id="questions-tab" class="${mode === 'questions' ? '' : 'secondary'}">Вопросы</button><button id="indicators-tab" class="${mode === 'indicators' ? '' : 'secondary'}">Показатели</button><button id="add-question" class="secondary">+ Добавить вопрос</button></div>
 <div id="editor-error" role="alert"></div><div id="undo-area"></div>
 <section class="surface" id="question-panel"><div class="editor-filters"><label>Поиск<input id="question-search" placeholder="Формулировка или код вопроса" value="${esc(search)}"></label><label>Раздел<select id="section-filter"><option value="">Все разделы</option>${moduleOptions(section)}</select></label><label>Показатель<select id="indicator-filter"><option value="">Все показатели</option>${indicatorOptions(indicator)}</select></label></div><div id="question-list"></div><nav id="question-pagination" class="question-pagination" aria-label="Страницы вопросов"></nav></section>
 <section class="surface" id="indicator-panel"><p>Исключение показателя убирает связанные вопросы из тестирования. Их можно вернуть вместе с показателем.</p><div id="indicator-list" class="indicator-grid"></div><nav id="indicator-pagination" class="list-pagination" aria-label="Страницы показателей"></nav></section>
 <section class="surface"><h2>Сохранённые шаблоны</h2>${state.profiles.map(t => `<div class="section-title"><p><b>${esc(t.name)}</b><small>${Object.keys(t.values || {}).length} показателей · ${esc(t.author)}</small></p><button class="secondary" data-edit-template="${esc(t.id)}">Редактировать</button></div>`).join('') || '<p>Пока нет сохранённых шаблонов.</p>'}${ctx.editingProfileId ? '<button id="new-template" class="secondary">Создать другой шаблон</button>' : ''}</section><dialog id="question-dialog" class="question-dialog"></dialog>`;
        for (const [id, key] of [['template-name', 'name'], ['template-author', 'author'], ['template-wave', 'wave']]) view.querySelector('#' + id).oninput = e => {
            p[key] = e.target.value;
            mark()
        };
        view.querySelector('#question-search').oninput = e => {
            search = e.target.value;
            page = 1;
            list()
        };
        view.querySelector('#section-filter').value = section;
        view.querySelector('#indicator-filter').value = indicator;
        view.querySelector('#section-filter').onchange = e => {
            section = e.target.value;
            page = 1;
            list()
        };
        view.querySelector('#indicator-filter').onchange = e => {
            indicator = e.target.value;
            page = 1;
            list()
        };
        view.querySelector('#questions-tab').onclick = () => {
            mode = 'questions';
            draw()
        };
        view.querySelector('#indicators-tab').onclick = () => {
            mode = 'indicators';
            draw()
        };
        view.querySelector('#question-panel').hidden = mode !== 'questions';
        view.querySelector('#indicator-panel').hidden = mode !== 'indicators';

        view.querySelector('#add-question').onclick = () => edit(null);
        view.querySelectorAll('[data-edit-template]').forEach(el => el.onclick = () => {
            if (!dirty || confirm('Открыть другой шаблон без сохранения текущих изменений?')) ctx.onSelect(el.dataset.editTemplate)
        });
        view.querySelector('#new-template')?.addEventListener('click', () => {
            if (!dirty || confirm('Создать шаблон без сохранения текущих изменений?')) ctx.onSelect('')
        });
        view.querySelector('#save-template').onclick = async () => {
            const button = view.querySelector('#save-template');
            button.disabled = true;
            try {
                p.group = '';
                p.basis = '';
                p.approved = true;
                await api(ctx.editingProfileId ? '/templates/' + encodeURIComponent(ctx.editingProfileId) : '/templates', {
                    method: ctx.editingProfileId ? 'PUT' : 'POST',
                    body: p
                });
                dirty = false;
                await ctx.onSaved()
            } catch (e) {
                view.querySelector('#editor-error').innerHTML = `<p class="error">${esc(e.message)}</p>`;
                button.disabled = false
            }
        };
        list();
        indicators();
    }

    function indicators() {
        const pagination = window.listPagination(view.querySelector('#indicator-pagination'), {
            total: state.bank.indicators.length,
            page: indicatorPage,
            pageSize: indicatorPageSize,
            onChange: (next, size) => {
                indicatorPage = next;
                indicatorPageSize = size;
                indicators()
            }
        });
        indicatorPage = pagination.page;
        view.querySelector('#indicator-list').innerHTML = state.bank.indicators.slice(pagination.start, pagination.end).map(i => `<label class="indicator-choice"><input type="checkbox" data-indicator="${i.id}" ${Object.hasOwn(p.values, i.id) ? 'checked' : ''}><span><b>${i.id} · ${esc(i.name)}</b>${i.definition ? `<small>${esc(i.definition)}</small>` : ''}<small>${p.questions.filter(q => q.indicator === i.id).length} вопросов</small></span></label>`).join('') || '<p class="muted">Показателей пока нет.</p>';
        view.querySelectorAll('[data-indicator]').forEach(el => el.onchange = () => {
            if (el.checked) p.values[el.dataset.indicator] = 5; else delete p.values[el.dataset.indicator];
            mark();
            if (indicator && !Object.hasOwn(p.values, indicator)) indicator = '';
            const filter = view.querySelector('#indicator-filter');
            filter.innerHTML = '<option value="">Все показатели</option>' + indicatorOptions(indicator);
            filter.value = indicator;
            list()
        });
    }

    function list() {
        const active = p.questions.filter(included),
            filtered = active.filter(i => (!section || (i.section || i.module) === section) && (!indicator || i.indicator === indicator) && (!search || (i.text + ' ' + i.id).toLowerCase().includes(search.toLowerCase())));
        const pages = Math.max(1, Math.ceil(filtered.length / pageSize));
        page = Math.min(page, pages);
        const start = (page - 1) * pageSize;
        view.querySelector('#editor-count').textContent = `${Object.keys(p.values).length} показателей · ${active.length} вопросов с дополнительными модулями`;
        view.querySelector('#question-list').innerHTML = `<p class="muted">Найдено: ${filtered.length}. ${section === 'E' ? 'Раздел наблюдателя.' : section === 'F' || section === 'V' ? 'Дополнительный модуль включается при создании ссылки.' : ''}</p>` + filtered.slice(start, start + pageSize).map(i => `<article class="editor-question"><div><small>${esc(i.id)} · ${esc(state.bank.modules[i.section || i.module])} · ${esc(title(i))}</small><p>${esc(i.text)}</p><span class="editor-badge">${scored(i) ? 'Входит в расчёт показателя' : 'Дополнительная оценка'}${i.reverse ? ' · обратный вопрос' : ''}</span></div><div class="editor-question-actions"><button class="secondary" data-edit="${esc(i.id)}">Редактировать</button><button class="text-button" data-remove="${esc(i.id)}">Удалить</button></div></article>`).join('');
        if (!filtered.length) view.querySelector('#question-list').innerHTML = '<p class="editor-empty">Вопросы не найдены. Измените поиск или фильтры либо добавьте новый вопрос.</p>';
        const pagination = view.querySelector('#question-pagination');
        pagination.innerHTML = `<label class="page-size">На странице<select id="question-page-size">${[10, 20, 50].map(n => `<option value="${n}" ${n === pageSize ? 'selected' : ''}>${n}</option>`).join('')}</select></label><span class="pagination-summary" role="status" tabindex="-1">${filtered.length ? `${start + 1}–${Math.min(start + pageSize, filtered.length)} из ${filtered.length}` : '0 вопросов'}</span><div class="pagination-controls"><button type="button" class="secondary" id="questions-prev" ${page === 1 ? 'disabled' : ''} aria-label="Предыдущая страница вопросов">← Назад</button><span>Страница ${page} из ${pages}</span><button type="button" class="secondary" id="questions-next" ${page === pages ? 'disabled' : ''} aria-label="Следующая страница вопросов">Далее →</button></div>`;
        pagination.querySelector('#question-page-size').onchange = e => {
            pageSize = Number(e.target.value);
            page = 1;
            list()
        };
        const turn = delta => {
            page += delta;
            list();
            view.querySelector('#question-pagination .pagination-summary').focus({preventScroll: true});
            view.querySelector('#question-panel').scrollIntoView({block: 'start'})
        };
        pagination.querySelector('#questions-prev').onclick = () => turn(-1);
        pagination.querySelector('#questions-next').onclick = () => turn(1);
        view.querySelectorAll('[data-edit]').forEach(b => b.onclick = () => edit(p.questions.find(i => i.id === b.dataset.edit)));
        view.querySelectorAll('[data-remove]').forEach(b => b.onclick = () => {
            const n = p.questions.findIndex(i => i.id === b.dataset.remove);
            undo = {item: p.questions[n], index: n};
            p.questions.splice(n, 1);
            mark();
            list()
        });
        view.querySelector('#undo-area').innerHTML = undo ? `<div class="info">Вопрос ${esc(undo.item.id)} удалён из шаблона. <button id="undo-delete" class="secondary">Отменить удаление</button></div>` : '';
        view.querySelector('#undo-delete')?.addEventListener('click', () => {
            p.questions.splice(undo.index, 0, undo.item);
            undo = null;
            mark();
            list()
        });
    }

    function edit(original) {
        const first = Object.keys(p.values)[0];
        if (!original && !first) {
            view.querySelector('#editor-error').innerHTML = '<p class="error">Сначала включите хотя бы один показатель.</p>';
            return
        }
        const item = structuredClone(original || {
            id: 'CUSTOM_' + crypto.randomUUID(),
            text: '',
            module: 'B',
            section: section || 'B',
            indicator: indicator || first,
            type: 'likert',
            min: 1,
            max: 5,
            scored: true
        });
        const dialog = view.querySelector('#question-dialog');
        dialog.innerHTML = `<form id="question-form"><div class="section-title"><h2>${original ? 'Редактировать вопрос' : 'Добавить вопрос'}</h2><button type="button" class="secondary" id="close-question" aria-label="Закрыть редактор">×</button></div><label>Формулировка<textarea id="question-text" rows="5" required maxlength="12000">${esc(item.text)}</textarea></label><div class="form-grid"><label>Раздел в анкете<select id="question-section">${moduleOptions(item.section || item.module)}</select></label><label>Показатель<select id="question-indicator">${indicatorOptions(item.indicator)}</select></label></div><p class="muted">Раздел меняет расположение вопроса. Связь с расчётом и тип ответа сохраняются.</p>${!original || scored(item) ? `<label class="check"><input id="question-reverse" type="checkbox" ${item.reverse ? 'checked' : ''}> Обратное кодирование (6 − ответ)</label>` : ''}${item.options?.length ? `<details><summary>Варианты ответа (${item.options.length})</summary>${item.options.map((o, j) => `<label>Вариант ${j + 1}<textarea data-option="${j}" required>${esc(o.text)}</textarea></label>`).join('')}</details>` : `<p>Шкала ответа: ${item.min}–${item.max}.</p>`}<div class="toolbar"><button type="submit">Применить к шаблону</button><button type="button" class="secondary" id="cancel-question">Отмена</button></div></form>`;
        // Auxiliary items may have indicators outside the 39 primary indicators.
        const sel = dialog.querySelector('#question-indicator');
        if (original && !state.bank.indicators.some(i => i.id === item.indicator)) {
            const opt = document.createElement('option');
            opt.value = item.indicator;
            opt.textContent = item.indicator;
            sel.append(opt);
            sel.value = item.indicator;
            sel.disabled = true
        }
        dialog.querySelector('#close-question').onclick = () => dialog.close();
        dialog.querySelector('#cancel-question').onclick = () => dialog.close();
        dialog.querySelector('#question-form').onsubmit = e => {
            e.preventDefault();
            item.text = dialog.querySelector('#question-text').value.trim();
            if (!item.text) return;
            item.section = dialog.querySelector('#question-section').value;
            item.indicator = sel.value;
            if (dialog.querySelector('#question-reverse')) item.reverse = dialog.querySelector('#question-reverse').checked;
            if (original && scored(original)) item.scored = true;
            dialog.querySelectorAll('[data-option]').forEach(el => item.options[Number(el.dataset.option)].text = el.value);
            if (original) p.questions[p.questions.findIndex(i => i.id === original.id)] = item; else p.questions.push(item);
            mark();
            dialog.close();
            list()
        };
        dialog.showModal();
    }

    draw();
};
