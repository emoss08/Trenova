const {useState,useEffect,useRef,useLayoutEffect,useMemo}=React;

const OB_STATES=[["Alabama","AL"],["Alaska","AK"],["Arizona","AZ"],["Arkansas","AR"],["California","CA"],["Colorado","CO"],["Connecticut","CT"],["Delaware","DE"],["District of Columbia","DC"],["Florida","FL"],["Georgia","GA"],["Hawaii","HI"],["Idaho","ID"],["Illinois","IL"],["Indiana","IN"],["Iowa","IA"],["Kansas","KS"],["Kentucky","KY"],["Louisiana","LA"],["Maine","ME"],["Maryland","MD"],["Massachusetts","MA"],["Michigan","MI"],["Minnesota","MN"],["Mississippi","MS"],["Missouri","MO"],["Montana","MT"],["Nebraska","NE"],["Nevada","NV"],["New Hampshire","NH"],["New Jersey","NJ"],["New Mexico","NM"],["New York","NY"],["North Carolina","NC"],["North Dakota","ND"],["Ohio","OH"],["Oklahoma","OK"],["Oregon","OR"],["Pennsylvania","PA"],["Rhode Island","RI"],["South Carolina","SC"],["South Dakota","SD"],["Tennessee","TN"],["Texas","TX"],["Utah","UT"],["Vermont","VT"],["Virginia","VA"],["Washington","WA"],["West Virginia","WV"],["Wisconsin","WI"],["Wyoming","WY"]];
const OB_TZS=[{v:"America/New_York",l:"Eastern time"},{v:"America/Chicago",l:"Central time"},{v:"America/Denver",l:"Mountain time"},{v:"America/Phoenix",l:"Arizona"},{v:"America/Los_Angeles",l:"Pacific time"},{v:"America/Anchorage",l:"Alaska time"},{v:"Pacific/Honolulu",l:"Hawaii time"}];
const OB_MODS={asset:["Dispatch","Fleet","Hours of service","Driver pay"],broker:["Carrier sourcing","Tenders","Rate confirmations","Carrier settlements"]};
const OB_OPS=[
  {id:"asset",t:"Asset carrier",d:"You haul freight with your own trucks and drivers. Dispatch, fleet, hours of service and driver pay lead the product.",mods:OB_MODS.asset},
  {id:"broker",t:"Freight brokerage",d:"You arrange freight with outside carriers. Carrier sourcing, tenders, rate confirmations and carrier settlements lead the product.",mods:OB_MODS.broker},
  {id:"both",t:"Both",d:"You run your own fleet and broker loads to partner carriers. Every part of the product is turned on.",mods:[...OB_MODS.asset,...OB_MODS.broker]}
];
const OB_SAMPLE=[["Customers",2,8],["Locations",4,25],["Workers",1,3],["Tractors",1,3],["Trailers",1,3],["Shipments",2,12]];
const OB_GROUPS=[["Company profile","Name, timezone and address"],["Operation type","Trucks, brokerage or both"],["Sample data","Start empty or with examples"],["Review","Confirm and finish"]];

const tzLabel=v=>(OB_TZS.find(t=>t.v===v)||{}).l||v;
const tzTime=v=>{try{return new Intl.DateTimeFormat("en-US",{hour:"numeric",minute:"2-digit",timeZone:v}).format(new Date())}catch(e){return""}};
const stAbbr=n=>(OB_STATES.find(s=>s[0]===n)||[])[1]||n;
const initials=n=>{const w=n.replace(/\b(LLC|INC|CO|CORP|LTD)\b\.?/gi,"").trim().split(/\s+/).filter(Boolean);return (w.length>1?w[0][0]+w[1][0]:(w[0]||"").slice(0,2)).toUpperCase()};
const addrLine=a=>[a.street,a.city,[stAbbr(a.state),a.zip].filter(Boolean).join(" ")].filter(Boolean).join(", ");
const opById=id=>OB_OPS.find(o=>o.id===id);

const Ic={
  up:<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M8 13V3M3.5 7.5 8 3l4.5 4.5"/></svg>,
  ck:<svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="m3.5 8.5 3 3 6-7"/></svg>,
  ed:<svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><path d="M10.5 2.5l3 3L6 13H3v-3z"/></svg>,
  ar:<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"><path d="M3 8h10M9 4l4 4-4 4"/></svg>,
  bk:<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M13 8H3M7 4 3 8l4 4"/></svg>,
  ent:<svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"><path d="M13 3v5a2 2 0 0 1-2 2H3M6 7l-3 3 3 3"/></svg>
};

/* Desk-voiced, typed message. segs: strings or {b:"bold"} */
function Typed({segs,animate,speed,onDone,onTick}){
  const flat=segs.map(s=>typeof s==="string"?s:s.b).join("");
  const [n,setN]=useState(animate?0:flat.length);
  const [thinking,setThinking]=useState(animate);
  const sp=useRef(speed);sp.current=speed;
  const done=useRef(onDone);done.current=onDone;
  const tick=useRef(onTick);tick.current=onTick;
  useEffect(()=>{
    if(!animate){done.current&&done.current();return}
    let alive=true,i=0,to;
    const step=()=>{if(!alive)return;i++;setN(i);tick.current&&tick.current();
      if(i>=flat.length){to=setTimeout(()=>alive&&done.current&&done.current(),180/sp.current);return}
      const ch=flat[i-1];let d=12+Math.random()*18;
      if(".?!".includes(ch))d+=260;else if(",—".includes(ch))d+=110;else if(ch==="\n")d+=200;
      to=setTimeout(step,d/sp.current)};
    to=setTimeout(()=>{if(!alive)return;setThinking(false);step()},650/sp.current);
    return()=>{alive=false;clearTimeout(to)};
  },[]);
  const N=animate?n:flat.length;let left=N;const out=[];
  segs.forEach((s,k)=>{if(left<=0)return;const t=typeof s==="string"?s:s.b;const part=t.slice(0,left);left-=t.length;out.push(typeof s==="string"?<React.Fragment key={k}>{part}</React.Fragment>:<b key={k}>{part}</b>)});
  const typing=N<flat.length;
  return <div>
    <div className="who"><span className={"dm"+(animate&&(thinking||typing)?" busy":"")}></span><b>Nova</b><span>Setup guide</span></div>
    {thinking&&animate?<div className="thk">Typing</div>:<div className="prose">{out}{typing&&<span className="caret"></span>}</div>}
  </div>;
}

function Rail({group}){
  return <aside className="rail">
    <div className="st">{OB_GROUPS.map(([t,s],i)=>{const c=i<group?"past":i===group?"cur":"";
      return <div key={i} className={"st-i "+c}><span className="st-d">{i<group&&Ic.ck}</span><span className="st-t">{t}</span><span className="st-s">{s}</span></div>})}</div>
    <p className="rail-n">Everything here can be changed later in Organization settings.</p>
  </aside>;
}

function Val({v,mono}){return v?<span key={v} className={"fill"+(mono?" mono":"")}>{v}</span>:<span className="pend"></span>}

function Preview({a,cur,live,stepIds}){
  const reached=id=>stepIds.indexOf(id)<cur||live;
  const [,setT]=useState(0);
  useEffect(()=>{const i=setInterval(()=>setT(x=>x+1),30000);return()=>clearInterval(i)},[]);
  const mods=a.op?opById(a.op).mods:[];
  const ini=initials(a.name);
  return <aside className="pvc">
    <div className="pv-cap"><span>Your workspace</span><span className={"pv-st"+(live?" live":"")}><i></i>{live?"Live":"Draft"}</span></div>
    <div className={"pv"+(live?" live":"")}>
      <div className="pv-h"><span className={"pv-av"+(ini?"":" no")}>{ini?<span key={ini}>{ini}</span>:"?"}</span>
        <span className="pv-n"><b>{a.name?<span key={a.name} className="fill">{a.name}</span>:<span className="pend"></span>}</b><span>{reached("tz")?`${tzLabel(a.tz)} · ${tzTime(a.tz)}`:"Setting up"}</span></span></div>
      <div className="pv-sec">
        <div className="pv-r"><span className="k">HQ</span><span className="v">{reached("addr")?<Val v={[a.city,stAbbr(a.state)].filter(Boolean).join(", ")}/>:<span className="pend"></span>}</span></div>
        <div className="pv-r"><span className="k">SCAC</span><span className="v">{reached("ids")?<Val v={a.scac||"—"} mono/>:<span className="pend"></span>}</span></div>
        <div className="pv-r"><span className="k">USDOT</span><span className="v">{reached("ids")?<Val v={a.dot||"—"} mono/>:<span className="pend"></span>}</span></div>
      </div>
      <div className="pv-sec"><div className="pv-l">Modules</div>
        <div className="mdl">{[...OB_MODS.asset,...OB_MODS.broker].map((m,i)=><span key={m} className={"md"+(mods.includes(m)?" on":"")} style={{transitionDelay:(mods.includes(m)?i*60:0)+"ms"}}><i style={{transitionDelay:(mods.includes(m)?i*60:0)+"ms"}}></i>{m}</span>)}</div>
      </div>
      <div className="pv-sec"><div className="pv-l">Sample data</div>
        {a.sample===true?<div className="smp">{OB_SAMPLE.map(([k,n],i)=><div key={k} style={{animation:`rise 420ms var(--settle) ${i*50}ms both`}}><b>{n}</b><span>{k}</span></div>)}</div>
          :a.sample===false?<div className="pv-e fill">Starting empty</div>:<span className="pend"></span>}
      </div>
    </div>
  </aside>;
}

function TextAsk({value,placeholder,onSubmit}){
  const [v,setV]=useState(value||"");const [bad,setBad]=useState(0);const r=useRef();
  useEffect(()=>{r.current&&r.current.focus()},[]);
  const go=()=>{if(!v.trim()){setBad(b=>b+1);r.current.focus();return}onSubmit(v.trim())};
  return <div>
    <div key={bad} className={"cmp"+(bad?" bad":"")}><span className="cmp-ring"></span>
      <input ref={r} value={v} placeholder={placeholder} onChange={e=>setV(e.target.value)} onKeyDown={e=>{if(e.key==="Enter")go()}}/>
      <button className="send" disabled={!v.trim()} onClick={go} aria-label="Send">{Ic.up}</button></div>
    <div className="hint">{bad?<span className="err">Your company name is required.</span>:<><span className="kbd">↵</span> to answer · <span className="kbd">esc</span> to go back</>}</div>
  </div>;
}

function TzAsk({value,detected,onSubmit}){
  const [sel,setSel]=useState(value);
  useEffect(()=>{const k=e=>{const L=[...OB_TZS.filter(t=>t.v===detected),...OB_TZS.filter(t=>t.v!==detected)];const i=+e.key-1;if(i>=0&&i<L.length)pick(L[i].v);if(e.key==="Enter")pick(sel)};window.addEventListener("keydown",k);return()=>window.removeEventListener("keydown",k)});
  const pick=v=>{setSel(v);setTimeout(()=>onSubmit(v),280)};
  const list=[...OB_TZS.filter(t=>t.v===detected),...OB_TZS.filter(t=>t.v!==detected)];
  return <div className="tzs">{list.map((t,i)=><button key={t.v} className={"tz"+(sel===t.v?" on":"")} style={{animationDelay:i*40+"ms"}} onClick={()=>pick(t.v)}>
    <b>{t.l}</b><span>{tzTime(t.v)}</span></button>)}</div>;
}

function AddrAsk({a,onSubmit}){
  const [f,setF]=useState({street:a.street,city:a.city,state:a.state,zip:a.zip});const [err,setErr]=useState({});const [n,setN]=useState(0);const r=useRef();
  useEffect(()=>{r.current&&r.current.focus()},[]);
  const set=k=>e=>{let v=e.target.value;if(k==="zip")v=v.replace(/\D/g,"").slice(0,5);setF({...f,[k]:v});if(err[k])setErr({...err,[k]:0})};
  const go=()=>{const e={street:!f.street.trim(),city:!f.city.trim(),state:!f.state,zip:!/^\d{5}$/.test(f.zip)};setErr(e);setN(n+1);if(!Object.values(e).some(Boolean))onSubmit(f)};
  const kd=e=>{if(e.key==="Enter")go()};
  const cls=k=>"in"+(err[k]?" bad":"");
  return <div className="fc">
    <div className="fg">
      <div className="fl full"><label>Street address</label><input key={"s"+(err.street?n:0)} ref={r} className={cls("street")} value={f.street} onChange={set("street")} onKeyDown={kd} placeholder="1200 Industrial Pkwy" autoComplete="street-address"/></div>
      <div className="fl"><label>City</label><input key={"c"+(err.city?n:0)} className={cls("city")} value={f.city} onChange={set("city")} onKeyDown={kd} placeholder="Whitsett"/></div>
      <div className="fl"><label>State</label><select key={"t"+(err.state?n:0)} className={cls("state")+(f.state?"":" ph")} value={f.state} onChange={set("state")} onKeyDown={kd}><option value="">Select</option>{OB_STATES.map(s=><option key={s[1]} value={s[0]}>{s[0]}</option>)}</select></div>
      <div className="fl"><label>ZIP</label><input key={"z"+(err.zip?n:0)} className={cls("zip")+" mono"} inputMode="numeric" value={f.zip} onChange={set("zip")} onKeyDown={kd} placeholder="27377"/></div>
    </div>
    <div className="fc-f">{Object.values(err).some(Boolean)?<span className="err">{err.zip&&f.zip?"ZIP code needs 5 digits.":"Fill in the highlighted fields."}</span>:<span className="hint"><span className="kbd">↵</span> to continue</span>}<span className="sp"></span><button className="bt ink" onClick={go}>Continue {Ic.ar}</button></div>
  </div>;
}

function IdsAsk({a,onSubmit}){
  const [f,setF]=useState({scac:a.scac,dot:a.dot});const [err,setErr]=useState({});const r=useRef();
  useEffect(()=>{r.current&&r.current.focus()},[]);
  const go=()=>{const e={scac:f.scac&&!/^[A-Z]{2,4}$/.test(f.scac),dot:f.dot&&!/^\d{1,8}$/.test(f.dot)};setErr(e);if(!e.scac&&!e.dot)onSubmit(f)};
  const kd=e=>{if(e.key==="Enter")go()};
  const empty=!f.scac&&!f.dot;
  return <div className="fc">
    <div className="fg two">
      <div className="fl"><label>SCAC code <i>optional</i></label><input ref={r} className={"in mono"+(err.scac?" bad":"")} value={f.scac} maxLength={4} placeholder="RVFL" onChange={e=>setF({...f,scac:e.target.value.toUpperCase().replace(/[^A-Z]/g,"")})} onKeyDown={kd}/><span className="help">{err.scac?<span className="err">2–4 letters.</span>:"Your Standard Carrier Alpha Code."}</span></div>
      <div className="fl"><label>USDOT number <i>optional</i></label><input className={"in mono"+(err.dot?" bad":"")} value={f.dot} inputMode="numeric" placeholder="3812045" onChange={e=>setF({...f,dot:e.target.value.replace(/\D/g,"").slice(0,8)})} onKeyDown={kd}/><span className="help">Issued by FMCSA.</span></div>
    </div>
    <div className="fc-f"><span className="hint"><span className="kbd">↵</span> to {empty?"skip":"continue"}</span><span className="sp"></span>{!empty&&<button className="bt" onClick={()=>onSubmit({scac:"",dot:""})}>Skip</button>}<button className="bt ink" onClick={go}>{empty?"Skip for now":"Continue"} {Ic.ar}</button></div>
  </div>;
}

function Choice({items,value,onSubmit,render}){
  const [sel,setSel]=useState(null);
  const pick=id=>{if(sel!==null)return;setSel(id);setTimeout(()=>onSubmit(id),420)};
  useEffect(()=>{const k=e=>{if(e.target.tagName==="INPUT")return;const i=+e.key-1;if(i>=0&&i<items.length)pick(items[i].id)};window.addEventListener("keydown",k);return()=>window.removeEventListener("keydown",k)});
  const on=sel!==null?sel:value;
  return <div className={"opts"+(sel!==null?" chosen":"")}>{items.map((it,i)=><button key={String(it.id)} className={"op"+(on===it.id?" on":"")} style={{animationDelay:i*60+"ms"}} onClick={()=>pick(it.id)}>
    <b>{it.t}</b><span className="kbd">{i+1}</span><p>{it.d}</p>{render&&render(it)}</button>)}</div>;
}

function Burst(){
  const cols=["oklch(0.75 0.14 220)","oklch(0.62 0.2 255)","oklch(0.65 0.22 345)","oklch(0.8 0.16 75)","oklch(0.72 0.145 152)"];
  return <span className="burst">{Array.from({length:22},(_,i)=><i key={i} className="spark" style={{"--a":(i*360/22+Math.random()*10)+"deg","--d":(34+Math.random()*40)+"px",background:cols[i%cols.length],animationDelay:(120+Math.random()*80)+"ms"}}></i>)}</span>;
}

Object.assign(window,{OB_STATES,OB_TZS,OB_MODS,OB_OPS,OB_SAMPLE,OB_GROUPS,tzLabel,tzTime,stAbbr,initials,addrLine,opById,Ic,Typed,Rail,Preview,TextAsk,TzAsk,AddrAsk,IdsAsk,Choice,Burst});
