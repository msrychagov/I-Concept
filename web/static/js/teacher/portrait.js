// The text a language model wrote for the teacher: the status, the signature
// and the polling are the same for every task. What a ready text looks like is
// drawn by window.portraitRenderers[task], which each task adds itself.
window.portraitRenderers = window.portraitRenderers || {};
window.portraitView = (() => {
    const SIGNATURE = 'Составлено автоматически по сведениям, которые сообщил студент';
    const PERIOD = 3000;
    const path = (task, id) => `/portraits/${encodeURIComponent(task)}/${encodeURIComponent(id)}`;
    const waiting = view => view.status === 'queued' || view.status === 'working';

    // Nothing for a ready text: it speaks for itself.
    function status(view, esc) {
        if (waiting(view)) return '<p class="portrait-status" role="status">Вывод готовится…</p>';
        if (view.status === 'failed') return `<p class="portrait-status" role="alert"><b>Не удалось составить вывод.</b> ${view.reason ? `<span class="muted">${esc(view.reason)}</span> ` : ''}<button type="button" class="secondary" data-portrait-retry>Повторить</button></p>`;
        if (view.status === 'off') return '<p class="portrait-status muted">Языковая модель не подключена.</p>';
        return '';
    }

    function signature(view, esc) {
        if (view.status !== 'ready') return '';
        const date = view.portrait.created ? new Date(view.portrait.created).toLocaleDateString('ru-RU') : '';
        return `<p class="portrait-signature muted">${esc(SIGNATURE)}${date ? ` · ${esc(date)}` : ''}</p>`;
    }

    // Asks for the text and keeps asking every 3 seconds while it is being made
    // and alive() says the window is still open. update(view) draws every answer.
    // The first request is the given one: "Повторить" starts with a POST.
    function start(api, task, id, {update, alive}, first = 'GET') {
        let timer, stopped = false;
        const ask = async method => {
            if (stopped || !alive()) return;
            let view;
            try {
                view = await api(path(task, id), method === 'GET' ? {} : {method});
            } catch (e) {
                view = {status: 'failed', reason: e.message || ''};
            }
            if (stopped || !alive()) return;
            update(view);
            if (waiting(view)) timer = setTimeout(() => ask('GET'), PERIOD);
        };
        ask(first);
        return {
            stop() {
                stopped = true;
                clearTimeout(timer);
            }
        };
    }

    return {status, signature, start};
})();
