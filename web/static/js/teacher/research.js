(() => {
    'use strict';
    const view = document.querySelector('#view');
    const messages = document.querySelector('#messages');
    let state, profile, editingProfileId = '', tab = 'overview', conceptTab = 'testing';
    const esc = value => String(value ?? '').replace(/[&<>"']/g, char => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[char]));
    const api = window.apiRequest;

    async function loadState() {
        const [templates, bank] = await Promise.all([api('/templates'), api('/question-bank')]);
        return {...templates, bank};
    }

    function notify(text) {
        messages.innerHTML = `<div class="info" role="status">${esc(text)}</div>`;
        scrollTo({top: 0, behavior: 'smooth'});
    }

    function profiles(editorView) {
        return window.templateEditor({
            view: editorView, state, profile, editingProfileId, api, esc,
            onDraft: draft => {
                profile = draft
            },
            onSelect: id => {
                editingProfileId = id;
                profile = structuredClone(id ? state.profiles.find(item => item.id === id) : state.emptyProfile);
                profiles(editorView);
            },
            onSaved: async () => {
                state = await loadState();
                editingProfileId = '';
                profile = structuredClone(state.emptyProfile);
                if (tab === 'profiles' && editorView.isConnected) {
                    profiles(editorView);
                    notify('Шаблон сохранён. Новые назначения будут использовать этот состав вопросов.');
                }
            }
        });
    }

    function concept() {
        view.innerHTML = '<header class="page-heading"><div><div class="eyebrow">Вид тестирования</div><h1>Я концепция</h1></div></header><div class="toolbar" aria-label="Разделы Я концепции"><button data-concept="testing">Назначения и результаты</button><button data-concept="templates">Шаблоны тестирования</button></div><div id="concept-content"></div>';
        const draw = () => {
            view.querySelectorAll('[data-concept]').forEach(b => {
                b.classList.toggle('secondary', b.dataset.concept !== conceptTab);
                b.setAttribute('aria-pressed', String(b.dataset.concept === conceptTab))
            });
            const slot = view.querySelector('#concept-content');
            slot.innerHTML = '<div></div>';
            const content = slot.firstElementChild;
            return conceptTab === 'templates' ? profiles(content) : window.teacherConceptTesting({
                view: content,
                state,
                api,
                esc
            });
        };
        view.querySelectorAll('[data-concept]').forEach(b => b.onclick = () => {
            conceptTab = b.dataset.concept;
            draw()
        });
        return draw();
    }

    function render() {
        messages.innerHTML = '';
        switch (tab) {
            case 'overview':
                return window.teachingDashboard({view, api, esc});
            case 'autobiography':
                return window.teacherAutobiography({view, api, esc});
            case 'profiles':
                return concept();
            case 'settings':
                return window.teacherSettings({view, esc});
            case 'platoons':
                return window.teacherPlatoons({view, api, esc});
            case 'model':
                return window.teacherMethodology({view, state, esc});
        }
    }

    document.querySelector('#nav').onclick = event => {
        const button = event.target.closest('[data-tab]');
        if (!button || !state) return;
        tab = button.dataset.tab;
        document.querySelectorAll('[data-tab]').forEach(item => item.classList.toggle('active', item === button));
        render();
    };
    document.querySelector('#teacher-logout').onclick = async () => {
        window.pageLoader.show('Выходим…');
        try {
            await window.logoutToHome();
        } catch (error) {
            messages.innerHTML = `<p class="error" role="alert">${esc(error.message)}</p>`;
            window.pageLoader.hide()
        }
    };

    async function load() {
        try {
            const [data, teacher] = await Promise.all([loadState(), window.teacherAccountAPI()]);
            state = data;
            profile = structuredClone(state.emptyProfile);
            window.showTeacherIdentity(teacher);
            await render();
        } catch (error) {
            view.innerHTML = `<h1>Не удалось загрузить кабинет</h1><p class="error">${esc(error.message)}</p><a href="/teacher">Войти заново</a>`
        } finally {
            window.pageLoader?.hide()
        }
    }

    load();
})();
