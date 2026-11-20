// Run with: node --test scripts/charts.test.cjs (no dependencies).
const {test} = require('node:test');
const assert = require('node:assert/strict');
const {readFileSync} = require('node:fs');
const {resolve} = require('node:path');
const vm = require('node:vm');
const context = vm.createContext({window: {}});
for (const file of ['charts', 'potential', 'test-portrait', 'values']) {
    vm.runInContext(readFileSync(resolve(__dirname, `../web/static/js/teacher/${file}.js`), 'utf8'), context);
}
const charts = context.window.diagnosticCharts;
const esc = x => String(x ?? '').replace(/[&<>"']/g, c => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[c]));

test('missing answers are not drawn as zero; a measured zero is drawn', () => {
    const html = charts.evidence([{name: 'Пропуск', statements: null}, {name: 'Ответ', statements: 1}], esc);
    assert.equal((html.match(/class="chart-track chart-missing"/g) || []).length, 1);
    assert.equal((html.match(/class="chart-fill /g) || []).length, 1);
    const climate = charts.climate([{side: 'Ответ', score: 0}, {side: 'Пропуск', score: null}], esc);
    assert.match(climate, /width:0%/);
    assert.match(climate, />0<\/strong>/);
    assert.match(climate, /Нет данных/);
    assert.doesNotMatch(climate + html, /NaN|undefined|Infinity/);
});

test('climate uses the signed scale and the mean for a platoon', () => {
    const html = charts.climate([{side: 'Один', score: 3, mean: -1.5}, {side: 'Два', score: null, mean: 3}], esc, true);
    assert.match(html, /left:25%;width:25%/);
    assert.match(html, /left:50%;width:50%/);
    assert.match(html, />-1,5<\/strong>/);
});

test('real and ideal are separate series; method scales keep their own ranges', () => {
    const html = charts.test({defender: [{name: 'Качества', real: 5, ideal: 10}], methods: [{name: 'Первая', score: 5, min: 0, max: 10}, {name: 'Вторая', score: 50, min: 0, max: 100}]}, esc);
    assert.match(html, /Я-реальное/);
    assert.match(html, /Я-идеальное/);
    assert.equal((html.match(/width:50%/g) || []).length, 3);
    assert.match(html, /шкала 0–10/);
    assert.match(html, /шкала 0–100/);
});

test('potential stays categorical and directions without experience stay folded', () => {
    const html = charts.potential({areas: [{title: 'Спорт', level: 'high', levelText: 'Высокий'}, {title: 'Наука', level: 'none', levelText: 'Не выражен'}]}, esc);
    assert.match(html, /Спорт/);
    assert.doesNotMatch(html, /Наука|width:|\/ 4|75%/);
    assert.match(charts.potential({areas: []}, esc), /Направлений с опытом пока нет/);
});

test('distribution opens the same student group and retains selection', () => {
    const rows = [{id: 'sport', title: 'Спорт', levels: [{level: 'high', levelText: 'Высокий', students: [{}, {}]}, {level: 'none', levelText: 'Не выражен', students: []}]}];
    const html = charts.distribution(rows, 4, esc, {area: 'sport', level: 'high'}, 'potential');
    assert.match(html, /width:50%/);
    assert.match(html, /data-area="sport" data-level="high" aria-pressed="true"/);
    assert.equal((html.match(/<button /g) || []).length, 1);
    const methods = charts.distribution([{id: 'will', kind: 'method', name: 'Воля', levels: rows[0].levels}], 4, esc, null, 'methods');
    assert.match(methods, /data-row="method:will"/);
    assert.equal(charts.distribution(rows, 0, esc, null, 'potential'), '');
});

test('value bars use absolute counts; author coverage uses marked texts', () => {
    const html = charts.values({ideas: [{label: 'Первая', accept: 2, reject: 2, everydayAccept: 0, everydayReject: 0}, {label: 'Вторая', accept: 1}]}, esc);
    assert.match(html, /width:25%/);
    assert.match(html, /отвергает: 2/);
    const authors = charts.valueAuthors({texts: 20, marked: 4, ideas: [{label: 'Идея', authors: 2}]}, esc);
    assert.match(authors, /width:50%/);
    assert.match(authors, /из 4 с готовой разметкой/);
    assert.doesNotMatch(authors, /width:10%/);
});

test('labels and button attributes escape user supplied HTML', () => {
    const attack = '\"><img src=x onerror=alert(1)>';
    const html = charts.distribution([{id: attack, title: attack, levels: [{level: 'high', levelText: attack, students: [{}]}]}], 1, esc, null, 'potential');
    assert.doesNotMatch(html, /<img/);
    assert.match(html, /&lt;img/);
});

test('essay processing states partition assigned essays', () => {
    const html = charts.essayProgress({submitted: 6, notSubmitted: 2, analysed: 4}, esc);
    assert.match(html, /назначено 8/);
    assert.equal((html.match(/width:25%/g) || []).length, 2);
    assert.match(html, /width:50%/);
    assert.equal(charts.essayProgress({submitted: 0, notSubmitted: 0, analysed: 0}, esc), '');
});
