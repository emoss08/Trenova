// Mocks of Trenova's form components. Names, props and visuals follow:
//   @trenova/shared/components/ui/{button,input,switch,segmented-control}.tsx
//   @/components/fields/{field-components,input-field,select-field,timezone-field,phone-number-field,
//                         sensitive-field,date-picker-field,segmented-field,switch-field,toggle-chips-field}.tsx
const cx = (...a) => a.filter(Boolean).join(" ");

function Button({ variant = "default", size = "default", className, children, ...p }) {
  return <button type="button" className={cx("tb", "tb-" + variant, "tb-s-" + size, className)} {...p}>{children}</button>;
}
function Input({ leftElement, rightElement, sideText, className, inputContainerClassName, ...p }) {
  return (
    <div className={cx("ti-wrap", inputContainerClassName)}>
      {leftElement && <span className="ti-l">{leftElement}</span>}
      <input className={cx("ui-field ti", leftElement && "pl", (rightElement || sideText) && "pr", className)} {...p} />
      {sideText && <span className="ti-side">{sideText}</span>}
      {rightElement && <span className="ti-r">{rightElement}</span>}
    </div>
  );
}
function FieldDescription({ description }) { return description ? <p className="f-desc">{description}</p> : null; }
function FieldWrapper({ label, description, layout = "stacked", required, className, children, id }) {
  if (layout === "inline") return (
    <div className={cx("fw-inline", className)} id={id}>
      <div className="fw-il"><label className={cx("f-label lg", required && "req")}>{label}</label><FieldDescription description={description} /></div>
      <div className="fw-ic">{children}</div>
    </div>
  );
  return (
    <div className={cx("fw", className)} id={id}>
      {label && <label className={cx("f-label", required && "req")}>{label}</label>}
      {children}
      <FieldDescription description={description} />
    </div>
  );
}
function InputField({ label, description, layout, id, ...p }) {
  return <FieldWrapper label={label} description={description} layout={layout} id={id}><Input {...p} /></FieldWrapper>;
}
function SensitiveField({ label, description, layout, ...p }) {
  const [show, setShow] = React.useState(false);
  return <FieldWrapper label={label} description={description} layout={layout}><Input type={show ? "text" : "password"} rightElement={<Button variant="ghost" size="icon-xs" aria-label={show ? "Hide" : "Show"} onClick={() => setShow(s => !s)}><UI n="eye" s={13} /></Button>} {...p} /></FieldWrapper>;
}
function PhoneNumberField({ label, description, layout, ...p }) {
  return <FieldWrapper label={label} description={description} layout={layout}><Input leftElement={<span className="mono" style={{ fontSize: 12 }}>+1</span>} {...p} /></FieldWrapper>;
}
function useOutside(ref, on, close) {
  React.useEffect(() => { if (!on) return; const h = e => { if (ref.current && !ref.current.contains(e.target)) close(); }; document.addEventListener("mousedown", h); return () => document.removeEventListener("mousedown", h); }, [on]);
}
function Select({ value, onChange, options, placeholder = "Select…", className }) {
  const [open, setOpen] = React.useState(false); const ref = React.useRef(null);
  useOutside(ref, open, () => setOpen(false));
  const cur = options.find(o => String(o.value) === String(value));
  return (
    <div className={cx("sel", className)} ref={ref}>
      <button type="button" className={cx("ui-field sel-t", open && "open")} onClick={() => setOpen(o => !o)}>
        <span className={cur ? "" : "ph"}>{cur ? cur.label : placeholder}</span>{cur && cur.description && <em>{cur.description}</em>}<UI n="chevD" s={13} />
      </button>
      {open && <div className="pop sel-p">{(options[0] && options[0].group ? [...new Set(options.map(o => o.group))] : [null]).map(g => (
        <div key={g || "_"}>{g && <div className="sel-g">{g}</div>}
          {options.filter(o => !g || o.group === g).map(o => <button key={o.value} type="button" className={cx("sel-o", String(o.value) === String(value) && "on")} onClick={() => { onChange(o.value); setOpen(false); }}><span>{o.label}</span>{o.description && <em>{o.description}</em>}{String(o.value) === String(value) && <UI n="check" s={13} w={2} />}</button>)}
        </div>))}</div>}
    </div>
  );
}
function SelectField({ label, description, layout, id, ...p }) {
  return <FieldWrapper label={label} description={description} layout={layout} id={id}><Select {...p} /></FieldWrapper>;
}
function TimezoneField(p) {
  return <SelectField placeholder="Select timezone" options={US_TZ.map(([v, l, c, off, g]) => ({ value: v, label: l, description: off, group: g }))} {...p} />;
}
function SegmentedControl({ items, value, onValueChange, fullWidth, "aria-label": al }) {
  const i = Math.max(0, items.findIndex(x => x.value === value));
  return (
    <div role="radiogroup" aria-label={al} className={cx("seg", fullWidth && "full")} style={{ "--n": items.length, "--i": i }}>
      <span className="kn"></span>
      {items.map(x => <button key={x.value} type="button" role="radio" aria-checked={x.value === value} disabled={x.disabled} className={x.value === value ? "on" : ""} onClick={() => onValueChange(x.value)}>
        <span className="seg-l">{x.icon && <UI n={x.icon} s={13} />}{x.label}</span>{x.caption !== undefined && <span className="seg-c">{x.caption}</span>}
      </button>)}
    </div>
  );
}
function SegmentedField({ label, description, layout, options, value, onChange, fullWidth = true, id }) {
  return <FieldWrapper label={label} description={description} layout={layout} id={id}><SegmentedControl aria-label={label} fullWidth={fullWidth} value={value} onValueChange={onChange} items={options} /></FieldWrapper>;
}
function Switch({ checked, onCheckedChange, size = "default", disabled, id }) {
  return <button type="button" id={id} role="switch" aria-checked={!!checked} disabled={disabled} className={cx("tsw", size === "sm" && "sm", checked && "on")} onClick={e => { e.stopPropagation(); onCheckedChange(!checked); }}><i></i></button>;
}
function SwitchField({ label, description, checked, onChange, disabled, outlined, position = "right", id, extra }) {
  const sw = <Switch checked={checked} onCheckedChange={onChange} disabled={disabled} />;
  return (
    <div className={cx("swf", outlined && "outlined", checked && "on", disabled && "dis")} id={id} onClick={() => !disabled && onChange(!checked)}>
      {position === "left" && sw}
      <div className="swf-t"><span className="swf-lr"><label className="f-label sw">{label}</label>{extra}</span>{description && <p className="f-desc">{description}</p>}</div>
      {position === "right" && sw}
    </div>
  );
}
function ToggleChipsField({ label, options, value, onChange, disabled }) {
  return (
    <div className="chips" role="group" aria-label={label}>
      {options.map(o => { const on = value.includes(o.value); return <button key={o.value} type="button" aria-pressed={on} disabled={disabled} className={cx("chip", on && "on")} onClick={() => onChange(on ? value.filter(v => v !== o.value) : [...value, o.value])}><UI n={on ? "check" : "plus"} s={11} w={2} />{o.label}</button>; })}
    </div>
  );
}
const MONTHS = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
const ord = d => d + (d % 10 === 1 && d !== 11 ? "st" : d % 10 === 2 && d !== 12 ? "nd" : d % 10 === 3 && d !== 13 ? "rd" : "th");
function DatePickerField({ label, description, layout, value, onChange, id }) {
  const [open, setOpen] = React.useState(false); const ref = React.useRef(null);
  useOutside(ref, open, () => setOpen(false));
  const sel = value ? new Date(value + "T12:00") : null;
  const [view, setView] = React.useState(() => { const d = sel || new Date(); return [d.getFullYear(), d.getMonth()]; });
  const [y, m] = view, first = new Date(y, m, 1).getDay(), days = new Date(y, m + 1, 0).getDate();
  const today = new Date(); const iso = d => `${y}-${String(m + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return (
    <FieldWrapper label={label} description={description} layout={layout} id={id}>
      <div className="sel" ref={ref}>
        <Button variant="outline" className={cx("dp-t", !sel && "empty")} onClick={() => setOpen(o => !o)}><UI n="calendar" s={15} />{sel ? `${MONTHS[sel.getMonth()]} ${ord(sel.getDate())}, ${sel.getFullYear()}` : "Pick a date"}</Button>
        {open && <div className="pop cal">
          <div className="cal-h"><Button variant="ghost" size="icon-sm" onClick={() => setView(m ? [y, m - 1] : [y - 1, 11])}><UI n="back" s={13} /></Button><b>{MONTHS[m]} {y}</b><Button variant="ghost" size="icon-sm" onClick={() => setView(m === 11 ? [y + 1, 0] : [y, m + 1])}><UI n="chevR" s={13} /></Button></div>
          <div className="cal-g">{["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"].map(d => <span key={d} className="cal-w">{d}</span>)}
            {[...Array(first)].map((_, i) => <span key={"e" + i}></span>)}
            {[...Array(days)].map((_, i) => { const d = i + 1, on = value === iso(d), td = today.getFullYear() === y && today.getMonth() === m && today.getDate() === d; return <button key={d} type="button" className={cx("cal-d", on && "on", td && "td")} onClick={() => { onChange(iso(d)); setOpen(false); }}>{d}</button>; })}
          </div>
        </div>}
      </div>
    </FieldWrapper>
  );
}
function Badge({ tone = "neutral", children }) { return <span className={"bdg " + tone}>{children}</span>; }
Object.assign(window, { cx, Button, Input, FieldWrapper, FieldDescription, InputField, SensitiveField, PhoneNumberField, Select, SelectField, TimezoneField, SegmentedControl, SegmentedField, Switch, SwitchField, ToggleChipsField, DatePickerField, Badge });
