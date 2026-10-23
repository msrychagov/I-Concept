window.teacherPlatoons = async function ({view, api, esc}) {
    view.innerHTML = '<div id="platoon-management" class="platoon-management"><p role="status">Загрузка взводов…</p></div><div id="cohort-management" class="platoon-management"></div><div id="instructor-management" class="platoon-management"></div>';
    const root = view.firstElementChild;
    window.teacherCohorts({view: view.children[1], api, esc});
    window.teacherInstructors({view: view.children[2], api, esc});
    let state, selected = '', issued = null;
    const options = (current = '') => state.platoons.map(p => `<option value="${esc(p.id)}" ${p.id === current ? 'selected' : ''}>${esc(p.name)}</option>`).join('');
    const message = (text, error = false) => {
        const node = root.querySelector('#platoon-feedback');
        if (node) {
            node.className = error ? 'error' : 'info';
            node.textContent = text
        }
    };

    async function refresh() {
        state = await api('/teachers/me/platoons');
        if (root.isConnected) render()
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

    function render() {
        const platoon = state.platoons.find(p => p.id === selected);
        const students = state.students.filter(s => selected === 'unassigned' ? !s.platoonId : !selected || s.platoonId === selected);
        root.innerHTML = `<header class="page-heading"><div><div class="eyebrow">${esc(state.vuc)}</div><h1>Управление взводом</h1></div></header>
   <p id="platoon-feedback" role="status" aria-live="polite"></p>
   <section class="surface settings-section"><h2>Учебные взводы</h2>
    <form id="create-platoon"><label>Название нового взвода<input name="name" required maxlength="80" placeholder="Например, 101"></label><button type="submit">Создать учебный взвод</button></form>
    <label>Показать состав<select id="platoon-filter"><option value="">Все взводы</option>${options(selected)}<option value="unassigned" ${selected === 'unassigned' ? 'selected' : ''}>Без взвода</option></select></label>
    ${platoon ? `<form id="rename-platoon"><label>Название взвода<input name="name" required maxlength="80" value="${esc(platoon.name)}"></label><button type="submit" class="secondary">Сохранить название</button></form>
    <h3>Приглашение студентов</h3><p class="muted">Ссылка действует 2 дня. Студенты автоматически попадут в ВУЦ «${esc(state.vuc)}», взвод ${esc(platoon.name)}. Новая ссылка отключит предыдущую.</p><button id="generate-link">Создать ссылку</button><div id="issued-invite"></div>` : '<p class="muted">Выберите взвод для переименования и выдачи приглашений.</p>'}
   </section>
   <section class="surface settings-section"><h2>Состав${platoon ? ' взвода ' + esc(platoon.name) : ''} · ${students.length}</h2>
    ${students.length ? `<div style="overflow-x:auto"><table><thead><tr><th>Студент</th><th>Взвод</th><th>Доступ</th><th>Действия</th></tr></thead><tbody>${students.map(s => `<tr data-student="${esc(s.id)}"><td>${esc(s.name)}<br><small>${esc(s.email)}</small></td><td><div class="platoon-controls"><select aria-label="Взвод для ${esc(s.name)}"><option value="">Без взвода</option>${options(s.platoonId)}</select><button data-action="move" class="secondary">Перевести</button></div></td><td>${s.disabled ? 'Отключён' : 'Активен'}</td><td><div class="platoon-actions">${s.platoonId ? '<button data-action="remove" class="secondary">Удалить из взвода</button>' : ''}<button data-action="${s.disabled ? 'enable' : 'disable'}" class="secondary">${s.disabled ? 'Восстановить доступ' : 'Отключить кабинет'}</button></div></td></tr>`).join('')}</tbody></table></div>` : '<p class="muted">Студентов пока нет.</p>'}
   </section>`;
        root.querySelector('#platoon-filter').onchange = e => {
            selected = e.target.value;
            issued = null;
            render()
        };
        root.querySelector('#create-platoon').onsubmit = e => {
            e.preventDefault();
            const f = e.currentTarget;
            run(f.querySelector('button'), async () => {
                const p = await api('/teachers/me/platoons', {method: 'POST', body: {name: f.elements.name.value}});
                selected = p.id;
                issued = null;
                await refresh();
                message('Взвод создан.')
            })
        };
        const rename = root.querySelector('#rename-platoon');
        if (rename) rename.onsubmit = e => {
            e.preventDefault();
            run(rename.querySelector('button'), async () => {
                await api('/teachers/me/platoons/' + encodeURIComponent(selected), {
                    method: 'PATCH',
                    body: {name: rename.elements.name.value}
                });
                await refresh();
                message('Название сохранено.')
            })
        };
        const generate = root.querySelector('#generate-link');
        if (generate) generate.onclick = () => run(generate, async () => {
            issued = await api('/teachers/me/student-registration-links', {
                method: 'POST',
                body: {platoonId: selected}
            });
            showInvite()
        });
        root.querySelectorAll('[data-action]').forEach(button => button.onclick = () => run(button, async () => {
            const row = button.closest('[data-student]'), action = button.dataset.action,
                student = state.students.find(s => s.id === row.dataset.student),
                platoonId = row.querySelector('select').value;
            if (action === 'move' && !platoonId) {
                message('Выберите взвод для перевода.', true);
                return
            }
            const prompt = action === 'disable' ? `Отключить личный кабинет студента «${student.name}»? Все активные сеансы будут завершены.` : action === 'remove' ? `Удалить студента «${student.name}» из взвода? Доступ к заданиям будет прекращён.` : null;
            if (prompt && !window.confirm(prompt)) return;
            await api('/teachers/me/students/' + encodeURIComponent(student.id), {
                method: 'PATCH',
                body: {action, platoonId}
            });
            await refresh();
            message('Изменения сохранены.');
        }));
        showInvite();
    }

    function showInvite() {
        const box = root.querySelector('#issued-invite');
        if (!box || !issued || issued.platoonId !== selected) return;
        const url = location.origin + issued.link;
        box.innerHTML = `<p>Действует до ${esc(new Date(issued.expires).toLocaleString('ru-RU'))}</p><p class="platoon-actions"><button type="button" id="copy-invite">Копировать ссылку</button> <button type="button" id="show-invite" class="secondary" aria-expanded="false" aria-controls="invite-value">Показать</button></p><div id="invite-value" hidden><label>Ссылка приглашения<input class="issued-link" readonly value="${esc(url)}"></label></div><p id="copy-status" role="status" aria-live="polite"></p>`;
        box.querySelector('#show-invite').onclick = e => {
            const field = box.querySelector('#invite-value');
            field.hidden = !field.hidden;
            e.currentTarget.textContent = field.hidden ? 'Показать' : 'Скрыть';
            e.currentTarget.setAttribute('aria-expanded', String(!field.hidden))
        };
        box.querySelector('#copy-invite').onclick = async () => {
            const status = box.querySelector('#copy-status');
            try {
                if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(url);
                else {
                    const temp = document.createElement('textarea');
                    temp.value = url;
                    temp.style.position = 'fixed';
                    temp.style.opacity = '0';
                    document.body.append(temp);
                    try {
                        temp.select();
                        if (!document.execCommand('copy')) throw new Error('copy')
                    } finally {
                        temp.remove()
                    }
                }
                status.textContent = 'Ссылка скопирована.';
            } catch {
                status.textContent = 'Браузер запретил копирование. Нажмите «Показать», чтобы скопировать ссылку вручную.'
            }
        };
    }

    try {
        await refresh()
    } catch (e) {
        if (root.isConnected) {
            root.innerHTML = `<h1>Управление взводом</h1><p class="error" role="alert">${esc(e.message)}</p><button id="retry-platoons">Повторить</button>`;
            root.querySelector('button').onclick = () => window.teacherPlatoons({view, api, esc})
        }
    }
};
