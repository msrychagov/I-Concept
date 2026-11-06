// The dates of the other passings of one student at the top of a window
// (decided 6 November 2026): the newest opens first, a date opens that one.
// slices: [{id, label}], newest first; with fewer than two nothing is drawn.
window.sliceBar = (slices, current, esc) => {
    if (!slices || slices.length < 2) return '';
    return `<nav class="slice-bar" aria-label="Другие прохождения"><span class="muted">Прохождения:</span>${slices.map(s => `<button type="button" class="${s.id === current ? '' : 'secondary'}" data-slice="${esc(s.id)}" ${s.id === current ? 'aria-current="true" disabled' : ''}>${esc(s.label)}</button>`).join('')}</nav>`;
};
window.sliceBar.bind = (root, slices, open) => {
    if (!slices || slices.length < 2) return;
    root.querySelectorAll('[data-slice]').forEach(b => b.onclick = () => open(b.dataset.slice));
};
