"use strict";

(() => {
  const Sim = (window.Sim = window.Sim || {});

  Sim.esc = (s) => String(s ?? "").replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));

  Sim.clock = (iso) => {
    if (!iso) return "";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime()) || d.getFullYear() < 2000) return "";
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
  };

  const esc = Sim.esc;
  const css = (v) => `var(--${v})`;

  function mark(x, y, s) {
    const c = s === "ok" ? "var(--ok)" : "var(--bad)";
    const glyph = s === "ok"
      ? `<path d="M${x - 4},${y} l3,3 l6,-7" fill="none" stroke="${c}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>`
      : `<path d="M${x - 4},${y - 4} l8,8 M${x + 4},${y - 4} l-8,8" fill="none" stroke="${c}" stroke-width="1.8" stroke-linecap="round"/>`;
    return `<circle cx="${x}" cy="${y}" r="9" fill="${s === "ok" ? "var(--ok-bg)" : "var(--bad-bg)"}"/>${glyph}`;
  }

  function eventColors(tone) {
    if (tone === "hold") return { bg: css("hold-bg"), text: css("hold"), dot: css("hold") };
    if (tone === "bad") return { bg: css("bad-bg"), text: css("bad"), dot: css("bad") };
    return { bg: css("rule-2"), text: css("ink-2"), dot: css("trenova") };
  }

  Sim.drawLadder = (svg, rows, selected) => {
    const W = svg.clientWidth || 640;
    const xcol = 120, pad = 12;
    const times = rows.map((m) => Sim.clock(m.at));
    const widest = Math.max(0, ...times.map((t) => t.length));
    const L = pad + 6 + (widest ? Math.ceil(widest * 6.4) + 8 : 0), R = W - xcol - pad - 6, X = W - xcol / 2 - pad;
    const rowH = 46;
    let y = 18;
    const out = [];
    out.push(`<line x1="${L}" y1="0" x2="${L}" y2="${rows.length * rowH + 40}" stroke="${css("partner")}" stroke-width="2" opacity=".35"/>`);
    out.push(`<line x1="${R}" y1="0" x2="${R}" y2="${rows.length * rowH + 40}" stroke="${css("trenova")}" stroke-width="2" opacity=".35"/>`);
    out.push(`<defs>
    <marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="context-stroke"/></marker>
  </defs>`);
    rows.forEach((m, index) => {
      if (m.gap) {
        out.push(`<g><line x1="${pad}" y1="${y - 6}" x2="${W - pad}" y2="${y - 6}" stroke="${css("ink-3")}" stroke-dasharray="2 4"/>
        <text x="${(L + R) / 2}" y="${y - 10}" text-anchor="middle" font-size="10.5" fill="${css("ink-3")}">${esc(m.gap)}</text></g>`);
        y += 14;
      }
      if (m.kind === "gap") return;
      const t = times[index];
      if (m.kind === "event") {
        const tone = eventColors(m.tone);
        const label = m.label + (m.detail ? ` · ${m.detail}` : "");
        const w = Math.max(40, Math.min(R - L - 40, label.length * 6.6 + 18));
        out.push(`<g role="listitem" aria-label="${esc(t + " " + label + (m.who ? " by " + m.who : ""))}">
        <text x="${pad}" y="${y + 14}" font-size="10.5" fill="${css("ink-3")}">${t}</text>
        <rect x="${R - w - 8}" y="${y + 2}" width="${w}" height="20" rx="3" fill="${tone.bg}"/>
        <text x="${R - 16}" y="${y + 16}" text-anchor="end" font-size="11" fill="${tone.text}" font-family="Geist, Inter, system-ui, sans-serif">${esc(label)}</text>
        <circle cx="${R}" cy="${y + 12}" r="3.5" fill="${tone.dot}"/>
        ${m.expect ? mark(X, y + 12, m.expect) : ""}
      </g>`);
        y += 30;
        return;
      }
      const toT = m.from === "partner";
      const x1 = toT ? L : R, x2 = toT ? R : L;
      const y1 = y + 10, y2 = y + 26;
      const isAck = !!m.ack;
      const bad = m.tone === "bad";
      const col = bad ? css("bad") : toT ? css("partner") : css("trenova");
      const label = `${m.set} · ${m.title}`;
      out.push(`<g class="msg${m.number === selected ? " sel" : ""}" data-m="${m.number}" tabindex="0" role="listitem"${m.verdict === "scheduled" ? ' opacity=".6"' : ""} aria-label="${esc(t + " " + (toT ? "partner to Trenova " : "Trenova to partner ") + label)}">
      <rect class="hit" x="${pad - 6}" y="${y}" width="${W - xcol - pad}" height="${rowH - 6}" rx="4" fill="transparent"/>
      <text x="${pad}" y="${y + 14}" font-size="10.5" fill="${css("ink-3")}">${t}</text>
      <line class="arrow" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${col}" stroke-width="${isAck ? 1.2 : 1.7}" ${isAck ? 'stroke-dasharray="4 3"' : ""} marker-end="url(#ah)"/>
      <text x="${(L + R) / 2}" y="${(y1 + y2) / 2 - 5}" text-anchor="middle" font-size="11.5" fill="${css("ink")}" font-weight="${isAck ? 400 : 600}">${esc(label)}</text>
      <text x="${(L + R) / 2}" y="${y2 + 12}" text-anchor="middle" font-size="10" fill="${css("ink-3")}">ctrl ${esc(m.control)}${m.note ? "  ·  " : ""}${esc(m.note || "")}</text>
      ${m.expect ? mark(X, y + 18, m.expect) : ""}
    </g>`);
      y += rowH;
    });
    svg.setAttribute("viewBox", `0 0 ${W} ${y + 10}`);
    svg.setAttribute("height", y + 10);
    svg.innerHTML = out.join("");
    return L;
  };
})();
