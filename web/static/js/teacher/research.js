// The cabinet of the teacher without the side menu (decided 18 November 2026,
// К1–К6): the home page is cards, a card opens its section, a section with
// several parts shows them as tiles and opens the most used one. The path line
// on top and the browser button «Назад» go one step back.
(() => {
    'use strict';
    const view = document.querySelector('#view');
    const messages = document.querySelector('#messages');
    let state, profile, editingProfileId = '', tab = 'overview', tabOptions = {}, peopleNav = null, conceptDraw = null, current = null;
    const esc = value => String(value ?? '').replace(/[&<>"']/g, char => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[char]));
    const api = window.apiRequest;

    const TITLES = {
        overview: 'Главная',
        autobiography: 'Автобиография',
        essay: 'Эссе',
        profiles: 'Диагностическое тестирование',
        platoons: 'Управление взводом',
        progress: 'Прохождение',
        model: 'Модель и методика',
        settings: 'Настройки'
    };

    // The tiles inside a section (К6); `open` is the part shown when the section opens.
    const PARTS = {
        autobiography: {
            open: 'fill', list: [
                {id: 'assign', icon: 'send', title: 'Назначить', note: 'Всем взводам или одному'},
                {id: 'list', icon: 'clipboard', title: 'Назначения', note: 'Открыть и закрыть доступ'},
                {id: 'fill', icon: 'file', title: 'Заполнение', note: 'Кто сдал, анкеты и выводы'},
                {id: 'potential', icon: 'chart', title: 'Потенциал по взводу', note: 'Уровни по шести направлениям'}
            ]
        },
        profiles: {
            open: 'results', list: [
                {id: 'assign', icon: 'send', title: 'Назначить', note: 'Всем взводам или одному'},
                {id: 'list', icon: 'clipboard', title: 'Назначенные', note: 'Открыть и закрыть доступ'},
                {id: 'results', icon: 'user', title: 'Результаты студентов', note: 'Результат и вывод по каждому'},
                {id: 'levels', icon: 'chart', title: 'Уровни методик по взводу', note: 'Вывод по взводу, ценности'}
                // «Шаблоны тестирования» are hidden since 27 November 2026: the test
                // is assigned whole, by one button; the editor (profiles) is kept.
            ]
        },
        essay: {
            open: 'given', list: [
                {id: 'give', icon: 'send', title: 'Выдать эссе выпуску', note: 'Выпуск и название'},
                {id: 'given', icon: 'book', title: 'Выданные эссе', note: 'Авторы, сводка по выпуску, ценности'}
            ]
        },
        platoons: {
            open: 'members', list: [
                {id: 'platoons', icon: 'users', title: 'Взводы и приглашения', note: 'Создать, переименовать, ссылка студентам'},
                {id: 'members', icon: 'user', title: 'Состав', note: 'Перевести, удалить, отключить кабинет'},
                {id: 'cohorts', icon: 'graduation', title: 'Выпуски', note: 'Создать, изменить, удалить'}
            ]
        }
    };
    const parts = {};

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

    // The templates are their own page; the other parts are the page of the assignments and results.
    function concept(section) {
        section.innerHTML = '<header class="page-heading"><div><div class="eyebrow">Кабинет преподавателя</div><h1>Диагностическое тестирование</h1></div></header><div data-parts></div><div id="concept-content"></div>';
        const draw = () => {
            const slot = section.querySelector('#concept-content');
            if (!slot) return;
            slot.innerHTML = '<div></div>';
            const content = slot.firstElementChild;
            return parts.profiles === 'templates' ? profiles(content) : window.teacherConceptTesting({view: content, state, api, esc});
        };
        conceptDraw = draw;
        return draw();
    }

    // ---- the tiles of the parts ------------------------------------------
    function tilesHtml() {
        return `<div class="path-tiles part-tiles" role="group" aria-label="Части раздела">${PARTS[tab].list.map(p => `<button type="button" class="path-tile" data-section-part="${p.id}"><span class="metric-emblem"><img class="ui-icon" src="/assets/images/icons/${p.icon}.svg" alt="" width="26" height="26"></span><b>${esc(p.title)}</b><small>${esc(p.note)}</small></button>`).join('')}</div>`;
    }

    // Sections redraw themselves; after every redraw the tiles are put back and
    // only the blocks of the chosen part stay visible.
    function applyParts() {
        if (!PARTS[tab]) return;
        const part = parts[tab];
        view.querySelectorAll('[data-parts]').forEach(slot => {
            if (!slot.firstElementChild) slot.innerHTML = tilesHtml();
            slot.querySelectorAll('[data-section-part]').forEach(b => {
                const on = b.dataset.sectionPart === part;
                b.classList.toggle('active', on);
                b.setAttribute('aria-pressed', String(on));
            });
        });
        view.querySelectorAll('[data-part-of]').forEach(block => block.classList.toggle('part-off', block.dataset.partOf !== part));
    }

    new MutationObserver(applyParts).observe(view, {childList: true, subtree: true});

    function setPart(next, push = true) {
        const before = parts[tab];
        if (before === next) return;
        parts[tab] = next;
        if (tab === 'profiles' && (before === 'templates') !== (next === 'templates')) conceptDraw?.();
        applyParts();
        if (push) remember(here());
    }

    view.addEventListener('click', event => {
        const button = event.target.closest('[data-parts] [data-section-part]');
        if (button && view.contains(button)) setPart(button.dataset.sectionPart);
    });

    // ---- screens -----------------------------------------------------------
    function crumbs() {
        const nav = document.createElement('nav');
        nav.className = 'path-crumbs';
        nav.setAttribute('aria-label', 'Путь');
        nav.innerHTML = `<button type="button" class="text-button" data-home>Главная</button><span class="path-sep" aria-hidden="true">›</span><span aria-current="page">${esc(TITLES[tab])}</span>`;
        nav.querySelector('[data-home]').onclick = () => go('overview');
        return nav;
    }

    function render() {
        messages.innerHTML = '';
        peopleNav = null;
        const section = document.createElement('div');
        view.replaceChildren(...(tab === 'overview' || tab === 'people' ? [] : [crumbs()]), section);
        switch (tab) {
            case 'overview':
                return window.teachingDashboard({view: section, api, esc});
            case 'autobiography':
                return window.teacherAutobiography({view: section, api, esc});
            case 'essay':
                return window.teacherEssays({view: section, api, esc});
            case 'profiles':
                return concept(section);
            case 'settings':
                return window.teacherSettings({view: section, esc});
            case 'platoons':
                return window.teacherPlatoons({view: section, api, esc});
            case 'progress':
                return window.teachingProgress({view: section, api, esc});
            case 'model':
                return window.teacherMethodology({view: section, state, esc});
            case 'people':
                return window.teacherPlatoonView({
                    view: section, api, esc, start: tabOptions.place || tabOptions.start || 'platoons', home: () => go('overview'),
                    // Every step of the path is a step of the browser history.
                    onPlace: place => remember({tab: 'people', place}),
                    bind: nav => peopleNav = nav
                });
        }
    }

    const here = () => ({tab, part: parts[tab], place: tabOptions.place});

    // The step the browser history stands on now.
    function remember(step) {
        current = step;
        history.pushState(step, '');
    }

    // A section by its tab; it opens on its most used part.
    function go(next, options = {}, push = true) {
        if (!state) return;
        document.querySelectorAll('dialog[open]').forEach(dialog => dialog.close());
        tab = next;
        tabOptions = options;
        if (PARTS[next]) parts[next] = options.part || PARTS[next].open;
        if (next === 'people' && !options.place) tabOptions = {place: {screen: options.start || 'platoons'}};
        render();
        if (push) remember(here());
        else current = here();
        scrollTo({top: 0, behavior: 'smooth'});
    }

    window.teacherGo = go;

    // «Назад» with an open window closes the window and stays on the step.
    addEventListener('popstate', event => {
        const s = event.state;
        if (!state || !s?.tab) return;
        const open = document.querySelectorAll('dialog[open]');
        if (open.length) {
            open.forEach(dialog => dialog.close());
            if (current) history.pushState(current, '');
            return;
        }
        current = s;
        if (s.tab === 'people' && tab === 'people' && peopleNav && s.place) return peopleNav.go(s.place);
        if (s.tab === tab && PARTS[tab] && s.part) return setPart(s.part, false);
        go(s.tab, {part: s.part, place: s.place}, false);
    });

    document.querySelector('#cabinet-home').onclick = event => {
        event.preventDefault();
        go('overview');
    };

    // The profile menu lives in the teacher header only. Logout behavior remains unchanged.
    const userMenu = document.querySelector('.teacher-user-menu');
    const userTrigger = document.querySelector('#teacher-user-trigger');
    const userDropdown = document.querySelector('#teacher-user-dropdown');
    const setUserMenuOpen = (open, restoreFocus = false) => {
        userMenu.classList.toggle('is-open', open);
        userTrigger.setAttribute('aria-expanded', String(open));
        userDropdown.setAttribute('aria-hidden', String(!open));
        userDropdown.inert = !open;
        if (restoreFocus) userTrigger.focus();
    };
    userTrigger.addEventListener('click', () => {
        setUserMenuOpen(!userMenu.classList.contains('is-open'));
    });
    userTrigger.addEventListener('keydown', event => {
        if (event.key === 'ArrowDown') {
            event.preventDefault();
            setUserMenuOpen(true);
            userDropdown.querySelector('[role="menuitem"]')?.focus();
        }
    });
    document.addEventListener('pointerdown', event => {
        if (!userMenu.contains(event.target)) setUserMenuOpen(false);
    });
    document.addEventListener('keydown', event => {
        if (event.key === 'Escape' && userMenu.classList.contains('is-open')) {
            event.preventDefault();
            setUserMenuOpen(false, true);
        }
    });
    userMenu.addEventListener('focusout', event => {
        if (!userMenu.contains(event.relatedTarget)) setUserMenuOpen(false);
    });

    document.querySelector('#teacher-settings').onclick = () => {
        setUserMenuOpen(false);
        go('settings');
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
            current = here();
            history.replaceState(current, '');
            await render();
        } catch (error) {
            view.innerHTML = `<h1>Не удалось загрузить кабинет</h1><p class="error">${esc(error.message)}</p><a href="/teacher">Войти заново</a>`
        } finally {
            window.pageLoader?.hide()
        }
    }

    load();
})();
