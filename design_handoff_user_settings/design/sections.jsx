function useNow(ms = 15000) {
  const [n, setN] = React.useState(() => new Date());
  React.useEffect(() => { const t = setInterval(() => setN(new Date()), ms); return () => clearInterval(t); }, []);
  return n;
}
const fmtTime = (d, tz, tf) => d.toLocaleTimeString("en-US", { timeZone: tz, hour: "numeric", minute: "2-digit", hour12: tf !== "24" });
const initials = n => n.split(/\s+/).filter(Boolean).slice(0, 2).map(x => x[0].toUpperCase()).join("") || "?";
const Rw = ({ children, id, sub }) => <div className={"rw" + (sub ? " sub" : "")} id={id}>{children}</div>;
const CardHead = ({ t, d, children }) => <div className="ch"><div><b>{t}</b>{d && <span>{d}</span>}</div>{children}</div>;

function ProfileSection({ s, set, toast }) {
  return (
    <>
      <div className="us-card">
        <CardHead t="About you" d="Shown to your team on comments, approvals and assignments." />
        <Rw id="row-photo">
          <FieldWrapper layout="inline" label="Photo" description="Square crop. JPG, PNG or WEBP.">
            <div className="ph-row">
              <Avatar s={s} size="sm" />
              <Button variant="outline" size="sm" onClick={() => set("photo", !s.photo)}><UI n="camera" s={14} />{s.photo ? "Change" : "Upload"}</Button>
              {s.photo && <Button variant="ghost" size="icon-sm" aria-label="Remove photo" onClick={() => set("photo", false)}><UI n="trash" s={14} /></Button>}
                          </div>
          </FieldWrapper>
        </Rw>
        {!s.photo && <Rw sub id="row-avatar"><FieldWrapper layout="inline" label="Avatar" description="No photo? Pick a gradient. Auto is made from your username, so it never changes."><AvatarPicker s={s} set={set} /></FieldWrapper></Rw>}
        <Rw id="row-name"><InputField layout="inline" label="Display name" value={s.name} onChange={e => set("name", e.target.value, true)} /></Rw>
        <Rw><InputField layout="inline" label="Job title" placeholder="e.g. Dispatcher" value={s.title} onChange={e => set("title", e.target.value, true)} /></Rw>
        <Rw><InputField layout="inline" label="Username" readOnly value={"@" + s.username} rightElement={<Button variant="ghost" size="icon-xs" aria-label="Copy username" onClick={() => toast("Copied @" + s.username)}><UI n="copy" s={13} /></Button>} /></Rw>
        <Rw><InputField layout="inline" label="Email" description="Managed by your organization. Ask an admin to change it." readOnly value={s.email} rightElement={<Badge tone="success"><UI n="check" s={10} w={2.4} />Verified</Badge>} /></Rw>
        <Rw><PhoneNumberField layout="inline" label="Phone" description="Optional. Only your teammates see it." placeholder="(312) 555-0142" value={s.phone} onChange={e => set("phone", e.target.value, true)} /></Rw>
      </div>
      <div className="us-card">
        <CardHead t="Availability" d="Let your team know where you are. Notifications follow it." />
        <Rw id="row-availability"><SegmentedField layout="inline" label="Status" options={[{ value: "available", label: "Available" }, { value: "focus", label: "Focus" }, { value: "away", label: "Away" }]} value={s.status} onChange={v => set("status", v)} /></Rw>
        {s.status === "focus" && <Rw sub><SegmentedField layout="inline" label="Focus for" description="Only urgent notifications and direct @mentions reach you." options={[{ value: "1h", label: "1 hour" }, { value: "2h", label: "2 hours" }, { value: "eod", label: "Rest of day" }]} value={s.focusFor} onChange={v => set("focusFor", v)} /></Rw>}
        {s.status === "away" && <>
          <Rw sub><DatePickerField layout="inline" label="Back on" value={s.backOn} onChange={v => set("backOn", v)} /></Rw>
          <Rw sub id="row-status"><SelectField layout="inline" label="Hand off to" description="Gets your approvals, tender reviews and @mentions while you’re away." placeholder="No one — hold them for me" value={s.delegate} onChange={v => set("delegate", v)} options={[{ value: "", label: "No one — hold them for me" }, ...US_TEAM.map(([k, n, r]) => ({ value: k, label: n, description: r }))]} /></Rw>
          <Rw sub><InputField layout="inline" label="Away note" description="Shown when someone @mentions you." placeholder="Out until Monday — Jordan has my loads." value={s.awayNote} onChange={e => set("awayNote", e.target.value, true)} /></Rw>
        </>}
      </div>
    </>
  );
}

function PrefsSection({ s, set }) {
  const now = useNow();
  const devTz = Intl.DateTimeFormat().resolvedOptions().timeZone;
  return (
    <div className="us-card">
      <Rw id="row-theme">
        <FieldWrapper layout="inline" label="Theme" description="Light, dark, or follow your system.">
          <div className="thm">
            {[["light", "Light", "l"], ["dark", "Dark", "d"], ["system", "System", "s"]].map(([v, l, c]) => (
              <button key={v} className={s.theme === v ? "on" : ""} onClick={() => set("theme", v)}>
                <span className={"thm-p " + c}>{c === "s" ? <><span className="h hl"><i></i><span><b></b><b></b></span></span><span className="h hd"><i></i><span><b></b><b></b></span></span></> : <span className="h"><i></i><span><b></b><b></b><b style={{ width: "70%" }}></b></span></span>}</span>
                <span>{l}</span>
              </button>
            ))}
          </div>
        </FieldWrapper>
      </Rw>
      <Rw id="row-language"><SelectField layout="inline" label="Language" description="Menus, labels and the emails Trenova sends you." value={s.lang} onChange={v => set("lang", v)} options={US_LANG.map(([v, l]) => ({ value: v, label: l }))} /></Rw>
      <Rw id="row-timezone"><TimezoneField layout="inline" label="Time zone" description={<>It’s <b className="mono" style={{ fontWeight: 500 }}>{fmtTime(now, s.tz, s.timeFormat)}</b> for you.{s.tz !== devTz && US_TZ.some(z => z[0] === devTz) && <> <a href="#" onClick={e => { e.preventDefault(); set("tz", devTz); }}>Use this device’s</a></>}</>} value={s.tz} onChange={v => set("tz", v)} /></Rw>
      <Rw id="row-timeformat"><SegmentedField layout="inline" label="Time format" description={<>Pickups show as <b className="mono" style={{ fontWeight: 500 }}>{s.timeFormat === "24" ? "14:30" : "2:30 PM"}</b></>} options={[{ value: "12", label: "12-hour" }, { value: "24", label: "24-hour" }]} value={s.timeFormat} onChange={v => set("timeFormat", v)} /></Rw>
      <Rw id="row-sidebar"><SegmentedField layout="inline" label="Sidebar" description="Full labels, or a slim rail of icons." options={[{ value: "full", label: "Labels" }, { value: "rail", label: "Rail" }]} value={s.sidebar} onChange={v => set("sidebar", v)} /></Rw>
      <Rw id="row-motion"><SwitchField label="Reduce motion" description="Fewer slides, springs and other movement." checked={s.motion === "reduced"} onChange={v => set("motion", v ? "reduced" : "full")} /></Rw>
    </div>
  );
}

function SecuritySection({ s, set, toast }) {
  const [pw, setPw] = React.useState(null);
  const [codes, setCodes] = React.useState(false);
  const [sess, setSess] = React.useState(US_SESSIONS);
  const [gone, setGone] = React.useState([]);
  const out = ids => { setGone(g => [...g, ...ids]); setTimeout(() => setSess(x => x.filter(y => !ids.includes(y.id))), 300); toast(ids.length > 1 ? "Signed out of " + ids.length + " sessions" : "Session signed out"); };
  const np = pw ? pw.n : "";
  const reqs = [["12+ characters", np.length >= 12], ["A number", /\d/.test(np)], ["Upper and lower case", /[a-z]/.test(np) && /[A-Z]/.test(np)], ["A symbol", /[^\w\s]/.test(np)]];
  const score = reqs.filter(r => r[1]).length;
  const mc = ["var(--b)", "var(--danger)", "var(--warn)", "var(--warn)", "var(--success)"][score];
  return (
    <>
      <div className="us-card">
        <CardHead t="Signing in" />
        <Rw id="row-password">
          <FieldWrapper layout="inline" label="Password" description={s.pwChanged ? "Changed just now." : "Last changed 94 days ago."}>
            {!pw ? <div><Button variant="outline" size="sm" onClick={() => setPw({ c: "", n: "", r: "" })}><UI n="key" s={14} />Change password</Button></div> : <div className="pw">
              <SensitiveField label="Current password" value={pw.c} onChange={e => setPw({ ...pw, c: e.target.value })} autoFocus />
              <SensitiveField label="New password" value={pw.n} onChange={e => setPw({ ...pw, n: e.target.value })} />
              <div className="meter">{[1, 2, 3, 4].map(i => <i key={i} style={{ background: i <= score ? mc : undefined }}></i>)}</div>
              <div className="reqs">{reqs.map(([l, ok]) => <span key={l} className={ok ? "ok" : ""}><UI n={ok ? "check" : "x"} s={11} w={2.2} />{l}</span>)}</div>
              <SensitiveField label="Confirm new password" value={pw.r} onChange={e => setPw({ ...pw, r: e.target.value })} />
              <div className="sub-a"><Button variant="ghost" size="sm" onClick={() => setPw(null)}>Cancel</Button><Button size="sm" disabled={!(pw.c && score === 4 && pw.r === pw.n)} onClick={() => { setPw(null); set("pwChanged", true); }}>Update password</Button></div>
            </div>}
          </FieldWrapper>
        </Rw>
        <Rw id="row-2fa">
          <FieldWrapper layout="inline" label="Two-factor authentication" description={s.mfa ? "Authenticator app. Asks for a code on new devices." : "Ask for a code from an authenticator app when you sign in."}>
            <div className="ph-row">{s.mfa ? <><Badge tone="success">On</Badge><Button variant="outline" size="sm" onClick={() => setCodes(c => !c)}>Recovery codes</Button><Button variant="quiet" size="sm" onClick={() => set("mfa", false)}>Turn off</Button></> : <Button size="sm" onClick={() => set("mfa", true)}><UI n="shieldOk" s={14} />Set up</Button>}</div>
            {s.mfa && codes && <div className="codes-w">
              <p className="f-desc">8 of 10 left. Each works once if you lose your phone.</p>
              <div className="codes">{["4F7K-29QX", "8MZT-1B6P", "Q3RN-77WD", "C9LA-0HJE", "T2VY-5KSM", "6WQE-PL3R", "N8FD-4XCA", "H1UB-93ZK", "R7MJ-02TE", "ZK4P-8YQN"].map((c, i) => <span key={c} className={i < 2 ? "used" : ""}>{c}</span>)}</div>
              <div className="ph-row"><Button variant="outline" size="xs" onClick={() => toast("Recovery codes copied")}><UI n="copy" s={12} />Copy</Button><Button variant="ghost" size="xs" onClick={() => toast("New recovery codes generated")}>Generate new codes</Button></div>
            </div>}
          </FieldWrapper>
        </Rw>
      </div>
      <div className="us-card" id="row-sessions">
        <CardHead t="Where you’re signed in">{sess.length > 1 && <Button variant="quiet" size="xs" onClick={() => out(sess.filter(x => !x.me).map(x => x.id))}><UI n="out" s={12} />Sign out everywhere else</Button>}</CardHead>
        {sess.map(x => (
          <div key={x.id} className={"ses" + (gone.includes(x.id) ? " gone" : "")}>
            <span className="ses-i"><UI n={x.dev} s={16} /></span>
            <div className="ses-t"><b>{x.t}{x.me && <Badge tone="brand">This device</Badge>}</b><span>{x.where} · <span className="mono">{x.ip}</span> · {x.me ? "Active now" : x.last}</span></div>
            {!x.me && <Button variant="outline" size="xs" onClick={() => out([x.id])}>Sign out</Button>}
          </div>
        ))}
      </div>
      <div className="us-card" id="row-activity">
        <CardHead t="Recent sign-ins" />
        <div className="log">{US_SIGNINS.map(([k, t, w, d], i) => <React.Fragment key={i}><i className={k}></i><span>{t}<em>{w}</em></span><time>{d}</time></React.Fragment>)}</div>
      </div>
    </>
  );
}
Object.assign(window, { useNow, fmtTime, initials, Rw, CardHead, ProfileSection, PrefsSection, SecuritySection });
