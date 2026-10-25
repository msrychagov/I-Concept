// The autobiography for the personal file, as a sheet of paper: the program
// writes it from the answers of a submitted autobiography. The teacher opens
// it from the answers of a student (?id=…), the student from the cabinet
// (?id=…&role=student). Printing leaves only the sheet.
(() => {
    const root = document.getElementById('sheet-root');
    const params = new URLSearchParams(location.search);
    const student = params.get('role') === 'student';
    const id = params.get('id') || '';
    const back = student ? '/student' : '/research';
    const esc = x => String(x ?? '').replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    })[c]);

    function draw(d) {
        document.title = `Автобиография — ${d.student} · ВУЦ`;
        // «г.», «ул.», «№» and the like stay on one line with what follows them.
        const glue = text => text.replace(/(^|[\s(«])(г\.|ул\.|д\.|кв\.|обл\.|с\.|пос\.|пр\.|№)\s+/g, '$1$2\u00a0');
        // A paragraph that ends with a colon introduces the next ones (the relatives).
        const paragraphs = d.paragraphs.map(p => `<p${p.endsWith(':') ? ' class="lead-in"' : ''}>${esc(glue(p))}</p>`).join('');
        root.innerHTML = `<nav class="sheet-tools" aria-label="Действия"><a href="${back}">← В кабинет</a><button type="button" id="print-sheet">Печать</button></nav>
<article class="sheet" aria-label="Автобиография"><h1>Автобиография</h1>${paragraphs}<footer class="sheet-sign"><span>${esc(d.date)}</span><span class="sign-line">(подпись)</span><span>${esc(d.surname)}</span></footer></article>`;
        root.querySelector('#print-sheet').onclick = () => window.print();
    }

    window.apiRequest((student ? '/students/me/submissions/' : '/submissions/') + encodeURIComponent(id) + '/autobiography')
        .then(draw)
        .catch(e => {
            root.innerHTML = `<p class="sheet-status error" role="alert">${esc(e.message)}</p><p class="sheet-status"><a href="${back}">В кабинет</a></p>`;
        });
})();
