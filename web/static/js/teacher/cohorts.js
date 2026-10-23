// Cohorts (graduating classes) of the center, shown under the platoons. A cohort
// is a name, a year and platoons; an essay is handed to a cohort. Every teacher
// of the center sees the cohorts, only the creator changes them.
window.teacherCohorts = async function ({view, api, esc}) {
    view.innerHTML = '<section class="surface settings-section"><h2>Выпуски</h2><p role="status">Загрузка выпусков…</p></section>';
    let state = {platoons: [], cohorts: []}, editing = '';
    const message = (text, error = false) => {
        const node = view.querySelector('#cohort-feedback');
        if (node) {
            node.className = error ? 'error' : 'info';
            node.textContent = text
        }
    };
    const platoonName = id => (state.platoons.find(p => p.id === id) || {}).name || '—';
    // A platoon is in one cohort only: platoons of other cohorts are shown but not offered.
    const taken = current => {
        const result = {};
        state.cohorts.filter(c => c.id !== current).forEach(c => c.platoonIds.forEach(id => result[id] = c.name));
        return result
    };
    const fields = (c = {}) => {
        const busy = taken(c.id);
        return `<div class="cohort-fields">
        <label>Название<input name="name" required maxlength="80" autocomplete="off" placeholder="Например, Выпуск 2027" value="${esc(c.name || '')}"></label>
        <label>Год выпуска<input name="year" required type="number" min="2000" max="2100" inputmode="numeric" value="${esc(c.year || new Date().getFullYear() + 1)}"></label></div>
        <fieldset class="cohort-platoons"><legend>Взводы выпуска</legend>${state.platoons.length ? state.platoons.map(p => `<label class="cohort-platoon"><input type="checkbox" name="platoon" value="${esc(p.id)}" ${(c.platoonIds || []).includes(p.id) ? 'checked' : ''} ${busy[p.id] ? 'disabled' : ''}> ${esc(p.name)}${busy[p.id] ? ` <small class="muted">— уже в «${esc(busy[p.id])}»</small>` : ''}</label>`).join('') : '<p class="muted">Во ВУЦ пока нет взводов — создайте их выше.</p>'}</fieldset>`
    };
    const values = form => ({
        name: form.elements.name.value,
        year: Number(form.elements.year.value),
        platoonIds: [...form.querySelectorAll('input[name=platoon]:checked')].map(box => box.value)
    });

    async function refresh() {
        state = await api('/teachers/me/cohorts');
        if (view.isConnected) render()
    }

    async function run(button, work) {
        button.disabled = true;
        try {
            await work()
        } catch (e) {
            message(e.message, true)
        } finally {
            button.disabled = false
        }
    }

    function row(c) {
        if (c.id === editing) return `<tr data-cohort="${esc(c.id)}"><td colspan="5"><form class="edit-cohort" aria-label="Изменить: ${esc(c.name)}">${fields(c)}
            <div class="platoon-actions"><button type="submit">Сохранить</button><button type="button" class="secondary" data-action="cancel">Отмена</button></div></form></td></tr>`;
        const actions = c.mine ? `<div class="platoon-actions"><button type="button" class="secondary" data-action="edit">Изменить</button><button type="button" class="secondary" data-action="delete">Удалить</button></div>` : '<small class="muted">Правит автор</small>';
        return `<tr data-cohort="${esc(c.id)}"><td><b>${esc(c.name)}</b></td><td data-label="Год">${esc(c.year)}</td>
            <td data-label="Взводы">${c.platoonIds.length ? esc(c.platoonIds.map(platoonName).join(', ')) : '—'} <small class="muted">(студентов: ${esc(c.students)})</small></td>
            <td data-label="Создал"><small>${esc(c.createdByName || '—')}${c.mine ? ' (вы)' : ''}</small></td><td>${actions}</td></tr>`
    }

    function render() {
        view.innerHTML = `<section class="surface settings-section" aria-labelledby="cohorts-title"><h2 id="cohorts-title">Выпуски · ${state.cohorts.length}</h2>
   <p class="muted">Выпуск — это год и взводы, которые пишут эссе вместе: по нему выдаётся эссе и собирается общая сводка. Взвод входит только в один выпуск. Выпуски видят все преподаватели ВУЦ, изменять и выдавать им эссе может только тот, кто выпуск создал.</p>
   <p id="cohort-feedback" role="status" aria-live="polite"></p>
   <form id="add-cohort" aria-label="Создать выпуск">${fields()}<button type="submit">Создать выпуск</button></form>
   ${state.cohorts.length ? `<div style="overflow-x:auto"><table class="instructor-table cohort-table"><thead><tr><th>Выпуск</th><th>Год</th><th>Взводы</th><th>Создал</th><th>Действия</th></tr></thead><tbody>${state.cohorts.map(row).join('')}</tbody></table></div>` : '<p class="muted">Выпусков пока нет.</p>'}
  </section>`;
        const add = view.querySelector('#add-cohort');
        add.onsubmit = e => {
            e.preventDefault();
            run(add.querySelector('button[type=submit]'), async () => {
                const saved = await api('/teachers/me/cohorts', {method: 'POST', body: values(add)});
                await refresh();
                message(`Создан выпуск: ${saved.name}.`)
            })
        };
        const edit = view.querySelector('.edit-cohort');
        if (edit) {
            edit.elements.name.focus();
            edit.onsubmit = e => {
                e.preventDefault();
                run(edit.querySelector('button[type=submit]'), async () => {
                    await api('/teachers/me/cohorts/' + encodeURIComponent(editing), {method: 'PUT', body: values(edit)});
                    editing = '';
                    await refresh();
                    message('Изменения сохранены.')
                })
            };
        }
        view.querySelectorAll('[data-action]').forEach(button => button.onclick = () => {
            const id = button.closest('[data-cohort]').dataset.cohort, action = button.dataset.action,
                cohort = state.cohorts.find(c => c.id === id);
            if (action === 'edit' || action === 'cancel') {
                editing = action === 'edit' ? id : '';
                render();
                return
            }
            if (!window.confirm(`Удалить выпуск «${cohort.name}»?`)) return;
            run(button, async () => {
                await api('/teachers/me/cohorts/' + encodeURIComponent(id), {method: 'DELETE'});
                await refresh();
                message(`Удалён выпуск: ${cohort.name}.`)
            })
        })
    }

    try {
        await refresh()
    } catch (e) {
        if (view.isConnected) {
            view.innerHTML = `<section class="surface settings-section"><h2>Выпуски</h2><p class="error" role="alert">${esc(e.message)}</p><button id="retry-cohorts">Повторить</button></section>`;
            view.querySelector('button').onclick = () => window.teacherCohorts({view, api, esc})
        }
    }
};
