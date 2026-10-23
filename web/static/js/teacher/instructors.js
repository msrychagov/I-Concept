// The instructor list of the center, shown under the platoons. One list per
// center: every teacher of the center sees and edits it.
window.teacherInstructors = async function ({view, api, esc}) {
    view.innerHTML = '<section class="surface settings-section"><h2>Преподаватели ВУЦ</h2><p role="status">Загрузка списка…</p></section>';
    let list = [], editing = '';
    const fullName = p => [p.surname, p.name, p.patronymic].filter(Boolean).join(' ');
    const fields = (p = {}) => `<div class="instructor-fields">
        <label>Фамилия<input name="surname" required maxlength="60" autocomplete="off" value="${esc(p.surname || '')}"></label>
        <label>Имя<input name="name" required maxlength="60" autocomplete="off" value="${esc(p.name || '')}"></label>
        <label>Отчество<input name="patronymic" maxlength="60" autocomplete="off" value="${esc(p.patronymic || '')}"></label>
        <label>Звание<input name="rank" maxlength="80" autocomplete="off" placeholder="Например, полковник" value="${esc(p.rank || '')}"></label></div>`;
    const values = form => ({
        surname: form.elements.surname.value,
        name: form.elements.name.value,
        patronymic: form.elements.patronymic.value,
        rank: form.elements.rank.value
    });
    const message = (text, error = false) => {
        const node = view.querySelector('#instructor-feedback');
        if (node) {
            node.className = error ? 'error' : 'info';
            node.textContent = text
        }
    };

    async function refresh() {
        list = (await api('/teachers/me/instructors')).instructors;
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

    function changed(p) {
        const when = p.updated ? new Date(p.updated).toLocaleDateString('ru-RU') : '';
        return esc([p.updatedByName, when].filter(Boolean).join(', '))
    }

    function row(p) {
        if (p.id === editing) return `<tr data-instructor="${esc(p.id)}"><td colspan="4"><form class="edit-instructor" aria-label="Изменить: ${esc(fullName(p))}">${fields(p)}
            <div class="platoon-actions"><button type="submit">Сохранить</button><button type="button" class="secondary" data-action="cancel">Отмена</button></div></form></td></tr>`;
        return `<tr data-instructor="${esc(p.id)}"><td><b>${esc(fullName(p))}</b></td><td data-label="Звание">${esc(p.rank || '—')}</td><td data-label="Изменил"><small>${changed(p)}</small></td>
            <td><div class="platoon-actions"><button type="button" class="secondary" data-action="edit">Изменить</button><button type="button" class="secondary" data-action="delete">Удалить</button></div></td></tr>`
    }

    function render() {
        view.innerHTML = `<section class="surface settings-section" aria-labelledby="instructors-title"><h2 id="instructors-title">Преподаватели ВУЦ · ${list.length}</h2>
   <p class="muted">Общий список ВУЦ: его видят и правят все преподаватели ВУЦ. По нему программа узнаёт преподавателя в текстах студентов, как бы его ни назвали — «полковник Иванов» или «Игорь Петрович». Учётные записи в список сами не попадают: добавьте всех, кто преподаёт.</p>
   <p id="instructor-feedback" role="status" aria-live="polite"></p>
   <form id="add-instructor" aria-label="Добавить преподавателя">${fields()}<button type="submit">Добавить преподавателя</button></form>
   ${list.length ? `<div style="overflow-x:auto"><table class="instructor-table"><thead><tr><th>Преподаватель</th><th>Звание</th><th>Изменил</th><th>Действия</th></tr></thead><tbody>${list.map(row).join('')}</tbody></table></div>` : '<p class="muted">В списке пока никого нет.</p>'}
  </section>`;
        const add = view.querySelector('#add-instructor');
        add.onsubmit = e => {
            e.preventDefault();
            run(add.querySelector('button[type=submit]'), async () => {
                const saved = await api('/teachers/me/instructors', {method: 'POST', body: values(add)});
                await refresh();
                message(`Добавлен: ${fullName(saved)}.`)
            })
        };
        const edit = view.querySelector('.edit-instructor');
        if (edit) {
            edit.elements.surname.focus();
            edit.onsubmit = e => {
                e.preventDefault();
                run(edit.querySelector('button[type=submit]'), async () => {
                    await api('/teachers/me/instructors/' + encodeURIComponent(editing), {method: 'PUT', body: values(edit)});
                    editing = '';
                    await refresh();
                    message('Изменения сохранены.')
                })
            };
        }
        view.querySelectorAll('[data-action]').forEach(button => button.onclick = () => {
            const id = button.closest('[data-instructor]').dataset.instructor, action = button.dataset.action,
                person = list.find(p => p.id === id);
            if (action === 'edit' || action === 'cancel') {
                editing = action === 'edit' ? id : '';
                render();
                return
            }
            if (!window.confirm(`Удалить «${fullName(person)}» из списка преподавателей ВУЦ?`)) return;
            run(button, async () => {
                await api('/teachers/me/instructors/' + encodeURIComponent(id), {method: 'DELETE'});
                await refresh();
                message(`Удалён: ${fullName(person)}.`)
            })
        });
    }

    try {
        await refresh()
    } catch (e) {
        if (view.isConnected) {
            view.innerHTML = `<section class="surface settings-section"><h2>Преподаватели ВУЦ</h2><p class="error" role="alert">${esc(e.message)}</p><button id="retry-instructors">Повторить</button></section>`;
            view.querySelector('button').onclick = () => window.teacherInstructors({view, api, esc})
        }
    }
};
