const CP_DEFAULTS = /*EDITMODE-BEGIN*/{
  "theme": "dark"
}/*EDITMODE-END*/;

const NAV = [["desk", "Desk"], ["inbox", "Triage"], ["truck", "Shipments"], ["dollar", "Billing"], ["building", "Customers"], ["users", "Workers"], ["trailer", "Equipment"], ["chart", "Analytics"], ["gear", "Administration"]];

function App() {
  const [t, setTweak] = useTweaks(CP_DEFAULTS);
  const [open, setOpen] = useState(true);
  const [nonce, setNonce] = useState(0);
  const [page, setPage] = useState({ title: "Shipments", crumb: ["Shipment management"] });
  const [sb, setSb] = useState(true);
  const [toast, setToast] = useState(null);
  const tt = useRef(0);
  const sysDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
  const dark = t.theme === "dark" || (t.theme === "system" && sysDark);

  const appToast = (m) => { clearTimeout(tt.current); setToast(m); tt.current = setTimeout(() => setToast(null), 2400); };
  const openPal = () => { setNonce((n) => n + 1); setOpen(true); };

  useEffect(() => {
    const h = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k" && !open) { e.preventDefault(); openPal(); }
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "b" && !open) { e.preventDefault(); setSb((s) => !s); }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [open]);

  const navigate = (title, crumb, q) => { setPage({ title, crumb: crumb || [] }); appToast(q ? `Opened “${q}” in the Desk` : `Opened ${title}`); };
  const active = NAV.find(([, l]) => page.crumb.concat(page.title).some((c) => c.toLowerCase().includes(l.toLowerCase().slice(0, 5)))) || NAV[2];

  return (
    <div className={"cp" + (dark ? " dk" : "")}>
      <div className={"app" + (sb ? "" : " nosb")}>
        <nav className="sb" aria-label="Main">
          <div className="sb-h"><img src="logo.png" alt=""></img>Trenova</div>
          {NAV.map(([i, l]) => <div key={l} className={"sb-i" + (active[1] === l ? " on" : "")}><Ic n={i} s={15}></Ic>{l}</div>)}
        </nav>
        <div className="mn">
          <header className="tp">
            <div className="cr">{page.crumb.map((c, i) => <React.Fragment key={i}><span>{c}</span><Ic n="chevR" s={12}></Ic></React.Fragment>)}<b>{page.title}</b></div>
            <span className="sp"></span>
            <button className="trig" onClick={openPal}><Ic n="search" s={14}></Ic>Search or ask Nova…<span className="sp"></span><Keys k={[MOD, "K"]}></Keys></button>
          </header>
          <main className="pg">
            <h1>{page.title}</h1>
            <div className="sk">{[0.32, 0.5, 0.41, 0.28, 0.46, 0.36, 0.4].map((w, i) => <div key={i}><i style={{ width: 72 }}></i><i style={{ width: w * 100 + "%" }}></i><span className="sp"></span><i style={{ width: 56 }}></i></div>)}</div>
          </main>
        </div>
      </div>
      {open && <Palette key={nonce} theme={t.theme} onTheme={(v) => setTweak("theme", v)} onClose={() => setOpen(false)} onNavigate={navigate} onToggleSidebar={() => setSb((s) => !s)} appToast={appToast}></Palette>}
      {toast && <div className="app-t" key={toast}><Ic n="check" s={14} w={2}></Ic>{toast}</div>}
      <TweaksPanel>
        <TweakSection label="Appearance"></TweakSection>
        <TweakRadio label="Theme" value={t.theme} options={["dark", "light", "system"]} onChange={(v) => setTweak("theme", v)}></TweakRadio>
        <TweakButton label="Reopen palette" onClick={openPal}></TweakButton>
      </TweaksPanel>
    </div>
  );
}
ReactDOM.createRoot(document.getElementById("root")).render(<App></App>);
