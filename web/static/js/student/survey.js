(() => {
    'use strict';
    const app = document.querySelector('#app');
    // part is the part of the test being answered; null on the list of parts.
    // A test without parts (older banks) is answered whole, as before.
    let data = null, mod = '', idx = 0, busy = false, partial = false, part = null, notice = '';
    const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[c]));
    const drafts = new Map();
    const sessionId = new URLSearchParams(location.search).get('id');

    async function api(path = '', method = 'GET', body) {
        try {
            return await window.apiRequest('/students/me/sessions/' + encodeURIComponent(sessionId) + path, {
                method,
                body
            })
        } catch (error) {
            partial = !!error.details?.partial;
            throw error
        }
    }

    function shuffle(a, seed) {
        let s = seed >>> 0;
        return a.map(x => {
            s = (s * 1664525 + 1013904223) >>> 0;
            return {x, r: s / 4294967296}
        }).sort((a, b) => a.r - b.r).map(v => v.x)
    }

    async function load() {
        try {
            data = await api();
            mod = data.items[0]?.module || 'A';
            render()
        } catch (e) {
            if (sessionId) app.innerHTML = `<main class="consent"><h1>Тестирование недоступно</h1><p class="error">${esc(e.message)}</p><a class="button" href="/student">Вернуться в кабинет</a></main>`; else location.replace('/student')
        }
    }

    function questions() {
        const seed = Array.from(mod).reduce((n, c) => Math.imul(n, 31) + c.charCodeAt(0), data.session.seed);
        return shuffle(data.items.filter(x => x.module === mod), seed)
    }

    // items are the questions being answered: of the open part, or all.
    function items() {
        const p = data.parts?.find(x => x.id === part);
        return p ? data.items.filter(i => p.modules.includes(i.module)) : data.items
    }

    // modules are the modules being answered, in the order of the part.
    function modules() {
        const p = data.parts?.find(x => x.id === part), present = [...new Set(items().map(i => i.module))];
        return p ? p.modules.filter(m => present.includes(m)) : present
    }

    function header(s, text) {
        return `<header class="student-header"><a href="/">Я-концепция / ВУЦ</a><span>${esc(s.participant)} · ${esc(s.group)} · ${esc(s.wave)}</span>${text ? `<span class="muted">${text}</span>` : ''}</header>`
    }

    // The list of the parts: each is submitted on its own, in any order.
    function renderParts() {
        const s = data.session, parts = data.parts, done = parts.filter(p => p.submitted).length;
        app.innerHTML = `<div class="student-layout">${header(s, `Сдано частей: ${done} из ${parts.length}`)}<main class="consent test-parts"><div class="eyebrow">Тестирование из ${parts.length} частей</div><h1>Выберите часть</h1><p>Части можно проходить в любом порядке и в разные дни. Ответы сохраняются сразу, начатую часть можно продолжить позже. Сданную часть изменить нельзя.</p>${notice ? `<p class="info" role="status">${esc(notice)}</p>` : ''}<div class="test-grid">${parts.map((p, n) => `<article class="test-card"><span class="status">${p.submitted ? 'Сдана' : p.answered ? 'В процессе' : 'Не начата'}</span><h3>Часть ${n + 1}. ${esc(p.name)}</h3><p class="muted">${p.total} вопросов${p.submitted ? '' : ` · отвечено ${p.answered}`}</p>${p.submitted ? '' : `<button data-part="${esc(p.id)}">${p.answered ? 'Продолжить' : 'Начать'}</button>`}</article>`).join('')}</div><div id="err"></div></main></div>`;
        notice = '';
        document.querySelectorAll('[data-part]').forEach(b => b.onclick = () => {
            part = b.dataset.part;
            mod = modules()[0];
            idx = 0;
            render()
        })
    }

    function optionText(i, v) {
        if (i.type === 'selfimage') return v;
        if (i.type === 'frequency') return ['Ни разу', 'Один раз', 'Два–три раза', 'Четыре–пять раз', 'Шесть раз и более'][v];
        return ['Совсем не соответствует', 'Скорее не соответствует', 'Отчасти соответствует', 'Скорее соответствует', 'Полностью соответствует'][v - 1]
    }

    function render() {
        const s = data.session, shown = items(), mods = modules(),
            answered = shown.filter(i => s.answers[i.id] !== undefined).length,
            current = data.parts?.find(x => x.id === part);
        if (!s.consent) {
            app.innerHTML = `<div class="student-layout"><header class="student-header"><a href="/">Я-концепция / ВУЦ</a><span>${esc(s.participant)} · ${esc(s.group)} · ${esc(s.wave)}</span></header><main class="consent"><div class="eyebrow">Перед началом</div><h1>${s.kind === 'observer' ? 'Педагогическое наблюдение' : 'Ваш опыт и восприятие среды'}</h1><p>Участие добровольное. Ответы будут связаны с вашим ФИО и номером взвода из личного кабинета, сохранены на сервере и доступны преподавателю.</p><p>Пилотная диагностика предназначена для педагогической работы, а не наказаний или кадровых решений. Можно оставить неприменимые вопросы без ответа.</p>${data.items.some(i=>i.responseOnly && i.type !== 'textarea')?'<p class="info">В тест входит отдельный раздел о психологическом благополучии. Его ответы также сохраняются с вашим именем и доступны преподавателю; они не используются для оценки воспитанности или пригодности. Вы можете отказаться от любого вопроса, выбрав «Предпочитаю не отвечать», либо оставить раздел без ответов.</p>':''}<button id="consent">Согласен(на) участвовать и сохранять ответы</button><div id="err"></div></main></div>`;
            document.querySelector('#consent').onclick = async () => {
                try {
                    const r = await api('/consent', 'PUT', {value: true, revision: s.revision});
                    Object.assign(s, r, {consent: true});
                    render()
                } catch (e) {
                    err(e.message)
                }
            };
            return
        }
        if (s.completed) {
            app.innerHTML = `<main class="consent"><div class="eyebrow">Сессия завершена</div><h1>Ответы сохранены</h1><p>${data.parts ? 'Все части тестирования сданы. ' : ''}Спасибо за участие.</p><p class="muted">Последнее сохранение: ${new Date(s.updated).toLocaleString('ru-RU')}</p></main>`;
            return
        }
        if (data.parts && !current) {
            renderParts();
            return
        }
        const q = questions();
        idx = Math.min(idx, q.length - 1);
        const i = q[idx], value = s.answers[i.id];
        let opts = i.options?.length ? i.options.map((o, k) => ({
            text: o.text,
            value: k,
            id: o.id
        })) : Array.from({length: i.max - i.min + 1}, (_, j) => ({
            text: optionText(i, j + i.min),
            value: j + i.min,
            id: j
        }));
        if (i.type === 'sjt') opts = shuffle(opts, s.seed + Array.from(i.id).reduce((n, c) => n + c.charCodeAt(0), 0));
        app.innerHTML = `<div class="student-layout">${header(s, `${current ? esc(current.name) + ': ' : ''}${answered} из ${shown.length}`)}<div class="progress-track"><div style="width:${answered / shown.length * 100}%"></div></div><div class="survey-workspace"><aside class="module-nav">${current ? '<button class="secondary" id="to-parts">← Все части</button>' : ''}<div class="eyebrow">Модули</div>${mods.map(m => `<button data-mod="${m}" class="${m === mod ? 'active' : ''}">${esc(data.modules[m])}<small>${shown.filter(x => x.module === m && s.answers[x.id] !== undefined).length} / ${shown.filter(x => x.module === m).length}</small></button>`).join('')}</aside><main class="question-surface"><div class="question-meta"><span>${esc(data.modules[mod])}</span><span>${idx + 1} / ${q.length}</span></div>${data.instructions?.[i.module] && i.type !== 'textarea' ? `<p class="info survey-instructions">${esc(data.instructions[i.module])}</p>` : ''}${i.timeWindow?`<p class="muted"><b>Период:</b> ${esc(i.timeWindow)}</p>`:''}${i.type === 'pair' ? `<h1 class="question-title pair-statements"><span><b>Первое</b>${esc(i.text)}</span><span><b>Второе</b>${esc(i.opposite)}</span></h1>` : `<h1 class="question-title">${esc(i.text)}</h1>`}${i.topic ? `<p class="muted">${esc(i.topic)}</p>` : ''}${i.hint ? `<p class="info activity-hint">${esc(i.hint)}</p>` : ''}${i.type === 'textarea' ? `<label>Ваш опыт и достижения<textarea id="activity-answer" rows="9" maxlength="4000" placeholder="Опишите участие, вашу роль и результаты…">${esc(drafts.get(i.id) ?? s.notes[i.id] ?? '')}</textarea></label><p class="muted">До 4000 символов. Если такого опыта нет, напишите «Не участвовал(а)».</p><button id="save-activity" type="button">Сохранить ответ</button>` : `<div class="answer-options ${i.type === 'selfimage' ? 'numeric' : ''}">${opts.map(o => `<label class="option ${value === o.value ? 'selected' : ''}"><input type="radio" name="answer" value="${o.value}" ${value === o.value ? 'checked' : ''}><span>${esc(o.text)}</span></label>`).join('')}</div>`}<button class="text-button" id="clear">Оставить без ответа / убрать оценку</button>${i.type !== 'textarea' ? `<label class="note-label">Комментарий или контекст (необязательно)<textarea id="note" maxlength="4000" rows="2">${esc(s.notes[i.id] || '')}</textarea></label>` : ''}<div id="err"></div><div class="question-footer"><button class="secondary" id="prev" ${idx === 0 ? 'disabled' : ''}>Назад</button><span id="status"></span>${idx < q.length - 1 ? '<button id="next">Далее</button>' : mods.indexOf(mod) < mods.length - 1 ? '<button id="nextmod">Следующий модуль</button>' : `<button id="finish">${current ? 'Сдать часть' : 'Завершить'}</button>`}</div></main></div></div>`;
        document.querySelectorAll('[data-mod]').forEach(b => b.onclick = () => navigate(() => {
            mod = b.dataset.mod;
            idx = 0;
            render()
        }));
        document.querySelectorAll('[name=answer]').forEach(x => x.onchange = () => save(i, Number(x.value)));
        document.querySelector('#clear').onclick = () => { if (i.type === 'textarea') drafts.set(i.id, ''); return save(i, null); };
        const textAnswer = document.querySelector('#activity-answer');
        if (textAnswer) {
            textAnswer.oninput = () => { drafts.set(i.id, textAnswer.value); document.querySelector('#status').textContent = 'Есть несохранённый ответ'; };
            document.querySelector('#save-activity').onclick = () => save(i, textAnswer.value.trim() ? 0 : null);
        }
        document.querySelector('#note')?.addEventListener('change', () => save(i, s.answers[i.id] ?? null));
        document.querySelector('#prev').onclick = () => navigate(() => {
            idx--;
            render()
        });
        document.querySelector('#next')?.addEventListener('click', () => navigate(() => {
            idx++;
            render()
        }));
        document.querySelector('#nextmod')?.addEventListener('click', () => navigate(() => {
            mod = mods[mods.indexOf(mod) + 1];
            idx = 0;
            render()
        }));
        document.querySelector('#finish')?.addEventListener('click', finish);
        document.querySelector('#to-parts')?.addEventListener('click', () => navigate(() => {
            part = null;
            render()
        }))
    }

    async function persistText() {
        const item = questions()[idx];
        if (item?.type !== 'textarea' || !drafts.has(item.id)) return true;
        return save(item, drafts.get(item.id).trim() ? 0 : null);
    }

    async function navigate(action) {
        if (busy || !await persistText()) return;
        action();
    }

    async function save(i, value) {
        if (busy) return false;
        busy = true;
        const note = i.type === 'textarea' ? (value === null ? '' : document.querySelector('#activity-answer').value) : document.querySelector('#note').value;
        document.querySelectorAll('button,input,textarea').forEach(x => x.disabled = true);
        const status = document.querySelector('#status');
        status.textContent = 'Сохранение…';
        try {
            const r = await api('/answers/' + encodeURIComponent(i.id), 'PUT', {
                value,
                note,
                revision: data.session.revision
            });
            if (value === null) delete data.session.answers[i.id]; else data.session.answers[i.id] = value;
            data.session.notes[i.id] = note;
            drafts.delete(i.id);
            if (r.parts) data.parts = r.parts;
            delete r.parts;
            Object.assign(data.session, r);
            render();
            return true
        } catch (e) {
            render();
            err(e.message);
            return false
        } finally {
            busy = false
        }
    }

    // finish submits the open part (the whole test without parts); after the
    // part the student is back on the list of parts.
    async function complete(allowPartial) {
        const name = data.parts?.find(x => x.id === part)?.name;
        const r = await api('/completion', 'POST', {allowPartial, revision: data.session.revision, ...(part ? {part} : {})});
        if (r.parts) data.parts = r.parts;
        delete r.parts;
        Object.assign(data.session, r);
        if (part) {
            notice = r.completed ? '' : `Часть «${name}» сдана.`;
            part = null
        }
        render()
    }

    async function finish() {
        if (busy || !await persistText()) return;
        try {
            await complete(false)
        } catch (e) {
            err(e.message + (partial ? ' ' : ''));
            if (partial) {
                const d = document.querySelector('#err');
                d.insertAdjacentHTML('beforeend', `<button id="partial">${part ? 'Сдать часть с пропусками' : 'Завершить с пропусками'}</button>`);
                document.querySelector('#partial').onclick = async () => {
                    try {
                        await complete(true)
                    } catch (e) {
                        err(e.message)
                    }
                }
            }
        }
    }

    function err(t) {
        const x = document.querySelector('#err');
        if (x) x.innerHTML = `<p class="error">${esc(t)}</p>`
    }

    window.addEventListener('beforeunload', event => {
        if (drafts.size || busy) { event.preventDefault(); event.returnValue = ''; }
    });

    if (sessionId) load(); else location.replace('/student');
    const back = document.createElement('a');
    back.href = '/student';
    back.textContent = '← Личный кабинет студента';
    back.className = 'button secondary';
    back.style.margin = '1rem';
    document.body.prepend(back);
})();
