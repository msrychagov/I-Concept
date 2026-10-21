// Presentation only: preserve the existing navigation and form event handlers.
(() => {
    const toggle = document.querySelector('.menu-toggle');
    const sidebar = document.getElementById('main-sidebar');
    if (!toggle || !sidebar) return;
    const backdrop = document.createElement('button');
    backdrop.className = 'nav-backdrop';
    backdrop.type = 'button';
    backdrop.setAttribute('aria-label', 'Закрыть меню');
    backdrop.hidden = true;
    document.body.append(backdrop);
    const setOpen = (open, restoreFocus = false) => {
        document.body.classList.toggle('nav-open', open);
        toggle.setAttribute('aria-expanded', String(open));
        toggle.setAttribute('aria-label', open ? 'Закрыть меню' : 'Открыть меню');
        backdrop.hidden = !open;
        if (open) sidebar.querySelector('button.active, a.active, a, button')?.focus();
        else if (restoreFocus) toggle.focus();
    };
    toggle.addEventListener('click', () => setOpen(!document.body.classList.contains('nav-open')));
    backdrop.addEventListener('click', () => setOpen(false, true));
    sidebar.addEventListener('click', event => {
        if (event.target.closest('button, a')) setOpen(false);
    });
    document.addEventListener('keydown', event => {
        if (!document.body.classList.contains('nav-open')) return;
        if (event.key === 'Escape') setOpen(false, true);
        if (event.key === 'Tab') {
            const items = [...sidebar.querySelectorAll('a, button:not([disabled])')];
            const first = items[0], last = items[items.length - 1];
            if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
            else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
        }
    });
    window.matchMedia('(min-width: 741px)').addEventListener('change', event => {
        if (event.matches) setOpen(false);
    });
})();
