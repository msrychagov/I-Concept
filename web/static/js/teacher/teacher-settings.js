window.teacherAccountAPI = async function () {
    return (await window.apiRequest('/teachers/me')).teacher
};
window.showTeacherIdentity = function (teacher) {
    const name = document.querySelector('#teacher-name'), avatar = document.querySelector('#teacher-avatar');
    if (name) name.textContent = teacher.name;
    if (avatar) avatar.textContent = teacher.name.trim().split(/\s+/).slice(0, 2).map(part => Array.from(part)[0]).join('').toUpperCase();
};
window.teacherSettings = async function ({view, esc}) {
    view.innerHTML = '<div id="teacher-settings"><p role="status">Загрузка настроек…</p></div>';
    const root = view.querySelector('#teacher-settings');
    let teacher;
    try {
        teacher = await window.teacherAccountAPI()
    } catch (e) {
        if (root.isConnected) root.innerHTML = `<p class="error" role="alert">${esc(e.message)}</p><button class="secondary" id="retry-settings">Повторить</button>`;
        root.querySelector('#retry-settings')?.addEventListener('click', () => window.teacherSettings({view, esc}));
        return
    }
    if (!root.isConnected) return;
    window.showTeacherIdentity(teacher);
    root.innerHTML = `<header class="page-heading"><div><div class="eyebrow">Личный профиль преподавателя</div><h1>Настройки</h1></div></header>
 <section class="surface settings-section"><h2>Военный учебный центр</h2><p>${esc(teacher.vuc || 'ВУЦ не назначен — обратитесь к администратору.')}</p><p class="muted">ВУЦ назначает администратор. Приглашения и состав студентов доступны в разделе «Управление взводом».</p></section>
 <section class="surface settings-section"><h2>Личные данные</h2><form id="teacher-profile-form"><div class="form-grid"><label>ФИО<input name="name" autocomplete="name" required maxlength="200" value="${esc(teacher.name)}"></label><label>Электронная почта<input name="email" type="email" autocomplete="username" required maxlength="254" value="${esc(teacher.email)}"></label></div><p class="muted">Электронная почта используется для входа в кабинет.</p><label id="email-password-field" hidden>Текущий пароль для смены почты<input name="currentPassword" type="password" autocomplete="current-password" maxlength="256" disabled></label><div class="settings-feedback" role="status" aria-live="polite"></div><button type="submit">Сохранить профиль</button></form></section>
 <section class="surface settings-section"><h2>Смена пароля</h2><form id="teacher-password-form"><label>Текущий пароль<input name="currentPassword" type="password" autocomplete="current-password" required maxlength="256"></label><div class="form-grid"><label>Новый пароль<input name="newPassword" type="password" autocomplete="new-password" required minlength="10" maxlength="256" aria-describedby="password-help"></label><label>Повторите новый пароль<input name="confirmPassword" type="password" autocomplete="new-password" required minlength="10" maxlength="256"></label></div><p class="muted" id="password-help">От 10 до 256 символов. После смены пароля потребуется заново войти на других устройствах.</p><div class="settings-feedback" role="status" aria-live="polite"></div><button type="submit">Изменить пароль</button></form></section>`;
    const profileForm = root.querySelector('#teacher-profile-form'),
        passwordForm = root.querySelector('#teacher-password-form');
    const emailField = root.querySelector('#email-password-field');
    const syncEmail = () => {
        const changed = profileForm.elements.email.value.trim().toLowerCase() !== teacher.email;
        emailField.hidden = !changed;
        profileForm.elements.currentPassword.disabled = !changed;
        profileForm.elements.currentPassword.required = changed
    };
    profileForm.elements.email.oninput = syncEmail;
    const feedback = (form, message, error = false) => {
        form.querySelector('.settings-feedback').innerHTML = `<p class="${error ? 'error' : 'info'}" ${error ? 'role="alert"' : ''}>${esc(message)}</p>`
    };
    const save = async (form, passwordChange) => {
        const fields = Object.fromEntries(new FormData(form)), button = form.querySelector('button[type=submit]');
        if (passwordChange && fields.newPassword !== fields.confirmPassword) {
            feedback(form, 'Новый пароль и подтверждение не совпадают.', true);
            passwordForm.elements.confirmPassword.focus();
            return
        }
        button.disabled = true;
        form.querySelector('.settings-feedback').textContent = 'Сохранение…';
        try {
            const updated = await window.apiRequest(passwordChange ? '/teachers/me/password' : '/teachers/me', {
                method: passwordChange ? 'PUT' : 'PATCH',
                body: fields
            });
            teacher = updated.teacher;
            window.showTeacherIdentity(teacher);
            if (!passwordChange) {
                profileForm.elements.name.value = teacher.name;
                profileForm.elements.email.value = teacher.email;
                profileForm.elements.currentPassword.value = '';
                syncEmail();
                feedback(form, 'Профиль сохранён.')
            } else {
                form.reset();
                feedback(form, 'Пароль изменён. На этом устройстве вход сохранён.')
            }
        } catch (e) {
            feedback(form, e.message, true)
        } finally {
            button.disabled = false
        }
    };
    profileForm.onsubmit = e => {
        e.preventDefault();
        save(profileForm, false)
    };
    passwordForm.onsubmit = e => {
        e.preventDefault();
        save(passwordForm, true)
    };
};
