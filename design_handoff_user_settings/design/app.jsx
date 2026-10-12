const US_KEY = "trenova-user-settings-v2";
const US_DEFAULTS = { name: "Maya Chen", title: "Dispatch lead", username: "maya.chen", email: "maya.chen@acmelogistics.com", phone: "", avatar: "auto", photo: false,
  status: "available", focusFor: "2h", backOn: "", delegate: "", awayNote: "",
  theme: "light", lang: "en-US", tz: "America/Chicago", timeFormat: "12", sidebar: "full", motion: "full",
  nx: nxDefaults(), pause: "off", deskMode: "urgent", emailMode: "urgent", digest: true, sound: true, quiet: { on: true, from: 19, to: 7, days: [1, 2, 3, 4, 5], crit: true },
  mfa: true, pwChanged: false };
const US_SECS = [["profile", "Profile", "user", "Who you are and where you are."], ["prefs", "Preferences", "sliders", "How Trenova looks and keeps time for you."], ["notify", "Notifications", "bell", "Choose what reaches you, where, and when. Every notification Trenova sends can be turned off here — except a few your organization requires."], ["security", "Security", "shield", "Your password, second factor and every place you’re signed in."]];
const US_INDEX = [
  ["Photo", "profile", "photo"], ["Display name", "profile", "name"], ["Job title", "profile", "name"], ["Phone", "profile", "name"], ["Email address", "profile", "name"], ["Availability · Focus · Away", "profile", "availability"], ["Hand off while away", "profile", "status"],
  ["Theme · dark mode", "prefs", "theme"], ["Language", "prefs", "language"], ["Time zone", "prefs", "timezone"], ["Time format · 24-hour", "prefs", "timeformat"], ["Sidebar", "prefs", "sidebar"], ["Motion · reduce", "prefs", "motion"],
  ["Pause notifications", "notify", "pause"], ["What you hear about", "notify", "matrix"], ["Email · digest", "notify", "digest"], ["Desktop notifications", "notify", "desktop"], ["Sound", "notify", "sound"], ["Quiet hours", "notify", "quiet"],
  ...NX_CATS.flatMap(c => c.ev.map(e => [e.t, "notify", "matrix", c.name])),
  ["Password", "security", "password"], ["Two-factor authentication", "security", "2fa"], ["Recovery codes", "security", "2fa"], ["Sessions · sign out", "security", "sessions"], ["Sign-in activity", "security", "activity"],
];

function useStore() {
  const [s, setS] = React.useState(() => { try { return { ...US_DEFAULTS, ...JSON.parse(localStorage.getItem(US_KEY) || "{}") }; } catch (e) { return US_DEFAULTS; } });
  React.useEffect(() => { try { localStorage.setItem(US_KEY, JSON.stringify(s)); } catch (e) {} }, [s]);
  return [s, setS];
}

function SettingsDialog({ onClose, closing, initial }) {
  const [s, setS] = useStore();
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  const [undo, setUndo] = React.useState(null);
  const [pulse, setPulse] = React.useState(0);
  const [msg, setMsg] = React.useState(null);
  const [nps, setNps] = React.useState([]);
  const [perm, setPerm] = React.useState(t.perm);
  const [sec, setSec] = React.useState("profile");
  const [q, setQ] = React.useState(""); const [qi, setQi] = React.useState(0);
  const main = React.useRef(null);
  const now = useNow();
  React.useEffect(() => setPerm(t.perm), [t.perm]);
  const set = (k, v, typing) => {
    setS(p => { if (!typing) setUndo({ prev: p, k, at: Date.now() }); return { ...p, [k]: v }; });
    setPulse(x => x + 1);
  };
  React.useEffect(() => { if (!undo) return; const id = setTimeout(() => setUndo(null), 4200); return () => clearTimeout(id); }, [undo]);
  React.useEffect(() => { if (!msg) return; const id = setTimeout(() => setMsg(null), 2400); return () => clearTimeout(id); }, [msg]);
  const [pz, setPz] = React.useState(false);
  React.useEffect(() => { if (!pulse) return; setPz(true); const id = setTimeout(() => setPz(false), 900); return () => clearTimeout(id); }, [pulse]);
  const sysDark = window.matchMedia && matchMedia("(prefers-color-scheme: dark)").matches;
  const dark = s.theme === "dark" || (s.theme === "system" && sysDark);
  const preview = (c, e) => {
    const id = Math.random().toString(36).slice(2);
    setNps(n => [{ id, c, e }, ...n].slice(0, 3));
    setTimeout(() => setNps(n => n.map(x => x.id === id ? { ...x, out: true } : x)), 4600);
    setTimeout(() => setNps(n => n.filter(x => x.id !== id)), 4900);
  };
  const askPerm = () => { setTimeout(() => { setPerm("granted"); setMsg("Desktop notifications allowed"); }, 350); };
  const onScroll = () => {
    const el = main.current; let cur = US_SECS[0][0];
    US_SECS.forEach(([k]) => { const x = document.getElementById("sec-" + k); if (x && x.offsetTop - el.scrollTop < 160) cur = k; });
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 4) cur = US_SECS[US_SECS.length - 1][0];
    setSec(cur);
  };
  const jump = (secK, rowK) => {
    const el = main.current, target = document.getElementById(rowK ? "row-" + rowK : "sec-" + secK) || document.getElementById("sec-" + secK);
    if (!target) return;
    const y = target.getBoundingClientRect().top - el.getBoundingClientRect().top + el.scrollTop - (rowK ? 90 : 12);
    el.scrollTo({ top: y });
    if (rowK) { target.classList.remove("flash"); void target.offsetWidth; target.classList.add("flash"); }
    setQ("");
  };
  const ql = q.trim().toLowerCase();
  const res = ql ? US_INDEX.filter(([l, , , c]) => l.toLowerCase().includes(ql) || (c || "").toLowerCase().includes(ql)).slice(0, 8) : [];
  React.useEffect(() => {
    const k = e => { if ((e.metaKey || e.ctrlKey) && e.key === "k") { e.preventDefault(); document.getElementById("us-find").focus(); } };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, []);
  const hr = +new Intl.DateTimeFormat("en-US", { timeZone: s.tz, hour: "numeric", hour12: false }).format(now) % 24;
  const greet = hr < 5 ? "Working late" : hr < 12 ? "Good morning" : hr < 17 ? "Good afternoon" : "Good evening";
  const first = s.name.split(" ")[0] || "there";
  const stat = { available: "Available", focus: "Focusing", away: "Away" }[s.status];
  return (
    <div className={"us us-dlg" + (dark ? " dk" : "") + (s.motion === "reduced" ? " rm" : "") + (closing ? " out" : "")} role="dialog" aria-modal="true" aria-label="Settings">
      <header className="us-top">
        <span className="us-logo">T</span>
        <div className="us-crumb"><b>Settings</b><span className="us-over">over Shipments</span></div>
        <span className="sp"></span>
        <span className={"us-saved" + (pz ? " pulse" : "")}><i></i>{pz ? "Saved" : "All changes saved"}</span>
        <Button variant="ghost" size="icon-sm" aria-label="Close settings" title="Close · Esc" onClick={onClose}><UI n="x" s={15} /></Button>
      </header>
      <div className="us-body">
        <nav className="us-nav">
          <div className="us-find">
            <label><UI n="search" s={14} /><input id="us-find" value={q} onChange={e => { setQ(e.target.value); setQi(0); }} onKeyDown={e => { if (e.key === "ArrowDown") { e.preventDefault(); setQi(i => Math.min(res.length - 1, i + 1)); } if (e.key === "ArrowUp") { e.preventDefault(); setQi(i => Math.max(0, i - 1)); } if (e.key === "Enter" && res[qi]) jump(res[qi][1], res[qi][2]); if (e.key === "Escape") setQ(""); }} placeholder="Find a setting" /><span className="kbd">⌘K</span></label>
            {ql && <div className="us-res">{res.length ? res.map(([l, sk, rk, c], i) => <button key={l + i} className={i === qi ? "on" : ""} onMouseEnter={() => setQi(i)} onClick={() => jump(sk, rk)}><b>{l}</b><span>{US_SECS.find(x => x[0] === sk)[1]}{c ? " · " + c : ""}</span></button>) : <em>Nothing matches “{q}”</em>}</div>}
          </div>
          {US_SECS.map(([k, l, ic]) => (
            <button key={k} className={"us-ni" + (sec === k ? " on" : "")} onClick={() => jump(k)}><UI n={ic} s={15} />{l}
              {k === "notify" && (s.pause !== "off" ? <i className="dot" title="Paused"></i> : <span className="new">New</span>)}
            </button>
          ))}
          <div className="us-nf">
            <span><span className="kbd">⌘</span><span className="kbd">⇧</span><span className="kbd">S</span> opens Settings anywhere</span>
            <button onClick={() => setMsg("Signed out (demo)")}><UI n="out" s={14} />Log out</button>
          </div>
        </nav>
        <main className="us-main" ref={main} onScroll={onScroll}>
          <div className="us-col">
            <div className="us-hero">
              <Avatar s={s} size="lg"><button onClick={() => set("photo", !s.photo)} aria-label="Change photo"><UI n="camera" s={18} /></button></Avatar>
              <div className="us-hi">
                <h1>{greet}, <em>{first}</em>.</h1>
                <p><span>{s.title || "No title"} · Acme Logistics</span><span><UI n="clock" s={12} />{fmtTime(now, s.tz, s.timeFormat)} in {(US_TZ.find(z => z[0] === s.tz) || US_TZ[0])[2]}</span>{s.pause !== "off" && <span><UI n="bellOff" s={12} />Notifications paused</span>}</p>
              </div>
              <button className={"us-stat " + s.status} onClick={() => jump("profile", "availability")}><i></i>{stat}</button>
            </div>
            {US_SECS.map(([k, l, ic, d]) => (
              <section key={k} id={"sec-" + k} className="us-sec" data-screen-label={l}>
                <div className="us-sh"><div><h2>{l}</h2><p>{d}</p></div>{k === "notify" && s.pause !== "off" && <Badge tone="warning"><UI n="bellOff" s={11} />Paused</Badge>}</div>
                {k === "profile" && <ProfileSection s={s} set={set} toast={setMsg} />}
                {k === "prefs" && <PrefsSection s={s} set={set} />}
                {k === "notify" && <NotifySection s={s} set={set} preview={preview} perm={perm} askPerm={askPerm} />}
                {k === "security" && <SecuritySection s={s} set={set} toast={setMsg} />}
              </section>
            ))}
          </div>
        </main>
      </div>
      {(undo || msg) && <div className="svd" key={msg || undo.at}>
        {msg ? <span>{msg}</span> : <><span><UI n="check" s={13} w={2.2} /> Saved</span><button onClick={() => { setS(undo.prev); setUndo(null); }}><UI n="undo" s={12} />Undo</button></>}
      </div>}
      <div className="np-stack">{nps.map(({ id, c, e, out }) => (
        <div key={id} className={"np" + (out ? " out" : "")}>
          <span className={"np-i tone-" + c.tone}><UI n={c.icon} s={15} /></span>
          <div className="np-t"><small><span>Trenova · {c.name}</span><span>now</span>{e.sev === "crit" && <Badge tone="danger">Urgent</Badge>}</small><b>{e.t}</b><span>{e.msg}</span></div>
        </div>
      ))}</div>
      <TweaksPanel>
        <TweakSection label="Notifications" />
        <TweakRadio label="Desktop permission" value={t.perm} options={["default", "granted", "denied"]} onChange={v => setTweak("perm", v)} />
        <TweakSection label="State" />
        <TweakRadio label="Theme" value={s.theme} options={["light", "dark", "system"]} onChange={v => set("theme", v)} />
        <TweakButton label="Reset all settings" secondary onClick={() => setS(US_DEFAULTS)} />
      </TweaksPanel>
    </div>
  );
}
function HostApp() {
  const [open, setOpen] = React.useState(() => localStorage.getItem("trenova-settings-open") !== "0");
  const [closing, setClosing] = React.useState(false);
  const show = v => { localStorage.setItem("trenova-settings-open", v ? "1" : "0"); setOpen(v); };
  const close = () => { setClosing(true); setTimeout(() => { setClosing(false); show(false); }, 180); };
  React.useEffect(() => {
    const k = e => {
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === "s") { e.preventDefault(); open ? close() : show(true); }
      if (e.key === "Escape" && open && !document.querySelector(".pop")) close();
    };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, [open]);
  let st = {}; try { st = { ...US_DEFAULTS, ...JSON.parse(localStorage.getItem(US_KEY) || "{}") }; } catch (e) { st = US_DEFAULTS; }
  const dark = st.theme === "dark" || (st.theme === "system" && window.matchMedia && matchMedia("(prefers-color-scheme: dark)").matches);
  return (
    <>
      <BackdropApp s={st} dark={dark} dim={open} onOpen={() => show(true)} />
      {open && <SettingsDialog onClose={close} closing={closing} />}
    </>
  );
}
ReactDOM.createRoot(document.getElementById("root")).render(<HostApp />);
