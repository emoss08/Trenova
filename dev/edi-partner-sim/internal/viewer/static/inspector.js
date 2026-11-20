"use strict";

(() => {
  const Sim = (window.Sim = window.Sim || {});
  const esc = Sim.esc;

  const ackWords = { A: "accepted", E: "accepted with errors", P: "partly accepted", R: "rejected", X: "rejected", M: "rejected", W: "rejected" };

  function chip(tone, text) {
    return `<span class="chip ${tone}"><i class="dot ${tone}"></i>${esc(text)}</span>`;
  }

  function verdictChip(row, message, partner) {
    const expects = partner.acknowledgment?.expects || "997";
    switch (row.verdict) {
      case "breaks": return chip("bad", `Breaks ${partner.display}'s guide`);
      case "rejects": return chip("bad", "Rejects what it answers");
      case "refused": return chip("bad", "Trenova rejected it");
      case "failed": return chip("bad", "Not delivered");
      case "read": return chip("ok", "Trenova read it");
      case "passes": return chip("ok", "Passes the partner's guide");
      case "delivered": return chip("ok", row.ack ? "Delivered to Trenova" : "Delivered");
      case "awaiting": return chip("hold", `Waiting for Trenova's ${expects}`);
      case "scheduled": return chip("hold", `Scheduled for ${Sim.clock(message.dueAt) || "later"}`);
      default: return chip("hold", "Not judged");
    }
  }

  function transport(m) {
    if (m.transport === "sftp") {
      const parts = ["SFTP"];
      if (m.fileName) parts.push(m.fileName);
      if (m.pickedUpAt) parts.push(`picked up ${Sim.clock(m.pickedUpAt)}`);
      return parts.join(" · ");
    }
    const parts = ["AS2"];
    if (m.as2) {
      const how = [m.as2.signed && "signed", m.as2.encrypted && "encrypted", m.as2.compressed && "compressed"].filter(Boolean);
      parts.push(how.length ? how.join(", ") : "plain");
    }
    if (m.mdn) {
      if (m.mdn.mode === "none") {
        parts.push("no MDN");
      } else {
        const how = [m.mdn.signed && "signed", m.mdn.mode];
        if (m.mdn.status && !["processed", "sent", "none"].includes(m.mdn.status)) how.push(m.mdn.status);
        parts.push(`MDN ${how.filter(Boolean).join(", ")}`);
      }
    }
    return parts.join(" · ");
  }

  function answeredBy(doc, message, partner) {
    if (doc.answeredBy) {
      const a = doc.answeredBy;
      return `${a.set} ${a.interchange} at ${Sim.clock(a.at)} · ${ackWords[a.code] || "received"}`;
    }
    const ack = message.ack;
    if (!ack || doc.set === "997" || doc.set === "999" || doc.set === "TA1") return "";
    const set = ack.set || partner.acknowledgment?.expects || "997";
    switch (ack.status) {
      case "awaiting": return `awaiting Trenova's ${set}`;
      case "scheduled": return `${set} scheduled for ${Sim.clock(ack.dueAt)}`;
      case "withheld": return `${set} withheld by the partner`;
      case "failed": return `the ${set} failed${ack.error ? ": " + ack.error : ""}`;
      default: return "";
    }
  }

  function segHTML(elements, sep, term, bad) {
    return elements.map((v, i) => {
      if (i === 0) return `<span class="sid">${esc(v)}</span>`;
      const note = bad.get(i);
      const val = note ? `<span class="el-bad" title="${esc(note.message)}">${esc(v)}</span>` : esc(v);
      return `<span class="sep">${esc(sep)}</span>${val}`;
    }).join("") + (term ? `<span class="sep">${esc(term)}</span>` : "");
  }

  function noteHTML(n) {
    const parts = [];
    if (n.code) parts.push(esc(n.code));
    if (n.rule) parts.push(`rule <code>${esc(n.rule)}</code>${n.file ? ` in <code>${esc(n.file)}</code>` : ""}`);
    else if (n.file) parts.push(`in <code>${esc(n.file)}</code>`);
    return `<div class="seg note-row"><div class="note${n.severity === "error" ? "" : " warn"}"><b>${esc(n.message)}</b>${parts.length ? `<span>${parts.join(" · ")}</span>` : ""}</div></div>`;
  }

  function line(elements, doc, notes, env) {
    const errors = notes.filter((n) => n.severity === "error");
    const bad = new Map(errors.filter((n) => n.element > 0).map((n) => [n.element, n]));
    return `<div class="seg${env ? " env" : ""}${errors.length ? " flag" : ""}"><span class="body">${segHTML(elements, doc.separator || "*", doc.terminator, bad)}</span></div>` +
      notes.map(noteHTML).join("");
  }

  function x12(doc) {
    const notes = doc.findings || [];
    const placed = new Set();
    const take = (pred) => notes.filter((n, i) => {
      if (placed.has(i) || !pred(n)) return false;
      placed.add(i);
      return true;
    });
    const envLine = (elements) => {
      const id = elements[0];
      return line(elements, doc, take((n) => !n.segment && n.segmentId === id), true);
    };
    const header = (doc.header || []).map(envLine).join("");
    const body = (doc.segments || []).map((elements, i) => line(elements, doc, take((n) => n.segment === i + 1), false)).join("");
    const trailer = (doc.trailer || []).map(envLine).join("");
    const rest = notes.filter((_, i) => !placed.has(i)).map(noteHTML).join("");
    return `<div class="x12" aria-label="Transaction text">${header}${rest}${body}${trailer}</div>`;
  }

  function scenarioHTML(scenario) {
    const dot = (s) => (s === "passed" ? "ok" : s === "failed" ? "bad" : "");
    const steps = scenario.steps.map((s) => `<li><i class="dot ${dot(s.status)}"></i><span>${esc(s.text)}</span>${s.why ? `<span class="why">${esc(s.why)}</span>` : ""}</li>`).join("");
    return `
    <div class="section">
      <h4>Scenario</h4>
      <ul class="steps">${steps}</ul>
    </div>${scenario.replay ? `
    <div class="section">
      <h4>Run this case again</h4>
      <div class="replay"><code id="replayCmd">${esc(scenario.replay)}</code><button class="btn" id="copy" type="button">Copy</button></div>
    </div>` : ""}`;
  }

  Sim.renderInspector = (root, meta, ctx) => {
    const { row, message, partner, scenario } = ctx;
    const doc = message?.documents?.[row?.document ?? -1];
    if (!row || !doc) {
      meta.textContent = "";
      root.innerHTML = ctx.empty ? `<p class="empty">${esc(ctx.empty)}</p>` : "";
      return;
    }
    const toT = row.from === "partner";
    const answered = answeredBy(doc, message, partner);
    const answers = (doc.answers || []).map((a) => `${a.set} ${a.interchange} · ${a.title}`).join("; ");
    meta.textContent = Sim.clock(row.at);
    root.innerHTML = `
    <div class="insp-head">
      <div class="insp-title"><h3>${esc(doc.set)} · ${esc(doc.title)}</h3>${verdictChip(row, message, partner)}</div>
      <dl class="facts">
        <dt>From</dt><dd>${toT ? esc(partner.display) : "Trenova"} → ${toT ? "Trenova" : esc(partner.display)}</dd>
        <dt>Control</dt><dd>ISA ${esc(doc.interchange)}${doc.set === "TA1" ? "" : ` · ST ${esc(doc.control)}`}</dd>
        <dt>Transport</dt><dd>${esc(transport(message))}</dd>
        ${answered ? `<dt>Answered by</dt><dd>${esc(answered)}</dd>` : ""}
        ${answers ? `<dt>Answers</dt><dd>${esc(answers)}</dd>` : ""}
        ${message.error ? `<dt>Error</dt><dd>${esc(message.error)}</dd>` : ""}
      </dl>
    </div>
    ${x12(doc)}
    ${scenario ? scenarioHTML(scenario) : ""}`;
    const btn = root.querySelector("#copy");
    if (btn) btn.addEventListener("click", () => {
      const code = root.querySelector("#replayCmd");
      Promise.resolve().then(() => navigator.clipboard.writeText(code.textContent)).then(() => { btn.textContent = "Copied"; }, () => {
        const r = document.createRange(); r.selectNodeContents(code); const s = getSelection(); s.removeAllRanges(); s.addRange(r);
      });
    });
  };
})();
