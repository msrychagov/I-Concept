window.apiRequest = async function (path, {method = 'GET', body} = {}) {
    const response = await fetch('/api/v1' + path, {
        method,
        credentials: 'same-origin',
        headers: body === undefined ? {} : {'Content-Type': 'application/json'},
        body: body === undefined ? undefined : JSON.stringify(body)
    });
    const data = await response.json();
    if (!response.ok) {
        const error = Error(data.error || 'Не удалось выполнить запрос.');
        error.status = response.status;
        error.details = data;
        throw error
    }
    return data;
};

window.logoutToHome = async function () {
    // Both roles may have an active cookie in the same browser.
    for (const role of ['teacher', 'student']) {
        await window.apiRequest('/auth/' + role + '/logout', {method: 'POST'});
    }
    window.location.replace('/');
};
