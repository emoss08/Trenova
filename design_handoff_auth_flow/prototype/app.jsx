const { useState, useRef, useLayoutEffect, useEffect, useCallback, useMemo } = React;

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "theme": "dark",
  "motion": true,
  "sidePanel": true,
  "showSso": false,
  "showAudience": true,
  "orgCount": 2
}/*EDITMODE-END*/;

const ORGS = [
  { id: "1", name: "Trenova Logistics", loc: "Los Angeles, California", code: "TRL", current: true },
  { id: "2", name: "Trenova Transportation", loc: "Los Angeles, California", code: "TRT" },
  { id: "3", name: "Meridian Freight Group", loc: "Dallas, Texas", code: "MFG" },
];
const ROLES = [
  { id: "r1", name: "Organization Administrator", desc: "Full access to every resource", tag: "System", perms: 214, on: true },
  { id: "r2", name: "Dispatch Supervisor", desc: "Loads, lanes, driver availability", tag: "Custom", perms: 68 },
  { id: "r3", name: "Billing Analyst", desc: "Invoicing, AR aging, settlements", tag: "Custom", perms: 41 },
];

const I = {
  building: (p) => <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" {...p}><rect x="5" y="3" width="14" height="18" rx="1.5"/><path d="M9 7h2M13 7h2M9 11h2M13 11h2M9 15h2M13 15h2"/></svg>,
  truck: (p) => <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" {...p}><path d="M3 6h11v10H3z"/><path d="M14 9h3.5l2.5 3v4h-6"/><circle cx="7" cy="18" r="1.6"/><circle cx="17" cy="18" r="1.6"/></svg>,
  shield: (p) => <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" {...p}><path d="M12 3l7 3v5.5c0 4.2-2.9 7.7-7 8.5-4.1-.8-7-4.3-7-8.5V6z"/></svg>,
  check: (p) => <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" {...p}><path d="M5 12.5l4.5 4.5L19 7"/></svg>,
  arrow: (p) => <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" {...p}><path d="M5 12h14M13 6l6 6-6 6"/></svg>,
  back: (p) => <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" {...p}><path d="M19 12H5M11 6l-6 6 6 6"/></svg>,
};

const LANES = [
  ["LAX", "PHX", "in transit"], ["DAL", "ATL", "loading"], ["CHI", "DET", "in transit"],
  ["SEA", "SLC", "delivered"], ["MEM", "OKC", "in transit"], ["NWK", "CLT", "dispatched"],
];
function Lanes() {
  const row = (rev, off) => (
    <div className={"lane-track" + (rev ? " rev" : "")}>
      {[...LANES.slice(off), ...LANES.slice(0, off), ...LANES.slice(off), ...LANES.slice(0, off)].map(([a, b, s], i) => (
        <span className="lane" key={i}><em>{a}</em>→<em>{b}</em><i />{s}</span>
      ))}
    </div>
  );
  return <div className="lanes">{row(false, 0)}{row(true, 3)}</div>;
}

function Morph({ children, deps }) {
  const outer = useRef(null), inner = useRef(null);
  useLayoutEffect(() => {
    const o = outer.current, n = inner.current;
    if (!o || !n) return;
    const set = () => { o.style.height = n.offsetHeight + "px"; };
    set();
    const ro = new ResizeObserver(set); ro.observe(n);
    return () => ro.disconnect();
  }, [deps]);
  return <div ref={outer} className="card-h"><div ref={inner}>{children}</div></div>;
}

/* counts up when the target changes */
function Tally({ value }) {
  const [n, setN] = useState(value);
  const from = useRef(value);
  useEffect(() => {
    const start = performance.now(), a = from.current, b = value;
    if (a === b) return;
    let raf;
    const tick = (t) => {
      const p = Math.min(1, (t - start) / 420);
      setN(Math.round(a + (b - a) * (1 - Math.pow(1 - p, 3))));
      if (p < 1) raf = requestAnimationFrame(tick); else from.current = b;
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [value]);
  return <>{n.toLocaleString()}</>;
}

function Receipt({ email, org, roles, step, session }) {
  const rows = [
    { k: "Identity", v: step === "login" ? null : email },
    { k: "Workspace", v: ["role", "done"].includes(step) ? org?.name : null },
    { k: "Roles", v: step === "done" ? roles.map(r => r.name).join(", ") : null },
    { k: "Session", v: step === "done" ? session : null },
  ];
  return (
    <div className="receipt">
      <div className="receipt-h">
        <span className="rcap">Credential</span>
        <span className="rcap">{step === "done" ? "Issued" : "Assembling"}</span>
      </div>
      {rows.map(r => (
        <div className="rrow" key={r.k}>
          <span className="k">{r.k}</span>
          {r.v ? <span className="v fill" key={r.v}>{r.v}</span> : <span className="v pending" />}
        </div>
      ))}
      {step === "done" && <span className="stamp">Authorized</span>}
    </div>
  );
}

function Segmented({ value, onChange }) {
  const wrap = useRef(null);
  const [k, setK] = useState({ x: 0, w: 0 });
  useLayoutEffect(() => {
    const el = wrap.current?.querySelector(`[data-v="${value}"]`);
    if (el) setK({ x: el.offsetLeft, w: el.offsetWidth });
  }, [value]);
  return (
    <div className="seg" ref={wrap}>
      <span className="knob" style={{ transform: `translateX(${k.x - 3}px)`, width: k.w }} />
      {[["office", "Office", I.building], ["driver", "Driver", I.truck]].map(([v, label, Icon]) => (
        <button key={v} data-v={v} className={value === v ? "on" : ""} onClick={() => onChange(v)}><Icon />{label}</button>
      ))}
    </div>
  );
}

function Option({ on, onClick, avatar, name, meta, chip, shortcut, index }) {
  return (
    <button type="button" className={"opt" + (on ? " on" : "")} onClick={onClick} aria-pressed={on} data-i={index}>
      <span className="av">{avatar}</span>
      <span style={{ minWidth: 0, flex: 1 }}>
        <span className="nm">{name}</span>
        <span className="mt">{meta}</span>
      </span>
      {chip && <span className="chip">{chip}</span>}
      {shortcut && <span className="sk">{shortcut}</span>}
      <span className="mk"><I.check /></span>
    </button>
  );
}

function App() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  const [step, setStep] = useState("login");
  const [audience, setAudience] = useState("office");
  const [email, setEmail] = useState("admin@trenova.app");
  const [pw, setPw] = useState("supersecret");
  const [reveal, setReveal] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [orgId, setOrgId] = useState("1");
  const [roleIds, setRoleIds] = useState(["r1"]);
  const session = useMemo(() => "sx_" + Math.random().toString(16).slice(2, 10), []);
  const [loads, setLoads] = useState(12480);
  useEffect(() => {
    const id = setInterval(() => setLoads(n => n + Math.floor(Math.random() * 7) - 2), 3200);
    return () => clearInterval(id);
  }, []);

  const orgs = ORGS.slice(0, t.orgCount);
  const org = orgs.find(o => o.id === orgId) || orgs[0];
  const roles = ROLES.filter(r => roleIds.includes(r.id));
  const perms = roles.reduce((s, r) => s + r.perms, 0);

  useEffect(() => { if (!orgs.some(o => o.id === orgId)) setOrgId(orgs[0].id); }, [t.orgCount]);
  useEffect(() => { document.body.className = (t.theme === "light" ? "theme-light " : "") + (t.motion ? "" : "motion-off"); }, [t.theme, t.motion]);

  const go = (next, delay = 700) => { setBusy(true); setTimeout(() => { setBusy(false); setStep(next); }, delay); };
  const signIn = (e) => {
    e?.preventDefault();
    if (!email.includes("@")) return setErr("Enter a valid work email address.");
    if (pw.length < 6) return setErr("Password must be at least 6 characters.");
    setErr(""); go(t.orgCount > 1 ? "org" : "role");
  };
  const restart = () => { setStep("login"); setRoleIds(["r1"]); setErr(""); };

  /* ⌘1..3 picks a row, ⌘↵ advances */
  const listRef = useRef(null);
  const advance = useCallback(() => {
    if (step === "org") go("role");
    else if (step === "role" && roleIds.length) go("done", 900);
  }, [step, roleIds.length]);
  useEffect(() => {
    if (step !== "org" && step !== "role") return;
    const onKey = (e) => {
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); return advance(); }
      if ((e.metaKey || e.ctrlKey) && /^[1-3]$/.test(e.key)) {
        e.preventDefault();
        const i = +e.key - 1;
        if (step === "org") { if (orgs[i]) setOrgId(orgs[i].id); }
        else if (ROLES[i]) setRoleIds(s => s.includes(ROLES[i].id) ? s.filter(x => x !== ROLES[i].id) : [...s, ROLES[i].id]);
      }
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        const items = [...(listRef.current?.querySelectorAll(".opt") || [])];
        if (!items.length) return;
        e.preventDefault();
        const cur = items.indexOf(document.activeElement);
        const n = e.key === "ArrowDown" ? (cur + 1) % items.length : (cur - 1 + items.length) % items.length;
        items[n].focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [step, orgs.length, advance]);

  const Btn = ({ children, ...p }) => <button className="btn" {...p}><span className="lbl" key={String(children)}>{children}</span></button>;

  let body;
  if (step === "login") {
    body = (
      <div className="pad enter" key="login">
        <div className="crumbs">
          <span className="count m">01 / {t.orgCount > 1 ? "03" : "02"}</span>
          <span className="count m">Secure sign-in</span>
        </div>
        <h1 className="title">{audience === "driver" ? "Driver sign-in" : "Welcome back"}</h1>
        <p className="sub">{audience === "driver" ? "Dash is where drivers see loads and pay." : <>Don't have an account yet? <a href="#" onClick={e => e.preventDefault()}>Create an account</a></>}</p>
        {t.showAudience ? <Segmented value={audience} onChange={setAudience} /> : <div style={{ height: 18 }} />}
        {audience === "driver" ? (
          <div className="stack" style={{ gap: 14 }}>
            <p style={{ margin: 0, fontSize: 12.5, color: "var(--muted-fg)" }}>Loads, settlement statements and pay — built for the phone.</p>
            <Btn onClick={() => go("org")} disabled={busy}>{busy ? <><span className="spin" />Redirecting</> : <>Continue to Dash <I.arrow /></>}</Btn>
            <p style={{ margin: 0, fontSize: 11.5, color: "var(--subtle-fg)" }}>First time here? Use the invitation link your carrier sent you.</p>
          </div>
        ) : (
          <form className="stack" style={{ gap: 14 }} onSubmit={signIn}>
            {t.showSso && (<>
              <div className="stack">
                <button type="button" className="sso"><img src="entra.svg" alt="" />Continue with Microsoft Entra</button>
                <button type="button" className="sso"><img className="inv" src="okta.svg" alt="" />Continue with Okta</button>
              </div>
              <div className="rule">or</div>
            </>)}
            <div className="field">
              <label className="lab" htmlFor="em">Email address <i>*</i></label>
              <div className={"ctl" + (err && !email.includes("@") ? " bad" : "")}>
                <input id="em" value={email} autoComplete="username" placeholder="name@work-email.com" onChange={e => { setEmail(e.target.value); setErr(""); }} />
              </div>
            </div>
            <div className="field">
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
                <label className="lab" htmlFor="pw">Password <i>*</i></label>
                <a href="#" style={{ fontSize: 11.5, color: "var(--muted-fg)" }} onClick={e => e.preventDefault()}>Forgot?</a>
              </div>
              <div className={"ctl" + (err && pw.length < 6 ? " bad" : "")}>
                <input id="pw" type={reveal ? "text" : "password"} value={pw} autoComplete="current-password" placeholder="••••••••" onChange={e => { setPw(e.target.value); setErr(""); }} />
                <button type="button" className="ghosty" onClick={() => setReveal(v => !v)}>{reveal ? "hide" : "show"}</button>
              </div>
            </div>
            {err && <div className="err">{err}</div>}
            <Btn type="submit" disabled={busy}>{busy ? <><span className="spin" />Verifying credentials</> : "Sign in"}</Btn>
          </form>
        )}
      </div>
    );
  } else if (step === "org") {
    body = (
      <div className="pad enter" key="org">
        <div className="crumbs">
          <span className="count m">02 / 03</span>
          <span className="count m">{orgs.length} available</span>
        </div>
        <h1 className="title">Select organization</h1>
        <p className="sub">Choose the workspace for this session.</p>
        <div className="stack" style={{ margin: "16px 0 14px" }} ref={listRef}>
          {orgs.map((o, i) => (
            <Option key={o.id} index={i} on={orgId === o.id} onClick={() => setOrgId(o.id)}
              avatar={o.code} name={o.name} meta={o.loc} chip={o.current ? "Current" : null} shortcut={`⌘${i + 1}`} />
          ))}
        </div>
        <Btn onClick={() => go("role")} disabled={busy}>{busy ? <><span className="spin" />Opening workspace</> : "Continue"}</Btn>
        <div className="tray">
          <button className="btn-link" onClick={restart}><I.back />Back</button>
          <span style={{ display: "flex", gap: 6, alignItems: "center" }}><span className="kbd">↑↓</span> move <span className="kbd">⌘↵</span> continue</span>
        </div>
      </div>
    );
  } else if (step === "role") {
    body = (
      <div className="pad enter" key="role">
        <div className="crumbs">
          <span className="count m">{t.orgCount > 1 ? "03 / 03" : "02 / 02"}</span>
          <span className="count m"><Tally value={perms} /> permissions</span>
        </div>
        <h1 className="title">Select active roles</h1>
        <p className="sub">Scope this session at {org?.name}. You can switch later without signing out.</p>
        <div className="stack" style={{ margin: "16px 0 14px" }} ref={listRef}>
          {ROLES.map((r, i) => (
            <Option key={r.id} index={i} on={roleIds.includes(r.id)}
              onClick={() => setRoleIds(s => s.includes(r.id) ? s.filter(x => x !== r.id) : [...s, r.id])}
              avatar={<I.shield />} name={r.name} meta={r.desc} chip={r.tag} shortcut={`⌘${i + 1}`} />
          ))}
        </div>
        <Btn onClick={() => go("done", 900)} disabled={busy || !roleIds.length}>
          {busy ? <><span className="spin" />Issuing credential</> : roleIds.length ? `Activate ${roleIds.length} role${roleIds.length > 1 ? "s" : ""}` : "Select at least one role"}
        </Btn>
        <div className="tray">
          <button className="btn-link" onClick={() => setStep(t.orgCount > 1 ? "org" : "login")}><I.back />Back</button>
          <span style={{ display: "flex", gap: 6, alignItems: "center" }}><span className="kbd">⌘1–3</span> toggle <span className="kbd">⌘↵</span> activate</span>
        </div>
      </div>
    );
  } else {
    body = (
      <div className="done enter" key="done">
        <span className="ring"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path className="draw" d="M5 12.5l4.5 4.5L19 7" /></svg></span>
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: 14, fontWeight: 550, letterSpacing: "-0.01em" }}>Entering {org?.name}</div>
          <div className="m" style={{ fontSize: 11.5, color: "var(--subtle-fg)", marginTop: 6 }}>{roles.length} role{roles.length > 1 ? "s" : ""} · {perms} permissions</div>
        </div>
        <div className="bar"><i /></div>
        <button className="btn-link" onClick={restart}>Run the flow again</button>
      </div>
    );
  }

  const brand = <><img src="logo.webp" alt="Trenova" style={{ width: 24, height: 24, objectFit: "contain" }} /><span className="wordmark">Trenova</span></>;

  return (
    <div className={"shell" + (t.sidePanel ? "" : " solo")}>
      {t.sidePanel && (
        <aside className="aside" data-step={step}>
          <div className="weave" /><div className="aura" />{t.motion && <div className="scan" />}
          <div className="brandline"><img src="logo.webp" alt="Trenova" /><span className="wordmark">Trenova</span><span className="env">Enterprise</span></div>
          <div style={{ position: "relative", display: "flex", flexDirection: "column", gap: 24 }}>
            <h2 className="pitch">Sign in once. The network never stopped moving.</h2>
            <div className="metrics">
              <div><b><Tally value={loads} /></b><span>loads in motion</span></div>
              <div><b>98.6%</b><span>on-time this week</span></div>
            </div>
            <Lanes />
            <Receipt email={email} org={org} roles={roles} step={step} session={session} />
          </div>
          <div className="foot">
            <span className="live"><i />Network operational</span>
            <span>us-west-2</span>
            <span>v4.12.0</span>
          </div>
        </aside>
      )}
      <main className="main">
        <div className="col">
          <div className="mobile-brand">{brand}</div>
          <div className={"card" + (busy ? " busy" : "")}>
            <span className="beamring" />
            <div className="inner"><Morph deps={step + audience + err + roleIds.length + busy + t.showSso + t.showAudience + t.orgCount}>{body}</Morph></div>
          </div>
          <p className="legal">By continuing you agree to our <a href="#" onClick={e => e.preventDefault()}>Terms of Service</a> and <a href="#" onClick={e => e.preventDefault()}>Privacy Policy</a>.</p>
        </div>
      </main>
      <TweaksPanel>
        <TweakSection label="Appearance" />
        <TweakRadio label="Theme" value={t.theme} options={["dark", "light"]} onChange={v => setTweak("theme", v)} />
        <TweakToggle label="Side panel" value={t.sidePanel} onChange={v => setTweak("sidePanel", v)} />
        <TweakToggle label="Motion" value={t.motion} onChange={v => setTweak("motion", v)} />
        <TweakSection label="Flow" />
        <TweakToggle label="Office / Driver tabs" value={t.showAudience} onChange={v => setTweak("showAudience", v)} />
        <TweakToggle label="SSO providers" value={t.showSso} onChange={v => setTweak("showSso", v)} />
        <TweakSlider label="Organizations" value={t.orgCount} min={1} max={3} step={1} onChange={v => setTweak("orgCount", v)} />
      </TweaksPanel>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
