"use strict";

(() => {
  const Sim = window.Sim;
  const esc = Sim.esc;
  const $ = (id) => document.getElementById(id);

  const EVENTS = [
    "message.received", "message.scheduled", "message.sent", "message.failed", "message.loaded",
    "ack.scheduled", "ack.sent", "ack.failed", "ack.withheld", "ack.received",
    "ta1.scheduled", "ta1.sent", "ta1.received", "answer.scheduled",
    "mdn.received", "mdn.sent", "mdn.failed", "file.picked_up", "reset",
    "run.started", "run.scenario", "run.step", "run.finished", "run.reset",
  ];

  const state = {
    partners: [],
    current: null,
    convs: [],
    detail: null,
    loaded: false,
    offline: false,
    sel: { partner: null, conv: null, msg: null },
    picked: false,
    inspKey: "",
  };

  async function api(path) {
    const response = await fetch(path, { headers: { Accept: "application/json" }, cache: "no-store" });
    if (response.status === 404) return null;
    if (!response.ok) throw new Error(`${path} answered ${response.status}`);
    return response.json();
  }

  const partnerOf = (name) => state.partners.find((p) => p.name === name);
  const titleCase = (s) => (s ? s[0].toUpperCase() + s.slice(1) : "");

  function duration(first, last) {
    const ms = Math.max(0, new Date(last) - new Date(first));
    const s = Math.round(ms / 1000);
    if (s < 1) return "";
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m`;
    return `${Math.floor(m / 60)}h ${String(m % 60).padStart(2, "0")}m`;
  }

  function when(iso) {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "";
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }

  function host(api) {
    try { return new URL(api).host; } catch { return api || ""; }
  }

  function runLine() {
    const r = state.current;
    if (!r) return state.loaded ? "no run yet" : "";
    const parts = [];
    if (r.kind === "conform") parts.push(`seed ${r.seed}`);
    else if (r.kind === "soak") parts.push(`soak run ${r.number}`);
    else parts.push(`run ${r.number}`);
    parts.push(when(r.startedAt));
    if (r.kind === "conform") parts.push("conform, no Trenova");
    else if (r.api) parts.push(`against ${host(r.api)}`);
    if (r.source === "report" && r.file) parts.push(`from ${r.file}`);
    else if (r.status === "running") parts.push("running");
    return parts.join(" · ");
  }

  function header() {
    $("run").textContent = runLine();
    const t = state.current?.tally || { passed: 0, failed: 0, waiting: 0 };
    $("tally").innerHTML =
      `<span><i class="dot ok"></i><b>${t.passed}</b> passed</span>` +
      `<span><i class="dot bad"></i><b>${t.failed}</b> failed</span>` +
      `<span><i class="dot hold"></i><b>${t.waiting}</b> waiting on a planner</span>`;
  }

  function partnerDot(tally) {
    if (!tally) return "";
    if (tally.failed) return "bad";
    if (tally.waiting) return "hold";
    return tally.passed ? "ok" : "";
  }

  function partners() {
    if (!state.loaded) { $("partners").innerHTML = ""; return; }
    if (state.offline && !state.partners.length) {
      $("partners").innerHTML = `<li class="row2 blank">The simulator is not answering.</li>`;
      return;
    }
    $("partners").innerHTML = state.partners.length ? state.partners.map((p) => {
      const wire = [p.transport.toUpperCase(), p.version, p.acknowledgment?.expects].filter((x) => x && x !== "none").join(" · ");
      return `
    <li><button data-p="${esc(p.name)}" aria-current="${p.name === state.sel.partner}">
      <span class="row1"><i class="dot ${partnerDot(p.run)}"></i><span class="name">${esc(p.display)}</span></span>
      <span class="row2">${esc(titleCase(p.role))} · ${esc(wire)}</span>
    </button></li>`;
    }).join("") : `<li class="row2 blank">No partners loaded. Start edi-sim serve with --partners set to a folder of partners.</li>`;
  }

  function convDot(scenario) {
    if (!scenario) return "";
    if (scenario.status === "failed") return "bad";
    if (scenario.status === "passed") return "ok";
    return scenario.waiting ? "hold" : "";
  }

  function convs() {
    const p = partnerOf(state.sel.partner);
    $("convMeta").textContent = p ? `${state.convs.length}` : "";
    $("laneP").innerHTML = p ? `${esc(p.display)}<small>${esc(p.them.id)} · ${esc(p.them.qualifier)}</small>` : "";
    $("laneT").innerHTML = p ? `Trenova<small>${esc(p.trenovaEnvelope.id)} · ${esc(p.trenovaEnvelope.qualifier)}</small>` : "Trenova";
    if (!p) { $("convs").innerHTML = ""; return; }
    $("convs").innerHTML = state.convs.length ? state.convs.map((c) => `
    <li><button data-c="${esc(c.key)}" aria-current="${c.key === state.sel.conv}">
      <span class="row1"><i class="dot ${convDot(c.scenario)}"></i><span class="name key">${esc(c.key)}</span><span class="count">${esc(duration(c.first, c.last))}</span></span>
      <span class="row2">${esc(c.scenario?.name || c.sets.join(" · "))}</span>
    </button></li>`).join("") : `<li class="row2 blank">No conversations with ${esc(p.display)} yet. They appear here as messages arrive.</li>`;
  }

  function ladder() {
    const svg = $("ladder");
    const detail = state.detail;
    const empty = $("ladderEmpty");
    const focused = document.activeElement?.closest?.("[data-m]")?.dataset.m;
    if (!detail) {
      svg.innerHTML = "";
      svg.setAttribute("height", 0);
      $("laneP").style.paddingLeft = "";
      $("ladderTitle").textContent = "Conversation";
      $("ladderMeta").textContent = "";
      empty.textContent = state.sel.partner && state.loaded && !state.convs.length ? "No conversation yet." : "";
      empty.hidden = !empty.textContent;
      return;
    }
    $("ladderTitle").textContent = detail.key;
    $("ladderMeta").textContent = detail.scenario?.name || "";
    empty.textContent = detail.rows.length ? "" : "Nothing on the wire yet.";
    empty.hidden = !empty.textContent;
    const lifeline = Sim.drawLadder(svg, detail.rows, state.sel.msg);
    $("laneP").style.paddingLeft = `${Math.max(0, lifeline - 18)}px`;
    if (focused) svg.querySelector(`[data-m="${focused}"]`)?.focus({ preventScroll: true });
  }

  function inspector(force) {
    const detail = state.detail;
    const row = detail?.rows.find((r) => r.kind === "message" && r.number === state.sel.msg);
    const message = row ? detail.messages.find((m) => m.id === row.messageId) : null;
    const ctx = {
      row,
      message,
      partner: partnerOf(state.sel.partner),
      scenario: detail?.scenario,
      empty: detail && detail.rows.length ? "Select a message on the ladder." : "",
    };
    const doc = message?.documents?.[row.document];
    const key = JSON.stringify([row, message?.state, doc?.answeredBy, doc?.answers, ctx.scenario?.steps?.map((s) => [s.status, s.why]), ctx.empty]);
    if (!force && key === state.inspKey) return;
    state.inspKey = key;
    Sim.renderInspector($("insp"), $("inspMeta"), ctx);
  }

  function defaultMessage() {
    const rows = state.detail?.rows.filter((r) => r.kind === "message") || [];
    const failing = rows.find((r) => r.tone === "bad" || r.expect === "bad");
    return (failing || rows[rows.length - 1])?.number ?? null;
  }

  function writeHash() {
    const { partner, conv, msg } = state.sel;
    if (!partner) return;
    let hash = `#${partner}`;
    if (conv) hash += `-${encodeURIComponent(conv)}`;
    if (conv && msg) hash += `-${msg}`;
    if (location.hash !== hash) history.replaceState(null, "", hash);
  }

  function readHash() {
    let raw = location.hash.slice(1);
    try { raw = decodeURIComponent(raw); } catch { return null; }
    const p = state.partners.find((x) => raw === x.name || raw.startsWith(`${x.name}-`));
    if (!p) return null;
    return { partner: p.name, rest: raw.slice(p.name.length + 1) };
  }

  function resolveRest(rest) {
    if (!rest) return { conv: null, msg: null };
    if (state.convs.some((c) => c.key === rest)) return { conv: rest, msg: null };
    const m = rest.match(/^(.*)-(\d+)$/);
    if (m && state.convs.some((c) => c.key === m[1])) return { conv: m[1], msg: Number(m[2]) };
    return { conv: rest, msg: null };
  }

  async function loadPartners() {
    const [listed, runs] = await Promise.all([api("/control/partners"), api("/control/runs")]);
    state.partners = listed?.partners || [];
    state.current = runs?.current || null;
  }

  async function loadConvs() {
    const p = state.sel.partner;
    state.convs = p ? (await api(`/control/${encodeURIComponent(p)}/conversations`))?.conversations || [] : [];
  }

  async function loadDetail() {
    const { partner, conv } = state.sel;
    state.detail = partner && conv ? await api(`/control/${encodeURIComponent(partner)}/conversations/${encodeURIComponent(conv)}`) : null;
    const present = state.detail?.rows.some((r) => r.kind === "message" && r.number === state.sel.msg);
    if (!state.picked || !present) {
      state.picked = state.picked && present;
      state.sel.msg = defaultMessage();
    }
  }

  function render() { header(); partners(); convs(); ladder(); inspector(true); writeHash(); }

  async function guarded(work) {
    try {
      await work();
      state.offline = false;
    } catch (error) {
      state.offline = true;
      banner(true, "The simulator is not answering · retrying");
    }
    state.loaded = true;
  }

  async function show(partner, conv, msg) {
    state.sel = { partner, conv: conv ?? null, msg: msg ?? null };
    state.picked = msg != null;
    await guarded(async () => {
      await loadConvs();
      if (!state.sel.conv && state.convs.length) state.sel.conv = state.convs[0].key;
      await loadDetail();
    });
    render();
  }

  async function start() {
    await guarded(loadPartners);
    const fromHash = readHash();
    const partner = fromHash?.partner || state.partners[0]?.name || null;
    if (!partner) { render(); return; }
    state.sel.partner = partner;
    await guarded(loadConvs);
    const { conv, msg } = resolveRest(fromHash?.rest);
    await show(partner, conv, msg);
  }

  const pending = { partners: false, convs: false, detail: false };
  let timer = null;

  function schedule(...kinds) {
    for (const k of kinds) pending[k] = true;
    if (!timer) timer = setTimeout(flush, 200);
  }

  async function flush() {
    timer = null;
    const due = { ...pending };
    pending.partners = pending.convs = pending.detail = false;
    await guarded(async () => {
      if (due.partners) await loadPartners();
      if (due.convs) {
        await loadConvs();
        if (!state.sel.conv && state.convs.length) { state.sel.conv = state.convs[0].key; due.detail = true; }
        if (state.sel.conv && !state.convs.some((c) => c.key === state.sel.conv) && !due.detail) due.detail = true;
      }
      if (due.detail) await loadDetail();
    });
    if (due.partners) { header(); partners(); }
    if (due.partners || due.convs) convs();
    if (due.detail) { ladder(); inspector(false); writeHash(); }
  }

  function onEvent(e) {
    let event;
    try { event = JSON.parse(e.data); } catch { return; }
    const { partner, conv } = state.sel;
    if (event.type === "reset" || event.type === "run.reset" || event.type === "run.started" || event.type === "run.finished") {
      schedule("partners", "convs", "detail");
      return;
    }
    if (event.type.startsWith("run.")) {
      schedule("partners");
      if (event.partner === partner) {
        schedule("convs");
        if (conv && event.run?.keys?.includes(conv)) schedule("detail");
      }
      return;
    }
    if (event.partner !== partner) return;
    schedule("convs");
    if (conv && (event.message?.keys || []).includes(conv)) schedule("detail");
  }

  function banner(lost, text) {
    $("conn").hidden = !lost;
    if (text) $("connText").textContent = text;
  }

  let stream = null;
  let lastId = "";
  let retry = 1000;
  let lost = false;

  function connect() {
    stream = new EventSource(`/control/events${lastId ? `?after=${encodeURIComponent(lastId)}` : ""}`);
    stream.onopen = () => {
      retry = 1000;
      if (lost || state.offline) { lost = false; schedule("partners", "convs", "detail"); }
      banner(false);
    };
    stream.onerror = () => {
      lost = true;
      banner(true, "Lost the simulator's event stream · reconnecting");
      if (stream.readyState === EventSource.CLOSED) {
        stream.close();
        setTimeout(connect, retry);
        retry = Math.min(retry * 2, 15000);
      }
    };
    const handle = (e) => { if (e.lastEventId) lastId = e.lastEventId; onEvent(e); };
    for (const type of EVENTS) stream.addEventListener(type, handle);
  }

  function selectMessage(number, focus) {
    state.sel.msg = number;
    state.picked = true;
    ladder();
    inspector(false);
    writeHash();
    if (focus) document.querySelector(`[data-m="${number}"]`)?.focus();
  }

  document.addEventListener("click", (e) => {
    const p = e.target.closest("[data-p]"); if (p) { show(p.dataset.p); return; }
    const c = e.target.closest("[data-c]"); if (c) { show(state.sel.partner, c.dataset.c); return; }
    const m = e.target.closest("[data-m]"); if (m) selectMessage(+m.dataset.m, false);
  });
  document.addEventListener("keydown", (e) => {
    const m = e.target.closest && e.target.closest("[data-m]");
    if (m && (e.key === "Enter" || e.key === " ")) { e.preventDefault(); selectMessage(+m.dataset.m, true); }
  });
  addEventListener("hashchange", async () => {
    const fromHash = readHash();
    if (!fromHash) return;
    if (fromHash.partner !== state.sel.partner) {
      state.sel.partner = fromHash.partner;
      await guarded(loadConvs);
      const { conv, msg } = resolveRest(fromHash.rest);
      await show(fromHash.partner, conv, msg);
      return;
    }
    const { conv, msg } = resolveRest(fromHash.rest);
    if (conv && conv !== state.sel.conv) { await show(fromHash.partner, conv, msg); return; }
    if (msg != null && msg !== state.sel.msg) selectMessage(msg, false);
  });
  let width = 0;
  const relayout = () => {
    const now = $("ladder").clientWidth;
    if (now === width) return;
    width = now;
    ladder();
  };
  addEventListener("resize", relayout);
  if ("ResizeObserver" in window) new ResizeObserver(relayout).observe($("ladderWrap"));

  start().then(connect);
})();
