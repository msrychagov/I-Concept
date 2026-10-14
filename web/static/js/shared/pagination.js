// Shared navigation for independently paginated lists.
window.listPagination = function (container, {total, page, pageSize, onChange}) {
    const pages = Math.max(1, Math.ceil(total / pageSize));
    page = Math.max(1, Math.min(page, pages));
    const start = (page - 1) * pageSize, end = Math.min(start + pageSize, total);
    container.innerHTML = `<label class="page-size">На странице<select>${[10, 20, 50].map(n => `<option value="${n}" ${n === pageSize ? 'selected' : ''}>${n}</option>`).join('')}</select></label><span class="pagination-summary" role="status" tabindex="-1">${total ? `${start + 1}–${end} из ${total}` : '0 записей'}</span><div class="pagination-controls"><button type="button" class="secondary" data-page-prev ${page === 1 ? 'disabled' : ''} aria-label="Предыдущая страница">← Назад</button><span>Страница ${page} из ${pages}</span><button type="button" class="secondary" data-page-next ${page === pages ? 'disabled' : ''} aria-label="Следующая страница">Далее →</button></div>`;
    container.querySelector('select').onchange = e => {
        onChange(1, Number(e.target.value));
        container.querySelector('select').focus({preventScroll: true})
    };
    const turn = delta => {
        onChange(page + delta, pageSize);
        container.querySelector('.pagination-summary').focus({preventScroll: true});
        container.parentElement.scrollIntoView({block: 'start'})
    };
    container.querySelector('[data-page-prev]').onclick = () => turn(-1);
    container.querySelector('[data-page-next]').onclick = () => turn(1);
    return {page, start, end};
};
