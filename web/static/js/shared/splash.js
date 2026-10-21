/* Cosmetic first-open animation: no changes to authentication or page navigation. */
(() => {
    'use strict';
    const seenKey = 'vuc-intro-shown-v1';
    const reduceMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reduceMotion) return;
    const fullDuration = 3250;
    let remaining = fullDuration;
    try {
        const state = sessionStorage.getItem(seenKey);
        if (state === 'done') return;
        if (state && state.startsWith('pending:')) {
            remaining = Number(state.slice(8)) - Date.now();
            if (!(remaining > 0)) {
                sessionStorage.setItem(seenKey, 'done');
                return;
            }
        } else {
            sessionStorage.setItem(seenKey, `pending:${Date.now() + fullDuration}`);
        }
    } catch (_) { /* storage may be disabled; animation still works */ }
    const intro = document.createElement('div');
    intro.id = 'vuc-intro';
    intro.setAttribute('role', 'status');
    intro.setAttribute('aria-label', 'Открытие электронной диагностической системы');
    intro.innerHTML = `
        <div class="vuc-intro-content">
            <div class="vuc-intro-mark" aria-hidden="true">
                <span class="vuc-intro-halo"></span>
                <img class="vuc-intro-crest" src="/assets/images/reference/crest.png" alt="">
                <img class="vuc-intro-gild" src="/assets/images/reference/crest.png" alt="">
                <img class="vuc-intro-star" src="/assets/images/reference/crest.png" alt="">
                <span class="vuc-intro-spark s1"></span><span class="vuc-intro-spark s2"></span>
                <span class="vuc-intro-spark s3"></span><span class="vuc-intro-spark s4"></span>
            </div>
            <span class="vuc-intro-rule" aria-hidden="true"></span>
            <p class="vuc-intro-kicker">Военный учебный центр</p>
            <h2 class="vuc-intro-title">Электронная диагностическая система</h2>
            <p class="vuc-intro-subtitle">Воспитания студента ВУЦ</p>
        </div>
        <button class="vuc-intro-skip" type="button" aria-label="Пропустить заставку">Пропустить <span aria-hidden="true">→</span></button>`;
    document.body.prepend(intro);

    let closed = false;
    let finishTimer;
    let removalTimer;
    const dismiss = () => {
        if (closed) return;
        closed = true;
        try { sessionStorage.setItem(seenKey, 'done'); } catch (_) { /* harmless */ }
        clearTimeout(finishTimer);
        document.removeEventListener('keydown', onKeydown);
        intro.classList.add('vuc-intro-closing');
        removalTimer = setTimeout(() => intro.remove(), 670);
    };
    const onKeydown = event => { if (event.key === 'Escape') dismiss(); };
    intro.querySelector('.vuc-intro-skip').addEventListener('click', dismiss);
    document.addEventListener('keydown', onKeydown);
    finishTimer = setTimeout(dismiss, remaining);
    window.addEventListener('pagehide', () => {
        clearTimeout(finishTimer);
        clearTimeout(removalTimer);
        document.removeEventListener('keydown', onKeydown);
    }, {once: true});
})();
