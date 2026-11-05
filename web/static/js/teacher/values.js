// Values by «Онтограф ценностей 1.1» (decree № 809): the marks of the model
// with word for word quotes, counted by the program. No levels and no common
// score: how many times and where. One text (an essay or the open answers of a
// test) — mount; many texts (an assignment of an essay, a platoon) — summary.
window.valuesBlock = (() => {
    const NOTE = 'Отметки модели по «Онтографу ценностей 1.1» (указ № 809), считает программа. Не уровень и не оценка: сколько раз и где об этом сказано. Считаются собственные слова автора и то, что он одобряет у других.';
    const PERIOD = 3000;
    const waiting = view => view.status === 'queued' || view.status === 'working';
    const capital = text => text ? text[0].toUpperCase() + text.slice(1) : text;

    // «2 · отвергает 1 · +1 в повседневной жизни»: the main count first.
    function numbers(c) {
        const parts = [String(c.accept)];
        if (c.reject) parts.push(`отвергает ${c.reject}`);
        if (c.everydayAccept) parts.push(`+${c.everydayAccept} в повседневной жизни`);
        if (c.everydayReject) parts.push(`отвергает в повседневной жизни ${c.everydayReject}`);
        return parts.join(' · ');
    }

    // A quote with what it is marked as; in the window of an essay a click opens
    // the whole answer (data-quote), in a test it stays as it is.
    function quote(q, esc, clickable) {
        const about = `${capital(q.label)} · ${q.whose}${q.sign === 'отвергает' ? ' · отвергает' : ''}${q.asked ? ' · ' + q.asked : ''}`;
        const inner = `<span>«${esc(q.quote)}»</span><small>${esc(about)}</small>`;
        return clickable ? `<button type="button" class="quote-chip" data-quote="${esc(q.question)}" data-source="current">${inner}</button>` : `<span class="quote-chip static">${inner}</span>`;
    }
    const quotes = (list, esc, clickable, label = 'Цитаты') => `<details class="essay-why"><summary>${esc(label)}</summary><div class="essay-why-body quote-list">${list.map(q => quote(q, esc, clickable)).join('')}</div></details>`;

    function count(c, esc, clickable) {
        if (!c.marks) return '<p class="muted">В тексте ценностей не отмечено.</p>';
        const met = c.ideas.filter(i => i.values.length);
        const absent = c.ideas.filter(i => !i.values.length).map(i => i.label);
        const ideas = met.map(i => `<div class="values-idea"><h4>${esc(i.label)} — ${esc(numbers(i))}</h4><ul class="values-list">${i.values.map(v => `<li><b>${esc(v.label)}</b> — ${esc(numbers(v))}${quotes(v.quotes, esc, clickable)}</li>`).join('')}</ul></div>`).join('');
        const anti = c.antivalues.length ? `<div class="values-idea"><h4>Антиценности</h4><ul class="values-list">${c.antivalues.map(v => {
            const parts = [];
            if (v.reject + v.everydayReject) parts.push(`осуждает ${v.reject + v.everydayReject}`);
            if (v.accept + v.everydayAccept) parts.push(`принимает как норму ${v.accept + v.everydayAccept}`);
            return `<li><b>${esc(v.label)}</b> <small class="muted">${esc(v.ideaLabel)}</small> — ${esc(parts.join(' · '))}${quotes(v.quotes, esc, clickable)}</li>`;
        }).join('')}</ul></div>` : '';
        const other = c.other.length ? quotes(c.other, esc, clickable, `Не считаются (чужие слова без своего отношения): ${c.other.length}`) : '';
        return `${ideas || '<p class="muted">Ценностей в собственных словах автора не отмечено.</p>'}${absent.length && met.length ? `<p class="muted">Не встретились: ${esc(absent.join(', '))}.</p>` : ''}${anti}${other}`;
    }

    function status(view, esc) {
        if (waiting(view)) return '<p class="portrait-status" role="status">Разметка готовится…</p>';
        if (view.status === 'failed') return `<p class="portrait-status" role="alert"><b>Не удалось разметить текст.</b> ${view.reason ? `<span class="muted">${esc(view.reason)}</span> ` : ''}<button type="button" class="secondary" data-values-retry>Повторить</button></p>`;
        if (view.status === 'off') return '<p class="portrait-status muted">Языковая модель не подключена.</p>';
        return '';
    }

    // Mounts the block of one text and returns the function that stops its
    // polling. A text with nothing to mark (404) leaves the element empty.
    function mount({element, api, esc, id, title, clickable = false}) {
        let timer, stopped = false;
        const ask = async method => {
            if (stopped || !element.isConnected) return;
            let view;
            try {
                view = await api('/values/' + encodeURIComponent(id), method === 'GET' ? {} : {method});
            } catch (e) {
                if (e.status === 404) {
                    element.innerHTML = '';
                    element.hidden = true;
                    return;
                }
                view = {status: 'failed', reason: e.message || ''};
            }
            if (stopped || !element.isConnected) return;
            element.hidden = false;
            element.innerHTML = `<h3>${esc(title)}</h3><p class="muted values-note">${esc(NOTE)}</p>${status(view, esc)}${view.status === 'ready' && view.count ? count(view.count, esc, clickable) : ''}`;
            element.querySelector('[data-values-retry]')?.addEventListener('click', () => ask('POST'));
            if (waiting(view)) timer = setTimeout(() => ask('GET'), PERIOD);
        };
        ask('GET');
        return () => {
            stopped = true;
            clearTimeout(timer);
        };
    }

    // The values over many texts: at how many authors (students) each value is
    // met, by idea; a number opens the list of names, a name opens the text.
    function summary(data, esc, {noun, of}) {
        if (!data) return '';
        if (!data.marked) return `<p class="muted">Разметки пока нет: ${data.texts ? `готова у 0 из ${data.texts}` : 'текстов нет'}.</p>`;
        const who = list => `<ul class="essay-source-list">${list.map(a => `<li><button type="button" class="link" ${a.submissionId ? `data-essay="${esc(a.submissionId)}"` : `data-session="${esc(a.sessionId)}"`}>${esc(a.name)}</button> <small class="muted">взвод ${esc(a.platoon)}</small></li>`).join('')}</ul>`;
        const line = v => {
            const parts = [];
            if (v.authors.length) parts.push(`<details class="essay-why"><summary>у ${v.authors.length} из ${data.marked}</summary><div class="essay-why-body">${who(v.authors)}</div></details>`);
            if (v.everyday.length) parts.push(`<details class="essay-why"><summary>только в повседневной жизни — у ${v.everyday.length}</summary><div class="essay-why-body">${who(v.everyday)}</div></details>`);
            if (v.rejecting.length) parts.push(`<details class="essay-why"><summary>отвергают — ${v.rejecting.length}</summary><div class="essay-why-body">${who(v.rejecting)}</div></details>`);
            return `<li><b>${esc(v.label)}</b> ${parts.join(' ')}</li>`;
        };
        const ideas = data.ideas.filter(i => i.values.length).map(i => `<div class="values-idea"><h4>${esc(i.label)} <small class="muted">· у ${i.authors} из ${data.marked}</small></h4><ul class="values-list">${i.values.map(line).join('')}</ul></div>`).join('');
        const anti = data.antivalues.length ? `<div class="values-idea"><h4>Антиценности</h4><ul class="values-list">${data.antivalues.map(v => `<li><b>${esc(v.label)}</b> <small class="muted">${esc(v.ideaLabel)}</small> ${v.rejecting.length ? `<details class="essay-why"><summary>осуждают — ${v.rejecting.length}</summary><div class="essay-why-body">${who(v.rejecting)}</div></details>` : ''} ${v.authors.length ? `<details class="essay-why"><summary>принимают как норму — ${v.authors.length}</summary><div class="essay-why-body">${who(v.authors)}</div></details>` : ''}</li>`).join('')}</ul></div>` : '';
        return `<p class="muted values-note">Разметка готова у ${data.marked} из ${data.texts} ${of}. У скольких ${noun} встречается ценность в их собственных словах или в том, что они одобряют у других; считает программа по отметкам модели. Не уровень и не оценка.</p>${ideas || `<p class="muted">Ценностей в собственных словах ${noun} не отмечено.</p>`}${anti}`;
    }

    return {mount, summary};
})();
