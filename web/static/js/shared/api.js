(() => {
    let timer;
    function notify(message) {
        let notice = document.getElementById('connection-notice');
        if (!notice) {
            notice = document.createElement('aside');
            notice.id = 'connection-notice';
            notice.className = 'connection-notice';
            notice.setAttribute('role', 'alert');
            const text = document.createElement('span');
            const close = document.createElement('button');
            close.type = 'button';
            close.className = 'connection-notice-close';
            close.setAttribute('aria-label', 'Закрыть уведомление');
            close.textContent = '×';
            close.onclick = () => notice.remove();
            notice.append(text, close);
            document.body.append(notice);
        }
        notice.querySelector('span').textContent = message;
        clearTimeout(timer);
        timer = setTimeout(() => notice.remove(), 10000);
    }
    const offlineMessage = 'Потеряно интернет-соединение. Проверьте подключение и повторите действие.';
    window.addEventListener('offline', () => notify(offlineMessage));
    window.addEventListener('online', () => {
        document.getElementById('connection-notice')?.remove();
        clearTimeout(timer);
    });
    window.apiRequest = async function (path, {method = 'GET', body} = {}) {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), 20000);
        let response, data;
        try {
            response = await fetch('/api/v1' + path, {
                method,
                credentials: 'same-origin',
                headers: body === undefined ? {} : {'Content-Type': 'application/json'},
                body: body === undefined ? undefined : JSON.stringify(body),
                signal: controller.signal
            });
            // Proxy/server failures can return HTML or an empty body.
            const text = await response.text();
            try { data = text ? JSON.parse(text) : {}; }
            catch { data = null; }
        } catch (cause) {
            const message = navigator.onLine === false ? offlineMessage :
                cause.name === 'AbortError' ? 'Сервер не ответил вовремя. Проверьте соединение и повторите действие.' :
                'Не удалось связаться с сервером. Возможны проблемы с соединением или на стороне сервера. Повторите действие.';
            notify(message);
            throw new Error(message, {cause});
        } finally {
            clearTimeout(timeout);
        }
        if (response.status >= 500 || !data || typeof data !== 'object') {
            const message = 'Проблема на стороне сервера. Повторите действие позже.';
            notify(message);
            const error = new Error(message);
            error.status = response.status;
            throw error;
        }
        if (!response.ok) {
            const error = new Error(data.error || 'Не удалось выполнить запрос.');
            error.status = response.status;
            error.details = data;
            throw error;
        }
        return data;
    };
})();

window.logoutToHome = async function () {
    // Both roles may have an active cookie in the same browser.
    for (const role of ['teacher', 'student']) {
        await window.apiRequest('/auth/' + role + '/logout', {method: 'POST'});
    }
    window.location.replace('/');
};
