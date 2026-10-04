const SETTINGS_DEFAULTS = { theme: "system", width: "default", text: "medium", celebrate: "confetti", ring: "on", refs: "on", send: "enter", start: "today", motion: "full",
  agent: "ag_billing", sharePage: "on", mentions: "on", slash: "on", mic: "on", presets: "on",
  drop: "page", scanDevice: "dev1", scanProfile: "Organization default", autoOpen: "off", artNotify: "on" };
const SETTINGS_KEY = "trenova-desk-settings";

function useDeskSettings() {
  const [s, setS] = React.useState(() => {
    try { return { ...SETTINGS_DEFAULTS, ...JSON.parse(localStorage.getItem(SETTINGS_KEY) || "{}") }; } catch (e) { return SETTINGS_DEFAULTS; }
  });
  const set = (k, v) => setS(p => { const n = { ...p, [k]: v }; try { localStorage.setItem(SETTINGS_KEY, JSON.stringify(n)); } catch (e) {} return n; });
  return [s, set];
}

function Seg({ value, options, onChange }) {
  const i = Math.max(0, options.findIndex(o => o[0] === value));
  return (
    <div className="sx-seg" style={{ "--n": options.length, "--i": i }}>
      <span className="sx-kn"></span>
      {options.map(([v, l]) => <button key={v} className={value === v ? "on" : ""} onClick={() => onChange(v)}>{l}</button>)}
    </div>
  );
}

function WidthPreview({ w }) {
  const pct = { narrow: 46, default: 62, wide: 84 }[w];
  return (
    <div className="sx-wp">
      <span className="sx-wp-sb"></span>
      <span className="sx-wp-main"><span className="sx-wp-col" style={{ width: pct + "%" }}><i style={{ width: "58%", alignSelf: "flex-end" }}></i><i></i><i style={{ width: "82%" }}></i><i style={{ width: "64%" }}></i><b></b></span></span>
    </div>
  );
}

const SETTINGS_SECTIONS = [
  ["appearance", "Appearance", "eye"],
  ["conversation", "Conversation", "chat"],
  ["composer", "Composer", "plus"],
  ["files", "Files & scanning", "copy"],
  ["artifacts", "Artifacts", "table"],
  ["agent", "Agent & approvals", "shield"],
];

function SettingsDialog({ s, set, onClose }) {
  const [sec, setSec] = React.useState("appearance");
  const [closing, setClosing] = React.useState(false);
  const close = () => { setClosing(true); setTimeout(onClose, 160); };
  React.useEffect(() => {
    const k = e => { if (e.key === "Escape") { e.stopPropagation(); close(); } };
    window.addEventListener("keydown", k, true); return () => window.removeEventListener("keydown", k, true);
  }, []);
  const Row = ({ t, d, children, wide }) => <div className={"sx-row" + (wide ? " wide" : "")}><div className="sx-l"><b>{t}</b><span>{d}</span></div><div className="sx-c">{children}</div></div>;
  return (
    <div className={"srch-wrap sx-wrap" + (closing ? " out" : "")} onMouseDown={e => { if (e.target === e.currentTarget) close(); }}>
      <div className="sx">
        <div className="sx-top"><h2>Settings</h2><button className="ib" onClick={close} title="Close  Esc"><Ic n="x" s={14} /></button></div>
        <nav className="sx-nav">
          {SETTINGS_SECTIONS.map(([k, l, ic]) => <button key={k} className={sec === k ? "on" : ""} onClick={() => setSec(k)}><Ic n={ic} s={14} />{l}</button>)}
          <span style={{ flex: 1 }}></span>
          <button className="sx-reset" onClick={() => Object.entries(SETTINGS_DEFAULTS).forEach(([k, v]) => set(k, v))}>Reset to defaults</button>
        </nav>
        <div className="sx-body" key={sec}>
          <div className="sx-head">{SETTINGS_SECTIONS.find(x => x[0] === sec)[1]}</div>
          {sec === "appearance" && <>
            <Row t="Theme" d="Match your system, or pick light or dark for Desk.">
              <Seg value={s.theme} onChange={v => set("theme", v)} options={[["system", "System"], ["light", "Light"], ["dark", "Dark"]]} />
            </Row>
            <Row t="Conversation width" d="How wide messages run. Wide fits more of a table or long reply per line." wide>
              <div className="sx-wopts">
                {[["narrow", "Narrow"], ["default", "Default"], ["wide", "Wide"]].map(([v, l]) => (
                  <button key={v} className={"sx-wo" + (s.width === v ? " on" : "")} onClick={() => set("width", v)}><WidthPreview w={v} /><span>{l}</span></button>
                ))}
              </div>
            </Row>
            <Row t="Motion" d="Reduced turns off confetti, the working border, typing effects and shimmer.">
              <Seg value={s.motion} onChange={v => set("motion", v)} options={[["full", "Full"], ["reduced", "Reduced"]]} />
            </Row>
            <Row t="Text size" d="Size of message text in the conversation.">
              <Seg value={s.text} onChange={v => set("text", v)} options={[["small", "Small"], ["medium", "Medium"], ["large", "Large"]]} />
            </Row>
          </>}
          {sec === "conversation" && <>
            <Row t="Send with" d="Which keys send a message. The other inserts a new line.">
              <Seg value={s.send} onChange={v => set("send", v)} options={[["enter", "Enter"], ["mod", "⌘ Enter"]]} />
            </Row>
            <Row t="Source numbers" d="Small numbers in replies that show which tool call each fact came from.">
              <Seg value={s.refs} onChange={v => set("refs", v)} options={[["on", "Show"], ["off", "Hide"]]} />
            </Row>
            <Row t="Open Desk to" d="What you see when you open Desk.">
              <Seg value={s.start} onChange={v => set("start", v)} options={[["today", "Today"], ["last", "Last conversation"]]} />
            </Row>
          </>}
          {sec === "composer" && <>
            <Row t="Start new conversations with" d="The agent the composer picks when you start fresh. You can still switch per message.">
              <select className="sx-select" value={s.agent} onChange={e => set("agent", e.target.value)}>{(window.AGENTS || []).map(a => <option key={a.id} value={a.id}>{a.name}</option>)}</select>
            </Row>
            <Row t="Share the page you're on" d="Send the current page with each message so the agent knows what you're looking at.">
              <Seg value={s.sharePage} onChange={v => set("sharePage", v)} options={[["on", "By default"], ["off", "Ask me"]]} />
            </Row>
            <Row t="@ mentions" d="Type @ to reference a shipment, customer, invoice, driver or carrier.">
              <Seg value={s.mentions} onChange={v => set("mentions", v)} options={[["on", "On"], ["off", "Off"]]} />
            </Row>
            <Row t="Slash commands" d="Type / for /status, /quote, /report and /explain.">
              <Seg value={s.slash} onChange={v => set("slash", v)} options={[["on", "On"], ["off", "Off"]]} />
            </Row>
            <Row t="Dictation" d="Show the microphone in the composer.">
              <Seg value={s.mic} onChange={v => set("mic", v)} options={[["on", "Show"], ["off", "Hide"]]} />
            </Row>
            <Row t="Suggested questions" d="Type out example questions in the empty composer on Today.">
              <Seg value={s.presets} onChange={v => set("presets", v)} options={[["on", "On"], ["off", "Off"]]} />
            </Row>
          </>}
          {sec === "files" && <>
            <Row t="Drop files" d="Where dropping a file attaches it to your message.">
              <Seg value={s.drop} onChange={v => set("drop", v)} options={[["page", "Anywhere"], ["composer", "On the composer"]]} />
            </Row>
            <Row t="Scan with" d="The computer Scan from Capture starts on. Only your own paired computers are listed.">
              <select className="sx-select" value={s.scanDevice} onChange={e => set("scanDevice", e.target.value)}><option value="dev1">AVERY-LAPTOP</option><option value="dev2">FRONT-DESK-PC</option><option value="ask">Ask each time</option></select>
            </Row>
            <Row t="Scan settings" d="The scan profile to start from. Your admin manages profiles in Scanning and printing.">
              <select className="sx-select" value={s.scanProfile} onChange={e => set("scanProfile", e.target.value)}>{["Organization default", "BOL · 300 dpi · grayscale · both sides", "POD · color · split on blank page"].map(p => <option key={p}>{p}</option>)}</select>
            </Row>
          </>}
          {sec === "artifacts" && <>
            <Row t="Open new artifacts" d="When an agent makes a table, record or draft.">
              <Seg value={s.autoOpen} onChange={v => set("autoOpen", v)} options={[["on", "Right away"], ["off", "When I click"]]} />
            </Row>
            <Row t="New artifact dot" d="Mark Workspace in the top bar when something new arrives while it's closed.">
              <Seg value={s.artNotify} onChange={v => set("artNotify", v)} options={[["on", "Show"], ["off", "Hide"]]} />
            </Row>
          </>}
          {sec === "agent" && <>
            <Row t="Approval celebration" d="What plays when you approve a proposed change.">
              <Seg value={s.celebrate} onChange={v => set("celebrate", v)} options={[["confetti", "Confetti"], ["subtle", "Subtle"], ["off", "Off"]]} />
            </Row>
            <Row t="Working border" d="The light that travels around the composer while an agent works.">
              <Seg value={s.ring} onChange={v => set("ring", v)} options={[["on", "Animated"], ["off", "Off"]]} />
            </Row>
          </>}
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { useDeskSettings, SettingsDialog });
