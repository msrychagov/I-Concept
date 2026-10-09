window.teacherResult=async function({view,api,esc,id}){
 const dialog=view.querySelector('#result-dialog');
 const fmt=value=>value==null?'Нет данных':Number(value).toFixed(2).replace('.',',');
 let data,page=1,pageSize=10;
 dialog.innerHTML='<h2>Загрузка результата…</h2>';dialog.showModal();
 async function load(){
  data=await api('/sessions/'+encodeURIComponent(id)+'/report');
 }
 function draw(){
  const s=data.session,r=data.report;
  const rows=r.evidence||[];
  const answered=rows.reduce((sum,row)=>sum+row.coverage,0);
  const total=rows.reduce((sum,row)=>sum+row.total,0);
  const statements=data.items.filter(item=>item.scored||['B','C','D'].includes(item.module));
  dialog.innerHTML=`<div class="section-title"><div><h2>${esc(s.participant)}</h2><p>Взвод ${esc(s.group)} · ${esc(s.wave)}</p></div><button class="secondary" id="close-result">Закрыть</button></div>
   <p>${s.completed?'Прохождение завершено':'Прохождение в процессе'} · ответов на утверждения ${answered} из ${total}</p>
   <h3>Отчёт по утверждениям</h3><p class="muted">Средний балл по показателю рассчитывается по шкале 1–5 с учётом обратных утверждений. При неполном наборе ответов балл не рассчитывается; пропуски не заменяются нулём.</p>
   <section><div id="evidence-list"></div><nav class="list-pagination" id="evidence-pagination" aria-label="Страницы диагностических показателей"></nav></section>
   <details><summary>Ответы на утверждения и комментарии</summary>${statements.map(item=>`<article class="bank-item"><b>${esc(item.text)}</b><p>${s.answers[item.id]===undefined?'Нет ответа':esc(item.options?.[s.answers[item.id]]?.text??s.answers[item.id])}</p><small>${esc(s.notes[item.id]||'')}</small></article>`).join('')}</details>
   <a class="button secondary" href="/api/v1/sessions/${encodeURIComponent(id)}/export">Экспорт сессии</a>`;
  dialog.querySelector('#close-result').onclick=()=>dialog.close();
  list();

 }
 function list(){
  const rows=data.report.evidence||[];
  const range=window.listPagination(dialog.querySelector('#evidence-pagination'),{total:rows.length,page,pageSize,onChange:(next,size)=>{page=next;pageSize=size;list()}});page=range.page;
  dialog.querySelector('#evidence-list').innerHTML=rows.slice(range.start,range.end).map(row=>`<details class="evidence-card"><summary><b>${esc(row.id)} · ${esc(row.name)}</b><span>${row.statements==null?'Недостаточно ответов':fmt(row.statements)+' / 5'} · ${row.coverage}/${row.total} пунктов</span></summary>
   <p>${esc(row.definition)}</p><p class="muted">${esc(row.limitations)}</p>
   <p><b>Средний балл:</b> ${row.statements==null?'Недостаточно ответов':fmt(row.statements)+' из 5'}. <b>Ответов:</b> ${row.coverage} из ${row.total}.</p>
   </details>`).join('');
 }
 try{await load();draw()}catch(error){dialog.innerHTML=`<p class="error">${esc(error.message)}</p><button id="close-result">Закрыть</button>`;dialog.querySelector('#close-result').onclick=()=>dialog.close()}
};
