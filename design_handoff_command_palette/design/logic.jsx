// Matching: each typed word must land at the start of a word, on a word's
// initials, or (3+ characters) inside a word. Short fragments never match
// mid-word — "ai" finds "AI control" and "Aisha", not "Trailers".
const BOUND = /[^a-z0-9]/i;
function isStart(text, i) {
  if (i === 0) return true;
  const p = text[i - 1], c = text[i];
  if (BOUND.test(p)) return true;
  if (/[a-z]/.test(p) && /[A-Z]/.test(c)) return true;
  if (/[a-z]/i.test(p) && /[0-9]/.test(c)) return true;
  return false;
}
function matchTerm(t, text) {
  if (!text) return null;
  const lt = text.toLowerCase();
  if (lt.startsWith(t)) return { s: 1, r: [[0, t.length]] };
  for (let i = 1; i < lt.length; i++) {
    if (isStart(text, i) && lt.startsWith(t, i)) return { s: 0.9 - Math.min(i, 40) * 0.002, r: [[i, i + t.length]] };
  }
  if (t.length >= 2) {
    const starts = [];
    for (let i = 0; i < text.length; i++) if (isStart(text, i) && /[a-z0-9]/i.test(text[i])) starts.push(i);
    let j = 0; const r = [];
    for (const i of starts) { if (j < t.length && lt[i] === t[j]) { r.push([i, i + 1]); j++; } }
    if (j === t.length && starts.length > 1) return { s: 0.72, r };
  }
  if (t.length >= 3) {
    const i = lt.indexOf(t);
    if (i >= 0) return { s: 0.5, r: [[i, i + t.length]] };
  }
  return null;
}

function fieldsFor(item) {
  if (item.kind === "record") {
    const d = item.d;
    switch (item.type) {
      case "shipment": return [{ k: "title", t: d.id, w: 1 }, { k: "cust", t: d.cust, w: 0.92 }, { k: "route", t: `${d.o} → ${d.d}`, w: 0.75 }, { k: "bol", t: "BOL " + d.bol, w: 0.85 }, { k: "drv", t: d.driver, w: 0.6, via: "Driver" }];
      case "customer": return [{ k: "title", t: d.name, w: 1 }, { k: "code", t: d.id, w: 0.95 }, { k: "city", t: d.city, w: 0.6 }];
      case "worker": return [{ k: "title", t: d.name, w: 1 }, { k: "code", t: d.id, w: 0.9 }, { k: "role", t: d.role, w: 0.5 }, { k: "home", t: d.home, w: 0.5 }];
      case "document": return [{ k: "title", t: d.name, w: 1 }, { k: "dkind", t: d.kind, w: 0.75 }, { k: "on", t: d.on, w: 0.8 }];
    }
  }
  if (item.kind === "page") return [{ k: "title", t: item.title, w: 1 }, { k: "trail", t: item.trail.join(" › "), w: 0.62 }, ...item.kw.map((x) => ({ k: "kw", t: x, w: 0.82, via: "Keyword" }))];
  if (item.kind === "command") return [{ k: "title", t: item.label, w: 1 }, ...item.kw.map((x) => ({ k: "kw", t: x, w: 0.8, via: "Keyword" })), { k: "desc", t: item.desc, w: 0.45 }];
  return [];
}

function scoreItem(q, item) {
  const fields = fieldsFor(item);
  const ql = q.trim().toLowerCase();
  if (!ql) return null;
  const title = fields[0];
  if (ql.includes(" ") && title.t.toLowerCase().startsWith(ql)) return { score: 1.05, hl: { title: [[0, ql.length]] }, via: null };
  const terms = ql.split(/\s+/);
  let sum = 0, onTitle = 0; const hl = {}; let via = null;
  for (const t of terms) {
    let best = null;
    for (const f of fields) {
      const m = matchTerm(t, f.t);
      if (m && (!best || m.s * f.w > best.v)) best = { v: m.s * f.w, f, r: m.r };
    }
    if (!best) return null;
    sum += best.v;
    if (best.f.k === "title") onTitle++;
    if (best.f.via) { if (!via) via = { label: best.f.via, text: best.f.t, r: best.r }; }
    else (hl[best.f.k] = hl[best.f.k] || []).push(...best.r);
  }
  const score = sum / terms.length + (onTitle === terms.length ? 0.04 : 0);
  return { score, hl, via };
}

function HL({ text, r }) {
  if (!r || !r.length || !text) return text || null;
  const rs = r.slice().sort((a, b) => a[0] - b[0]);
  const merged = [];
  for (const x of rs) { const last = merged[merged.length - 1]; if (last && x[0] <= last[1]) last[1] = Math.max(last[1], x[1]); else merged.push([x[0], x[1]]); }
  const out = []; let at = 0;
  merged.forEach(([a, b], i) => { if (a > at) out.push(text.slice(at, a)); out.push(<mark key={i} className="hl">{text.slice(a, b)}</mark>); at = b; });
  if (at < text.length) out.push(text.slice(at));
  return out;
}

const QWORDS = /^(how|what|whats|what's|why|which|who|whos|when|where|is|are|did|does|do|can|could|should|will|show me|list|find me|tell me|summarize|draft|any)\b/i;
function isQuestion(q) {
  const s = q.trim();
  if (!s) return false;
  if (/\?$/.test(s)) return true;
  return QWORDS.test(s) && s.split(/\s+/).length >= 3;
}

function scopeOf(item) { return item.kind === "record" ? item.type : item.kind; }

function search(q, scope, commands) {
  const pool = [...RECORDS, ...PAGES, ...commands].filter((it) => scope === "all" || scopeOf(it) === scope);
  const out = [];
  for (const item of pool) { const m = scoreItem(q, item); if (m) out.push({ item, m }); }
  out.sort((a, b) => b.m.score - a.m.score);
  return out;
}

// ——— Nova answers (canned, grounded in the fixture records) ———
function novaAnswer(q, subject, hits) {
  const ql = q.toLowerCase();
  if (subject) {
    const r = subject, d = r.d, k = r.key;
    const qs = SUBJECT_QUESTIONS[r.type];
    let i = qs.indexOf(q); if (i < 0) i = /draft|write|email|remind/.test(ql) ? (r.type === "shipment" ? 1 : 2) : /bill|invoice|expir|match|sent|who/.test(ql) ? (r.type === "customer" ? 1 : 2) : 0;
    const steps = { shipment: ["Read stops, ELD pings and events", "Checked the delivery window"], customer: ["Read open invoices and aging", "Compared volume to last quarter"], worker: ["Read hours of service from the ELD", "Checked compliance dates"], document: ["Read the extraction", "Compared it to the linked record"] }[r.type];
    let text = "";
    if (r.type === "shipment") {
      text = [
        `[[${k}]] is ${d.late.toLowerCase()} — current ETA ${d.eta}. ${d.ev.length > 1 ? `Most of the slip came from one event: ${d.ev[1][0].toLowerCase()}.` : ""} ${d.note ? d.note + "." : ""}`,
        `Here's a draft for ${d.cust}: “Quick update on ${d.id} (BOL ${d.bol}). ${d.driver ? d.driver + " is" : "It's"} ${d.done > 0 && d.done < 1 ? `on the way to ${d.d}, now expected ${d.eta.replace("Today ", "at ")}` : `scheduled to pick up ${d.pu}`}. We'll send the signed POD the moment it's in.”`,
        d.status === "ready" || d.status === "completed" || d.status === "invoiced" ? `Nothing — [[${k}]] is delivered and the POD matched. ${money(d.total)} goes to [[r:customer:${d.code}]] on the next billing run.` : `It still needs a signed POD after delivery in ${d.d}. Charges are complete at ${money(d.total)}, so [[${k}]] moves to the billing queue on its own once the POD lands.`,
      ][i];
    } else if (r.type === "customer") {
      const past = d.aging[1] + d.aging[2] + d.aging[3];
      text = [
        past ? `[[${k}]] owes ${money(d.bal)}, of which ${money(past)} is past ${d.terms}. ${d.aging[3] ? `${money(d.aging[3])} is over 90 days — mostly one disputed detention charge.` : ""} ${d.credit > 0.8 ? `They're at ${Math.round(d.credit * 100)}% of a ${d.limit} credit limit.` : ""}` : `Nothing — every invoice for [[${k}]] is inside ${d.terms}.`,
        `[[${k}]] has ${d.open} open load${d.open > 1 ? "s" : ""} and is running about 12% ahead of last quarter's pace, most of it out of ${d.city}.`,
        `Here's a draft to ${d.contact.split(" · ")[0]}: “A friendly reminder that ${money(past || d.bal)} on your account is past ${d.terms}. I've attached the open invoices — happy to resend anything.”`,
      ][i];
    } else if (r.type === "worker") {
      const tight = d.hos[0][2] < 0.3;
      text = [
        tight ? `Not safely. [[${k}]] has ${d.hos[0][1]} of drive time and is still on [[r:shipment:${d.load}]]. Look at [[r:worker:W-1088]] instead — a full 11h clock, home in Houston.` : `Yes. [[${k}]] has ${d.hos[0][1]} of drive time${d.load ? ` after [[r:shipment:${d.load}]]` : ""} and ${d.hos[2][1]} on the 70-hour cycle.`,
        `${d.medWarn ? `The medical card expires ${d.med} — under 30 days.` : "Nothing in the next 90 days."} CDL is good through ${d.cdl}.`,
        `[[${k}]] drove 41h 20m this week across 5 loads with no violations; ${d.hos[2][1]} remain on the cycle.`,
      ][i];
    } else {
      const on = recordById(d.on);
      text = [
        `From this ${d.kind.toLowerCase()} Nova pulled ${d.fields.map((f) => f[0].toLowerCase() + " " + f[1]).join(", ")} — ${d.conf}% confidence.`,
        on ? `Yes — it lines up with [[${on.key}]]. ${d.fields[0][0]} and ${d.fields[1][0].toLowerCase()} match what's on the record.` : "There's no linked record to compare it to yet.",
        `It came in through ${d.by} on ${d.when}.`,
      ][i];
    }
    return { steps, text };
  }
  if (/late|behind|delay|risk|eta|on time/.test(ql)) return { steps: ["Read 42 in-transit shipments", "Compared ETAs to delivery windows", "Checked driver hours"], text: "Two loads are behind. [[r:shipment:S-240193]] is 1h 40m late into Atlanta after a 52-minute weigh station hold, but Acme's receiver is open until 18:00. [[r:shipment:S-240201]] is the real risk — [[r:worker:W-1063]] has 2h 10m of drive time for about 3h of road. Warn [[r:customer:HALV]] now or relay the last leg." };
  if (/bill|invoice|revenue|ready/.test(ql)) return { steps: ["Read the billing queue", "Checked PODs and accessorials"], text: "Six shipments are ready to invoice, worth $18,420 — including [[r:shipment:S-240177]] for [[r:customer:NWG]], whose POD matched yesterday. Two more are held for missing lumper receipts." };
  if (/hour|hos|drive|available|driver|who can/.test(ql)) return { steps: ["Read hours of service for 64 workers", "Matched against open pickups"], text: "Nine drivers are available now. The best fit for tomorrow is [[r:worker:W-1088]] — a full 11h clock, home in Houston, right where [[r:shipment:S-240205]] picks up at 10:00 with no driver yet." };
  if (/owe|ar\b|overdue|aging|balance|pay|credit/.test(ql)) return { steps: ["Read open AR for 128 customers"], text: "[[r:customer:HALV]] carries the most past-due: $21,000 over 60 days at 88% of their credit limit. [[r:customer:SBEV]] is already on hold with $7,400 past 60." };
  const recs = hits.filter((h) => h.item.kind === "record").slice(0, 2);
  return {
    steps: ["Searched shipments, customers, workers and documents"],
    text: recs.length ? `I don't have a direct answer for that yet. The closest records are ${recs.map((h) => `[[${h.item.key}]]`).join(" and ")}. Continue in the Desk and I'll dig further, including mail in Triage.` : "I couldn't find anything in your records about that. Continue in the Desk and I'll look wider, including mail in Triage.",
  };
}
function tokenize(text) {
  const out = [];
  text.split(/(\[\[[^\]]+\]\])/).forEach((p) => {
    if (!p) return;
    const m = p.match(/^\[\[(.+)\]\]$/);
    if (m) out.push({ ref: m[1] }); else p.split(/(\s+)/).forEach((w) => w && out.push(w));
  });
  return out;
}

// ——— Actions ———
const A = (id, label, icon, intent, keys, extra) => ({ id, label, icon, intent, keys, ...extra });
const RECORD_OPEN = { shipment: "Open shipment", customer: "Open customer", worker: "Open worker", document: "Preview document" };
function buildActions(item, ctx) {
  if (!item) return [];
  const link = (href, label) => [A("tab", "Open in new tab", "upRight", { t: "tab", label }, [MOD, "↵"]), A("link", "Copy link", "link", { t: "copy", what: "Link", text: location.origin + href }, [MOD, "L"])];
  if (item.kind === "record") {
    const d = item.d, ty = item.type;
    const base = [A("open", RECORD_OPEN[ty], ty === "document" ? "eye" : "enter", { t: "nav", rec: item }, ["↵"]), A("ask", "Ask Nova about this", "nova", { t: "ask", rec: item }, [MOD, "I"])];
    const href = `/${ty}s/${item.id}`;
    const extra = {
      shipment: [
        !d.driver && A("assign", "Assign a driver", "user", { t: "navl", label: `Assign driver · ${d.id}` }),
        A("cust", `Open ${d.cust}`, "building", { t: "nav", rec: recordById(d.code) }),
        A("bol", "Copy BOL", "copy", { t: "copy", what: "BOL", text: d.bol }),
      ],
      customer: [
        A("ledger", "Open customer ledger", "book", { t: "navl", label: `Customer ledger · ${d.name}` }),
        A("newship", `New shipment for ${d.name}`, "plus", { t: "navl", label: `New shipment · ${d.name}` }),
      ],
      worker: [
        A("comp", "Open compliance", "shield", { t: "navl", label: `${d.name} · Compliance` }),
        A("pto", "Open time off", "calendar", { t: "navl", label: `${d.name} · Time off` }),
        d.load && A("load", `Open ${d.load}`, "truck", { t: "nav", rec: recordById(d.load) }),
      ],
      document: [
        A("dl", "Download", "download", { t: "toast", msg: `Downloading ${d.name}` }),
        recordById(d.on) && A("rec", `Open ${d.on}`, "folder", { t: "nav", rec: recordById(d.on) }),
      ],
    }[ty].filter(Boolean);
    return [...base, ...link(href, item.title), A("id", ty === "shipment" ? "Copy PRO number" : ty === "document" ? "Copy file name" : "Copy name", "copy", { t: "copy", what: ty === "shipment" ? "PRO number" : "Name", text: item.title }, [ALT, "C"]), ...extra];
  }
  if (item.kind === "page") {
    const pinned = ctx.pinned.includes(item.id);
    const rel = (item.related || []).map((id) => ctx.commands.find((c) => c.id === id)).filter(Boolean).map((c) => A("rel-" + c.id, c.label, c.icon, { t: "cmd", cmd: c }));
    return [A("open", "Open page", "enter", { t: "navp", page: item }, ["↵"]), ...link(item.href, item.title), A("pin", pinned ? "Unpin from home" : "Pin to home", "pin", { t: "pin", id: item.id }, [MOD, "P"]), ...rel];
  }
  if (item.kind === "command") return [A("open", item.label, item.icon, { t: "cmd", cmd: item }, ["↵"], { danger: item.danger })];
  if (item.kind === "attention") { const p = PAGE_BY_ID[item.page]; return [A("open", "Open the queue", "enter", { t: "navl", label: item.label, crumb: p.trail.concat(p.title) }, ["↵"]), ...link(p.href, item.label)]; }
  if (item.kind === "ask") return [A("open", "Ask Nova", "nova", { t: "nova" }, ["↵"])];
  return [];
}

Object.assign(window, { matchTerm, scoreItem, HL, isQuestion, search, scopeOf, novaAnswer, tokenize, buildActions });
