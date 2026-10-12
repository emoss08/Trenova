const fmtH = (h, f) => {
  const hh = Math.floor(h) % 24, mm = h % 1 ? "30" : "00";
  if (f === "24") return String(hh).padStart(2, "0") + ":" + mm;
  return ((hh % 12) || 12) + ":" + mm + " " + (hh < 12 ? "AM" : "PM");
};
const HOURS = [...Array(48)].map((_, i) => i / 2);
const PAUSE_OPTS = [{ value: "off", label: "Not paused" }, { value: "1h", label: "For 1 hour" }, { value: "tom", label: "Until 8:00 AM tomorrow" }, { value: "mon", label: "Until Monday" }];
const MODE_OPTS = [{ value: "off", label: "Off" }, { value: "urgent", label: "Urgent only" }, { value: "all", label: "Everything" }];
const isLocked = e => !!e.lock;

function NxCategory({ c, nx, setNx, open, onToggle }) {
  const free = c.ev.filter(e => !isLocked(e));
  const on = c.ev.filter(e => nx[e.k]).length;
  const allOn = free.every(e => nx[e.k]);
  const setAll = v => { const n = { ...nx }; free.forEach(e => { n[e.k] = v; }); setNx(n); };
  return (
    <div className={"nc" + (open ? " open" : "")}>
      <div className="nc-r">
        <button className="nc-n" onClick={onToggle} aria-expanded={open}>
          <span className={"nc-ti tone-" + c.tone}><UI n={c.icon} s={14} /></span>
          <span className="nc-t"><b>{c.name}</b><span>{c.blurb}</span></span>
        </button>
        {c.ev.length > 1 && <Button variant="quiet" size="xs" onClick={onToggle}>{on === c.ev.length ? "All " + c.ev.length : on + " of " + c.ev.length}<UI n="chevD" s={12} /></Button>}
        <Switch checked={on > 0} onCheckedChange={() => setAll(!(on > 0))} />
      </div>
      {open && c.ev.length > 1 && <div className="nc-evs">
        {c.ev.map(e => (
          <SwitchField key={e.k} label={e.t} checked={!!nx[e.k]} disabled={isLocked(e)} onChange={v => setNx({ ...nx, [e.k]: v })}
            extra={<>{e.sev === "crit" && <Badge tone="danger">Urgent</Badge>}{isLocked(e) && <Badge><UI n="lock" s={10} />Required</Badge>}</>}
            description={isLocked(e) ? e.lock : null} />
        ))}
      </div>}
    </div>
  );
}

function NotifySection({ s, set, preview, perm, askPerm }) {
  const [open, setOpen] = React.useState(null);
  const tf = s.timeFormat, q = s.quiet;
  const hourOpts = HOURS.map(h => ({ value: h, label: fmtH(h, tf) }));
  const deskDesc = s.deskMode === "off" ? "Nothing pops up outside Trenova." : perm === "granted"
    ? <>Allowed in this browser. <a href="#" onClick={e => { e.preventDefault(); preview(NX_CATS[2], NX_CATS[2].ev[1]); }}>Send a test</a></>
    : <>This browser hasn’t allowed Trenova yet. <a href="#" onClick={e => { e.preventDefault(); askPerm(); }}>Allow notifications</a></>;
  return (
    <>
      <div className="us-card">
        <CardHead t="Delivery" d="Everything lands in the bell. Choose what also reaches you outside Trenova." />
        <Rw id="row-desktop"><SegmentedField layout="inline" label="Desktop" description={deskDesc} options={MODE_OPTS} value={s.deskMode} onChange={v => set("deskMode", v)} /></Rw>
        <Rw id="row-digest"><SegmentedField layout="inline" label="Email" description={s.emailMode === "off" ? "No notification emails." : <>Sent to {s.email}</>} options={MODE_OPTS} value={s.emailMode} onChange={v => set("emailMode", v)} /></Rw>
        {s.emailMode === "all" && <Rw sub><SwitchField label="Bundle into a daily digest" description="Non-urgent emails arrive together at 8:00 AM. Urgent ones still come right away." checked={s.digest} onChange={v => set("digest", v)} /></Rw>}
        <Rw id="row-sound"><SwitchField label="Sound" description="A soft chime with desktop notifications." checked={s.sound} onChange={v => set("sound", v)} /></Rw>
      </div>
      <div className="us-card">
        <Rw id="row-quiet"><SwitchField label="Quiet hours" description="Hold desktop notifications and emails during these hours. They wait in the bell." checked={q.on} onChange={v => set("quiet", { ...q, on: v })} /></Rw>
        {q.on && <>
          <Rw sub><FieldWrapper layout="inline" label="Hours"><div className="qh-pair"><Select value={q.from} onChange={v => set("quiet", { ...q, from: +v })} options={hourOpts} /><span>to</span><Select value={q.to} onChange={v => set("quiet", { ...q, to: +v })} options={hourOpts} /></div></FieldWrapper></Rw>
          <Rw sub><FieldWrapper layout="inline" label="Days"><ToggleChipsField label="Quiet days" options={["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((l, i) => ({ value: i, label: l }))} value={q.days} onChange={v => set("quiet", { ...q, days: v.sort() })} /></FieldWrapper></Rw>
          <Rw sub><SwitchField label="Let urgent ones through" description="Waterfalls running dry, blocked carriers, failed EDI and the like." checked={q.crit} onChange={v => set("quiet", { ...q, crit: v })} /></Rw>
        </>}
      </div>
      <div className="us-card">
        <Rw id="row-pause"><SelectField layout="inline" label="Pause notifications" description="Stop desktop notifications and emails for a while." value={s.pause} onChange={v => set("pause", v)} options={PAUSE_OPTS} /></Rw>
      </div>
      <div className="us-card" id="row-matrix">
        <CardHead t="What you hear about" d="Turn off anything you don’t need. A few are required by your organization." />
        {NX_CATS.map(c => <NxCategory key={c.id} c={c} nx={s.nx} setNx={v => set("nx", v)} open={open === c.id} onToggle={() => setOpen(o => o === c.id ? null : c.id)} />)}
      </div>
    </>
  );
}
Object.assign(window, { NotifySection, fmtH, PAUSE_OPTS });
