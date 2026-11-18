const app = document.getElementById('app');
const K = ['a', 'b', 'c', 'd'];

async function api(path, opts = {}) {
  const r = await fetch(path, { headers: { 'Content-Type': 'application/json' }, ...opts });
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}
const esc = s => String(s ?? '').replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const fmt = s => esc(s).replace(/`([^`]+)`/g, '<code>$1</code>').replace(/\*\*([^*]+)\*\*/g, '<b>$1</b>').replace(/\n/g, '<br>');
const mmss = s => { s = Math.max(0, Math.round(s)); const h = Math.floor(s / 3600), m = Math.floor(s % 3600 / 60), x = s % 60; return (h ? h + ':' + String(m).padStart(2, '0') : m) + ':' + String(x).padStart(2, '0'); };
const when = t => new Date(t).toLocaleString('en-IN', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
const pct = (a, b) => b ? Math.round(a * 1000 / b) / 10 : 0;
const barClass = p => p >= 80 ? '' : p >= 60 ? 'mid' : 'low';

function questionBody(q) {
  return (q.passage ? `<div class="passage">${fmt(q.passage)}</div>` : '') +
    `<div class="stem">${fmt(q.question)}</div>` + (q.code ? `<pre>${esc(q.code)}</pre>` : '');
}

let cleanup = null;
async function route() {
  if (cleanup) { const c = cleanup; cleanup = null; c(); }
  const h = location.hash.slice(1) || '/';
  const [path, qs] = h.split('?');
  const p = path.split('/').filter(Boolean);
  window.scrollTo(0, 0);
  try {
    if (p[0] === 'exam') await exam(p[1]);
    else if (p[0] === 'result') await result(p[1]);
    else if (p[0] === 'practice') await practice(qs || '');
    else if (p[0] === 'stats') await stats();
    else await home();
  } catch (e) {
    app.innerHTML = `<div class="card"><b>Something went wrong.</b><pre>${esc(e.message)}</pre><a href="#/">Home</a></div>`;
  }
}
window.addEventListener('hashchange', route);
route();

/* ---------------- home ---------------- */
async function home() {
  const [{ banks, areas }, exams] = await Promise.all([api('/api/banks'), api('/api/exams')]);
  const open = exams.filter(e => !e.Submitted);
  const done = exams.filter(e => e.Submitted);
  app.innerHTML = `
  <h1>Mock tests</h1>
  ${open.length ? `<div class="card"><b>Not submitted</b>${open.map(e => `
    <div class="row" style="margin-top:6px"><span>${esc(e.Title)} (${e.Mode === 'ab' ? 'Parts A and B' : 'full paper'}), started ${when(e.Started)}</span>
    <button class="dark" data-resume="${e.ID}">Resume</button></div>`).join('')}</div>` : ''}
  <div class="grid2">${banks.map(b => `
    <div class="card"><h2 style="margin-top:0">${esc(b.Title)}</h2>
      <p class="muted small">Part A ${b.PartA} questions, Part B ${b.PartB} questions. +1 for a correct answer, −0.25 for a wrong one.</p>
      <div class="row">
        <button class="primary" data-start="${b.ID}" data-mode="full">Full paper, ${b.Minutes} min</button>
        <button data-start="${b.ID}" data-mode="ab">Parts A and B only, ${b.Minutes - 35} min</button>
      </div>
      <p class="muted small">Full paper: the timer includes Part C, which you write by hand on paper, as in the real exam.</p>
    </div>`).join('')}</div>

  <h2>Past exams</h2>
  ${done.length ? `<table><tr><th>Date</th><th>Test</th><th>Mode</th><th class="num">Part A</th><th class="num">Part B</th><th class="num">Total</th><th></th></tr>
    ${done.map(e => `<tr><td>${when(e.Submitted)}</td><td>${esc(e.Title)}</td><td>${e.Mode === 'ab' ? 'A and B' : 'Full'}</td>
      <td class="num">${e.A.score.toFixed(2)} / 40</td><td class="num">${e.B.score.toFixed(2)} / 60</td><td class="num"><b>${e.Score.toFixed(2)}</b> / 100</td>
      <td><a href="#/result/${e.ID}">Review</a></td></tr>`).join('')}</table>` : '<p class="muted">No exam taken yet.</p>'}

  <h2>Practice by topic</h2>
  <div class="card topics">
    <p class="muted small" style="margin-top:0">Questions come from all the banks. Each answer is checked at once and shown with its explanation.</p>
    ${areas.map(a => `<fieldset><legend><label><input type="checkbox" data-area="${esc(a.name)}"> ${esc(a.name)}</label></legend>
      ${a.topics.map(t => `<label><input type="checkbox" name="topic" value="${esc(t)}" data-in="${esc(a.name)}"> ${esc(t)}</label>`).join('')}</fieldset>`).join('')}
    <div class="row">
      <select id="pmode"><option value="all">All questions</option><option value="new">Only questions I have not answered</option><option value="mistakes">Only my open mistakes</option></select>
      <select id="pn"><option>10</option><option selected>20</option><option>40</option><option value="1000">All</option></select>
      <button class="primary" id="pgo">Start practice</button>
      <span class="muted small">No topic ticked means every topic.</span>
    </div>
  </div>`;
  app.querySelectorAll('[data-start]').forEach(b => b.onclick = async () => {
    b.disabled = true;
    const r = await api('/api/exams', { method: 'POST', body: JSON.stringify({ bank: b.dataset.start, mode: b.dataset.mode }) });
    location.hash = '#/exam/' + r.id;
  });
  app.querySelectorAll('[data-resume]').forEach(b => b.onclick = () => location.hash = '#/exam/' + b.dataset.resume);
  app.querySelectorAll('[data-area]').forEach(c => c.onchange = () =>
    app.querySelectorAll(`[data-in="${CSS.escape(c.dataset.area)}"]`).forEach(x => x.checked = c.checked));
  document.getElementById('pgo').onclick = () => {
    const topics = [...app.querySelectorAll('input[name=topic]:checked')].map(x => x.value);
    location.hash = '#/practice?' + new URLSearchParams({ topics: topics.join(','), mode: document.getElementById('pmode').value, n: document.getElementById('pn').value });
  };
}

/* ---------------- exam ---------------- */
async function exam(id) {
  const d = await api('/api/exams/' + id);
  const at = d.attempt;
  if (at.submitted) { location.hash = '#/result/' + id; return; }
  const qs = d.questions, partc = d.partc || [];
  const answers = at.answers || {};
  const events = at.events || [];
  let partcSec = at.partc_seconds || 0;
  const t0 = new Date(at.started).getTime();
  const skew = Date.now() - new Date(d.now).getTime();
  const deadline = t0 + at.minutes * 60000;
  const serverNow = () => Date.now() - skew;
  const elapsed = () => (serverNow() - t0) / 1000;
  const R = q => answers[q.id] || (answers[q.id] = { choice: '', marked: false, visits: 0, seconds: 0, history: [] });
  let cur = 0, sec = 'A', shownAt = Date.now(), focused = document.hasFocus(), dirty = false, submitting = false;
  R(qs[0]).visits = Math.max(1, R(qs[0]).visits);

  function flushTime() {
    const now = Date.now();
    if (focused) {
      const s = (now - shownAt) / 1000;
      if (sec === 'C') partcSec += s; else R(qs[cur]).seconds += s;
    }
    shownAt = now;
  }
  async function save(keepalive) {
    flushTime();
    dirty = false;
    try {
      await fetch('/api/exams/' + id, { method: 'PUT', keepalive, headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ answers, events, partc_seconds: partcSec }) });
    } catch (e) { dirty = true; }
  }
  const touch = () => { dirty = true; clearTimeout(touch.t); touch.t = setTimeout(() => save(), 1200); };
  function state(q) {
    const r = answers[q.id];
    if (!r || !r.visits) return 'nv';
    if (r.marked) return r.choice ? 'am' : 'mk';
    return r.choice ? 'an' : 'na';
  }
  function go(i) {
    flushTime();
    cur = Math.max(0, Math.min(qs.length - 1, i));
    sec = qs[cur].part;
    R(qs[cur]).visits++;
    touch();
    render();
  }
  function counts(list) {
    const c = { nv: 0, na: 0, an: 0, mk: 0, am: 0 };
    list.forEach(q => c[state(q)]++);
    return c;
  }
  async function submit(auto) {
    if (submitting) return;
    if (!auto) {
      const c = counts(qs);
      inDialog = true;
      const ok = confirm(`Submit the exam?\n\nAnswered: ${c.an + c.am} (marked for review among them: ${c.am})\nNot answered: ${c.na + c.mk}\nNot visited: ${c.nv}\n\nAnswered questions that are marked for review are scored.`);
      setTimeout(() => { inDialog = false; }, 500);
      if (!ok) return;
    }
    submitting = true;
    flushTime();
    await fetch('/api/exams/' + id + '/submit', { method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ answers, events, partc_seconds: partcSec }) });
    location.hash = '#/result/' + id;
  }

  let inDialog = false;
  const onBlur = () => { if (inDialog || submitting) return; flushTime(); focused = false; events.push({ t: elapsed(), k: 'blur' }); touch(); render(); };
  const onFocus = () => { shownAt = Date.now(); focused = true; events.push({ t: elapsed(), k: 'focus' }); };
  const onUnload = () => save(true);
  window.addEventListener('blur', onBlur);
  window.addEventListener('focus', onFocus);
  window.addEventListener('beforeunload', onUnload);
  const tick = setInterval(() => {
    const left = (deadline - serverNow()) / 1000;
    const el = document.getElementById('timer');
    if (el) { el.textContent = mmss(left); el.classList.toggle('low', left < 300); }
    if (left <= 0) submit(true);
  }, 500);
  const autosave = setInterval(() => { if (dirty || sec === 'C') save(); }, 15000);
  cleanup = () => {
    clearInterval(tick); clearInterval(autosave);
    window.removeEventListener('blur', onBlur); window.removeEventListener('focus', onFocus); window.removeEventListener('beforeunload', onUnload);
    if (!submitting) save(true);
  };

  function render() {
    const parts = ['A', 'B'].concat(partc.length ? ['C'] : []);
    const inSec = qs.map((q, i) => [q, i]).filter(([q]) => q.part === sec);
    const blurs = events.filter(e => e.k === 'blur').length;
    let pane;
    if (sec === 'C') {
      pane = `<div class="qhead"><span>Part C: descriptive</span><span class="muted small">write on paper</span></div>
        <p>Write these answers <b>by hand on paper</b>, as in the real exam. When you finish, photograph the pages and send them to Claude for marking.</p>
        ${partc.map(p => `<div class="card"><b>${esc(p.id)}. ${fmt(p.title)}</b><p>${fmt(p.prompt)}</p><p class="muted small">Target about ${p.words} words.</p></div>`).join('')}`;
    } else {
      const q = qs[cur], r = R(q);
      pane = `<div class="qhead"><span>Question ${cur + 1}</span><span class="muted small">Part ${q.part} · +1 / −0.25</span></div>
        ${questionBody(q)}
        ${q.options.map((o, k) => `<div class="opt ${r.choice === K[k] ? 'sel' : ''}" data-k="${K[k]}"><span class="k">(${K[k]})</span><span>${fmt(o)}</span></div>`).join('')}
        <div class="actions">
          <button id="mark">Mark for review &amp; next</button><button id="clear">Clear response</button><span class="grow"></span>
          <button id="prev" ${cur === 0 ? 'disabled' : ''}>Previous</button><button class="primary" id="next">Save &amp; next</button>
        </div>`;
    }
    const c = counts(inSec.map(([q]) => q));
    app.innerHTML = `
      <div class="exambar"><div><b>${esc(d.title)}</b> <span class="muted small">${at.mode === 'ab' ? 'Parts A and B' : 'Full paper'}</span></div>
        <div>Time left <span id="timer" class="timer">${mmss((deadline - serverNow()) / 1000)}</span></div></div>
      ${blurs ? `<div class="blur-note">You have left the exam window ${blurs} time${blurs > 1 ? 's' : ''}. In the real exam this can get you flagged.</div>` : ''}
      <div class="tabs">${parts.map(p => `<button data-sec="${p}" class="${p === sec ? 'on' : ''}">Part ${p}${p === 'C' ? ' (paper)' : ` (${qs.filter(q => q.part === p).length})`}</button>`).join('')}</div>
      <div class="exam">
        <div class="card">${pane}</div>
        <div class="card">
          ${sec === 'C' ? '<p class="muted small">Part C has no answers to mark here.</p>' : `
          <div class="legend">
            <div><span class="b" style="background:var(--green)"></span>${c.an} answered</div>
            <div><span class="b" style="background:var(--red)"></span>${c.na} not answered</div>
            <div><span class="b"></span>${c.nv} not visited</div>
            <div><span class="b" style="background:var(--purple)"></span>${c.mk} marked</div>
            <div style="grid-column:span 2"><span class="b" style="background:var(--purple)"></span>${c.am} answered and marked (scored)</div>
          </div>
          <div class="palette">${inSec.map(([q, i]) => `<button class="pq ${state(q)} ${i === cur ? 'cur' : ''}" data-go="${i}">${i + 1}</button>`).join('')}</div>`}
          <button class="dark" id="submit" style="width:100%;margin-top:14px">Submit exam</button>
        </div>
      </div>`;
    app.querySelectorAll('[data-sec]').forEach(b => b.onclick = () => {
      if (b.dataset.sec === 'C') { flushTime(); sec = 'C'; render(); return; }
      const i = qs.findIndex(q => q.part === b.dataset.sec);
      go(i);
    });
    app.querySelectorAll('[data-go]').forEach(b => b.onclick = () => go(+b.dataset.go));
    app.querySelectorAll('.opt').forEach(o => o.onclick = () => {
      const r = R(qs[cur]);
      if (r.choice === o.dataset.k) return;
      r.choice = o.dataset.k;
      r.history.push({ t: elapsed(), c: r.choice });
      touch(); render();
    });
    const on = (id, f) => { const b = document.getElementById(id); if (b) b.onclick = f; };
    on('clear', () => { const r = R(qs[cur]); if (r.choice) { r.choice = ''; r.history.push({ t: elapsed(), c: '' }); touch(); render(); } });
    on('mark', () => { R(qs[cur]).marked = true; go(cur + 1); });
    on('next', () => { R(qs[cur]).marked = false; if (cur + 1 < qs.length) go(cur + 1); else { touch(); render(); } });
    on('prev', () => go(cur - 1));
    on('submit', () => submit(false));
  }
  render();
}

/* ---------------- result ---------------- */
async function result(id) {
  const d = await api('/api/exams/' + id);
  const at = d.attempt, r = at.result;
  if (!r) { location.hash = '#/exam/' + id; return; }
  const byId = Object.fromEntries(d.questions.map((q, i) => [q.id, { ...q, n: i + 1 }]));
  const A = r.parts.A, B = r.parts.B, T = r.total;
  const tallyRow = (name, t) => `<tr><td>${name}</td><td class="num">${t.correct}</td><td class="num">${t.wrong}</td><td class="num">${t.blank}</td>
    <td class="num">${pct(t.correct, t.correct + t.wrong)}%</td><td class="num"><b>${t.score.toFixed(2)}</b></td><td class="num">${mmss(t.seconds)}</td></tr>`;
  let filter = r.questions.some(q => q.status !== 'correct') ? 'wrong+blank' : 'all';
  app.innerHTML = `
    <h1>${esc(d.title)}: result</h1>
    <p class="muted">${at.mode === 'ab' ? 'Parts A and B' : 'Full paper'} · submitted ${when(at.submitted)} · used ${mmss(r.elapsed_seconds)} of ${at.minutes}:00${r.late ? ' · <b>submitted late</b>' : ''}</p>
    <div class="kpis">
      <div class="kpi"><div class="v">${T.score.toFixed(2)}</div><div class="l">score out of 100 (Parts A and B)</div></div>
      <div class="kpi"><div class="v">${A.score.toFixed(2)}</div><div class="l">Part A out of 40</div></div>
      <div class="kpi"><div class="v">${B.score.toFixed(2)}</div><div class="l">Part B out of 60</div></div>
      <div class="kpi"><div class="v">−${r.neg_lost.toFixed(2)}</div><div class="l">lost to negative marking (${T.wrong} wrong)</div></div>
      <div class="kpi"><div class="v">${pct(T.correct, T.correct + T.wrong)}%</div><div class="l">accuracy on attempted questions</div></div>
    </div>
    <table><tr><th>Part</th><th class="num">Correct</th><th class="num">Wrong</th><th class="num">Blank</th><th class="num">Accuracy</th><th class="num">Score</th><th class="num">Time</th></tr>
      ${tallyRow('A', A)}${tallyRow('B', B)}${tallyRow('Total', T)}</table>
    <p class="small">Answer changes: right to wrong <b>${r.right_to_wrong}</b>, wrong to right <b>${r.wrong_to_right}</b>, wrong to wrong <b>${r.wrong_to_wrong}</b>.
      Left the exam window <b>${r.blurs}</b> time${r.blurs === 1 ? '' : 's'}.${at.mode === 'full' ? ` Time on the Part C tab: <b>${mmss(at.partc_seconds)}</b>.` : ''}</p>

    <h2>By topic</h2>
    <table><tr><th>Area</th><th>Topic</th><th class="num">Correct</th><th class="num">Wrong</th><th class="num">Blank</th><th class="num">Score</th><th class="num">Avg time</th></tr>
      ${r.topics.map(t => { const n = t.correct + t.wrong + t.blank; return `<tr style="${t.wrong + t.blank ? 'background:#fdf6f4' : ''}"><td class="muted">${esc(t.area)}</td><td>${esc(t.topic)}</td>
        <td class="num">${t.correct}</td><td class="num">${t.wrong}</td><td class="num">${t.blank}</td><td class="num">${t.score.toFixed(2)}</td><td class="num">${mmss(t.seconds / n)}</td></tr>`; }).join('')}</table>

    <h2>Review</h2>
    <div class="row filters" id="filters">${['wrong+blank', 'wrong', 'blank', 'changed', 'marked', 'slow', 'all'].map(f => `<button data-f="${f}">${{ 'wrong+blank': 'Wrong and blank', wrong: 'Wrong', blank: 'Blank', changed: 'Changed answer', marked: 'Marked', slow: 'Slowest 15', all: 'All' }[f]}</button>`).join('')}</div>
    <div id="review" style="margin-top:10px"></div>
    ${d.partc ? `<h2>Part C: marking guide</h2>
      <p class="small">Mark your handwritten answers against this, or photograph them and send them to Claude.</p>
      ${d.partc.map(p => `<div class="card"><b>${esc(p.id)}. ${fmt(p.title)}</b><p>${fmt(p.prompt)}</p><div class="explain">${fmt(p.guide)}</div></div>`).join('')}
      ${d.partc_note ? `<p class="small">${fmt(d.partc_note)}</p>` : ''}` : ''}
    <p><a href="#/">Back to home</a></p>`;

  function showReview() {
    let list = r.questions;
    if (filter === 'wrong+blank') list = list.filter(q => q.status !== 'correct');
    else if (filter === 'wrong' || filter === 'blank') list = list.filter(q => q.status === filter);
    else if (filter === 'changed') list = list.filter(q => q.changes > 0);
    else if (filter === 'marked') list = list.filter(q => q.marked);
    else if (filter === 'slow') list = [...list].sort((a, b) => b.seconds - a.seconds).slice(0, 15);
    document.querySelectorAll('#filters button').forEach(b => b.classList.toggle('on', b.dataset.f === filter));
    document.getElementById('review').innerHTML = list.length ? list.map(x => {
      const q = byId[x.id];
      return `<div class="card">
        <div class="qhead"><span>Question ${q.n} <span class="tag ${x.status}">${x.status}</span> <span class="tag">${esc(q.topic)}</span></span>
          <span class="muted small">${Math.round(x.seconds)} s${x.changes ? ` · changed ${x.changes}× (first pick ${x.first})` : ''}${x.marked ? ' · marked' : ''}</span></div>
        ${questionBody(q)}
        ${q.options.map((o, k) => `<div class="opt ${K[k] === x.answer ? 'right' : K[k] === x.choice ? 'wrongpick' : ''}"><span class="k">(${K[k]})</span><span>${fmt(o)}</span>
          ${K[k] === x.choice ? '<span class="muted small" style="margin-left:auto">your answer</span>' : ''}</div>`).join('')}
        <div class="explain">${fmt(x.explanation)}${x.source ? ` <span class="muted small">[${esc(x.source)}]</span>` : ''}</div></div>`;
    }).join('') : '<p class="muted">Nothing in this list.</p>';
  }
  document.querySelectorAll('#filters button').forEach(b => b.onclick = () => { filter = b.dataset.f; showReview(); });
  showReview();
}

/* ---------------- practice ---------------- */
async function practice(qs) {
  const d = await api('/api/practice?' + qs);
  const list = d.questions;
  if (!list.length) {
    app.innerHTML = `<div class="card">No questions match this choice${new URLSearchParams(qs).get('mode') === 'mistakes' ? ': you have no open mistakes in these topics' : ''}. <a href="#/">Back</a></div>`;
    return;
  }
  let i = 0, shown = Date.now();
  const done = [];
  function render() {
    if (i >= list.length) return summary();
    const q = list[i], res = done[i];
    const right = done.filter(x => x.correct).length;
    app.innerHTML = `
      <div class="exambar"><div><b>Practice</b> <span class="muted small">question ${i + 1} of ${list.length} · ${right} correct so far</span></div>
        <button id="stop">Finish</button></div>
      <div class="card">
        <div class="qhead"><span><span class="tag">${esc(q.topic)}</span></span><span class="muted small">${esc(q.bank)}</span></div>
        ${questionBody(q)}
        ${q.options.map((o, k) => {
          let cls = '';
          if (res) cls = K[k] === res.answer ? 'right' : K[k] === res.choice ? 'wrongpick' : '';
          return `<div class="opt ${cls}" data-k="${K[k]}"><span class="k">(${K[k]})</span><span>${fmt(o)}</span></div>`;
        }).join('')}
        ${res ? `<div class="explain"><b>${res.correct ? 'Correct.' : 'Wrong: the answer is (' + res.answer + ').'}</b> ${fmt(res.explanation)}${res.source ? ` <span class="muted small">[${esc(res.source)}]</span>` : ''}</div>
          <div class="actions"><span class="grow"></span><button class="primary" id="next">${i + 1 < list.length ? 'Next question' : 'See summary'}</button></div>` : ''}
      </div>`;
    if (!res) app.querySelectorAll('.opt').forEach(o => o.onclick = async () => {
      const seconds = (Date.now() - shown) / 1000;
      const out = await api('/api/practice/answer', { method: 'POST', body: JSON.stringify({ session: d.session, qid: q.id, choice: o.dataset.k, seconds }) });
      done[i] = { ...out, choice: o.dataset.k, q, seconds };
      render();
    });
    const nx = document.getElementById('next');
    if (nx) { nx.onclick = () => { i++; shown = Date.now(); render(); }; nx.focus(); }
    document.getElementById('stop').onclick = summary;
  }
  function summary() {
    const ans = done.filter(Boolean), right = ans.filter(x => x.correct).length, wrong = ans.filter(x => !x.correct);
    app.innerHTML = `<h1>Practice summary</h1>
      <div class="kpis"><div class="kpi"><div class="v">${right} / ${ans.length}</div><div class="l">correct</div></div>
        <div class="kpi"><div class="v">${pct(right, ans.length)}%</div><div class="l">accuracy</div></div>
        <div class="kpi"><div class="v">${ans.length ? Math.round(ans.reduce((s, x) => s + x.seconds, 0) / ans.length) : 0} s</div><div class="l">average time</div></div></div>
      ${wrong.length ? '<h2>Wrong answers</h2>' + wrong.map(x => `<div class="card"><span class="tag">${esc(x.q.topic)}</span>${questionBody(x.q)}
        <p class="small">You chose (${x.choice}): ${fmt(x.q.options[K.indexOf(x.choice)])}<br>Answer (${x.answer}): ${fmt(x.q.options[K.indexOf(x.answer)])}</p>
        <div class="explain">${fmt(x.explanation)}</div></div>`).join('') : ''}
      <div class="row"><a class="btn" href="#/">Home</a><a class="btn" href="#/stats">Topics and mistakes</a></div>`;
  }
  render();
}

/* ---------------- stats ---------------- */
async function stats() {
  const s = await api('/api/stats');
  const rows = s.topics;
  app.innerHTML = `<h1>Topics and mistakes</h1>
    <p class="muted small">Counts every answer you have given, in exams and in practice. An open mistake is a question whose latest answer was wrong.</p>
    <table><tr><th>Area</th><th>Topic</th><th class="num">Seen</th><th class="num">Answers</th><th>Accuracy</th><th class="num">Avg time</th><th class="num">Open mistakes</th><th></th></tr>
    ${rows.map(t => `<tr><td class="muted">${esc(t.area)}</td><td>${esc(t.topic)}</td><td class="num">${t.seen} / ${t.total}</td><td class="num">${t.answers}</td>
      <td>${t.answers ? `<div class="row" style="flex-wrap:nowrap"><div class="bar"><i class="${barClass(t.accuracy)}" style="width:${t.accuracy}%"></i></div><span class="small">${t.accuracy}%</span></div>` : '<span class="muted small">not yet</span>'}</td>
      <td class="num">${t.answers ? t.avg_seconds + ' s' : ''}</td><td class="num">${t.open_mistakes || ''}</td>
      <td><a href="#/practice?${new URLSearchParams({ topics: t.topic, mode: 'all', n: 1000 })}">Practise</a></td></tr>`).join('')}</table>
    <h2>Open mistakes (${s.mistakes.length})</h2>
    ${s.mistakes.length ? `<p><a class="btn" href="#/practice?${new URLSearchParams({ mode: 'mistakes', n: 1000 })}">Practise all open mistakes</a></p>` +
      s.mistakes.map(m => `<div class="card"><div class="qhead"><span><span class="tag">${esc(m.q.topic)}</span> <span class="muted small">${esc(m.q.id)}</span></span>
        <span class="muted small">wrong ${m.times_wrong}×, right ${m.times_right}× · last ${esc(m.last)}</span></div>
        ${questionBody(m.q)}
        ${m.q.options.map((o, k) => `<div class="opt ${K[k] === m.answer ? 'right' : K[k] === m.last_choice ? 'wrongpick' : ''}"><span class="k">(${K[k]})</span><span>${fmt(o)}</span></div>`).join('')}
        <div class="explain">${fmt(m.explanation)}</div></div>`).join('') : '<p class="muted">None yet.</p>'}`;
}
