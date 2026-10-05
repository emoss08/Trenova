const TWEAK_DEFAULTS=/*EDITMODE-BEGIN*/{"theme":"dark","speed":1,"firstName":"Marcus"}/*EDITMODE-END*/;
const STEP_IDS=["name","tz","addr","ids","op","sample","review"];
const GROUP_OF={name:0,tz:0,addr:0,ids:0,op:1,sample:2,review:3};
const DETECTED=(()=>{try{const z=Intl.DateTimeFormat().resolvedOptions().timeZone;return OB_TZS.some(t=>t.v===z)?z:"America/New_York"}catch(e){return "America/New_York"}})();
const BLANK={name:"",tz:DETECTED,street:"",city:"",state:"",zip:"",scac:"",dot:"",op:null,sample:null};

function say(id,a,u){
  const n={b:a.name||"your company"};
  switch(id){
    case "name":return [u?`Hi ${u}, I'm Nova. `:"Hi, I'm Nova. ","I'll help you set up your Trenova workspace. It takes about a minute, and you can change anything later.\n\nTo start, what's your company called?"];
    case "tz":return ["Got it, ",n,". Which timezone should dispatch, appointments and reports use? Your browser is set to ",{b:tzLabel(DETECTED)},"."];
    case "addr":return ["Where is ",n," headquartered? This becomes the default origin on new shipments."];
    case "ids":return ["Do you have a SCAC code or USDOT number? Both are optional — add them now or skip."];
    case "op":return ["How does ",n," move freight? This decides which parts of Trenova lead."];
    case "sample":return ["Want me to load sample data? Every screen will have something to show, and you can delete any of it at any time."];
    case "review":return [u?`That's everything, ${u}. `:"That's everything. ","Please check everything below before I create your workspace. If anything's wrong, click the line to fix it."];
  }
}
function ansText(id,a){
  switch(id){
    case "name":return a.name;
    case "tz":return tzLabel(a.tz);
    case "addr":return addrLine(a);
    case "ids":return a.scac||a.dot?[a.scac&&"SCAC "+a.scac,a.dot&&"USDOT "+a.dot].filter(Boolean).join(" · "):null;
    case "op":return opById(a.op).t;
    case "sample":return a.sample?"Load sample data":"Start empty";
    case "review":return "Finish setup";
  }
}

function Review({a,onEdit,onFinish}){
  useEffect(()=>{const k=e=>{if(e.key==="Enter"&&(e.metaKey||e.ctrlKey))onFinish()};window.addEventListener("keydown",k);return()=>window.removeEventListener("keydown",k)});
  const row=(k,v,step,i,mono)=><button key={k} className="rv-r" style={{animationDelay:120+i*45+"ms"}} onClick={()=>onEdit(step)}><span className="k">{k}</span><span className={"v"+(v?"":" no")+(mono?" mono":"")}>{v||"—"}</span></button>;
  return <div className="rv">
    <div className="rv-g"><div className="rv-h">Company profile</div>
      {row("Company name",a.name,0,0)}{row("Timezone",tzLabel(a.tz),1,1)}{row("Address",addrLine(a),2,2)}{row("SCAC code",a.scac,3,3,true)}{row("USDOT number",a.dot,3,4,true)}</div>
    <div className="rv-g"><div className="rv-h">Setup</div>
      {row("Operation type",opById(a.op).t,4,5)}{row("Sample data",a.sample?"Load sample data":"Start empty",5,6)}</div>
    <div className="rv-f"><span className="n">You can change all of this in Organization settings.</span><span className="sp"></span><button className="bt ink" onClick={onFinish}>Looks good, finish setup <span className="kbd">⌘↵</span></button></div>
  </div>;
}

function Build({a,speed,onReady}){
  const lines=useMemo(()=>{const op=opById(a.op);return [
    ["Creating ",a.name],["Setting the clock to ",tzLabel(a.tz)],["Saving ",`${a.city}, ${stAbbr(a.state)}`," as headquarters"],
    ["Turning on ",`${op.mods.length} modules`," for ",op.t.toLowerCase()],
    a.sample?["Loading ",`${OB_SAMPLE.reduce((s,r)=>s+r[1],0)} sample records`]:["Leaving records empty"],["Making you the owner"]]},[]);
  const [i,setI]=useState(0);const [el,setEl]=useState([]);
  useEffect(()=>{if(i>=lines.length){const t=setTimeout(onReady,300/speed);return()=>clearTimeout(t)}
    const d=(560+Math.random()*520)/speed;const t=setTimeout(()=>{setEl(e=>[...e,(d*speed/1000).toFixed(1)+"s"]);setI(i+1)},d);return()=>clearTimeout(t)},[i]);
  return <div className="nar">{lines.slice(0,i+1).filter((_,k)=>k<lines.length).map((l,k)=>{const past=k<i;
    return <div key={k} className={"nl "+(past?"past":"cur")}><span className="ck">{past?Ic.ck:<span className="spn"></span>}</span>
      <span className="tx">{l.map((p,j)=>j%2?<b key={j} style={{fontWeight:500,color:past?"var(--fg2)":"inherit"}}>{p}</b>:p)}</span>{past&&<span className="el">{el[k]}</span>}</div>})}</div>;
}

function App(){
  const [t,setTweak]=useTweaks(TWEAK_DEFAULTS);
  const [a,setA]=useState(BLANK);
  const [cur,setCur]=useState(0);
  const [far,setFar]=useState(0);
  const [seen,setSeen]=useState(()=>new Set());
  const [phase,setPhase]=useState("setup");
  const [built,setBuilt]=useState(false);
  const [run,setRun]=useState(0);
  const [left,setLeft]=useState(false);
  useEffect(()=>{if(left){const t=setTimeout(()=>setLeft(false),2200);return()=>clearTimeout(t)}},[left]);
  const sc=useRef(),fl=useRef();

  const toBottom=(smooth=true)=>{const s=sc.current;if(s)s.scrollTo({top:s.scrollHeight,behavior:smooth?"smooth":"auto"})};
  useEffect(()=>{const ro=new ResizeObserver(()=>toBottom());fl.current&&ro.observe(fl.current);return()=>ro.disconnect()},[run]);

  const reset=()=>{setA(BLANK);setCur(0);setFar(0);setSeen(new Set());setPhase("setup");setBuilt(false);setRun(r=>r+1)};
  const submit=patch=>{const na={...a,...patch};setA(na);const nx=far>cur?far:cur+1;setCur(nx);setFar(Math.max(far,nx))};
  const back=()=>{if(phase!=="setup")return;if(cur===0){setLeft(true);return}setCur(cur-1);setFar(cur-1)};
  useEffect(()=>{const k=e=>{if(e.key==="Escape")back()};window.addEventListener("keydown",k);return()=>window.removeEventListener("keydown",k)});
  const edit=i=>{if(phase!=="setup")return;setCur(i)};
  const finish=()=>{setPhase("building")};
  const markSeen=i=>setSeen(s=>s.has(i)?s:new Set(s).add(i));

  const prog=phase!=="setup"?(built?100:92):Math.round(cur/STEP_IDS.length*86)+4;
  const steps=STEP_IDS.slice(0,cur+1);

  const widget=id=>{switch(id){
    case "name":return <TextAsk value={a.name} placeholder="Company name" onSubmit={v=>submit({name:v})}/>;
    case "tz":return <TzAsk value={a.tz} detected={DETECTED} onSubmit={v=>submit({tz:v})}/>;
    case "addr":return <AddrAsk a={a} onSubmit={f=>submit(f)}/>;
    case "ids":return <IdsAsk a={a} onSubmit={f=>submit(f)}/>;
    case "op":return <Choice items={OB_OPS} value={a.op} onSubmit={v=>submit({op:v})} render={o=><span className="mods">{o.mods.map(m=><span key={m}>{m}</span>)}</span>}/>;
    case "sample":return <Choice items={[{id:true,t:"Load sample data",d:"A few customers, locations, equipment and shipments, so every screen has something to show."},{id:false,t:"Start empty",d:"A clean workspace. Add your own records from day one."}]} value={a.sample} onSubmit={v=>submit({sample:v})}
      render={o=>o.id===true&&<><span className="meters">{OB_SAMPLE.map(([k,n,m])=><span key={k} className="mt"><div>{k}<span>{n} of {m}</span></div><i><u style={{"--w":n/m*100+"%"}}></u></i></span>)}</span><span className="lim">Sample records count toward the free demo's limits, the same as records you create. Deleting one frees its slot.</span></>}/>;
    case "review":return <Review a={a} onEdit={edit} onFinish={finish}/>;
  }};

  return <div className={"ob"+(t.theme==="dark"?" dk":"")}>
    <button className="bk" onClick={back} disabled={phase!=="setup"} title="Back" aria-label="Back"><span className="bk-ic"><img className="bk-l" src="logo.png" alt=""/><span className="bk-a">{Ic.bk}</span></span></button>
    {left&&<div className="toast">Would return to the account details form</div>}
    <span className="tbar"><i style={{width:prog+"%"}}></i></span>
    <div className="room">
      <div className="scroll" ref={sc}>
        <div className="grid" key={run}>
          <main className="flow" ref={fl}>
            {steps.map((id,i)=>{const isCur=i===cur&&phase==="setup";const ans=i<cur||phase!=="setup"?ansText(id,a):undefined;
              return <section key={id} className="turn" data-screen-label={id}>
                <Typed segs={say(id,a,(t.firstName||"").trim())} animate={!seen.has(i)} speed={t.speed} onDone={()=>markSeen(i)}/>
                {isCur&&seen.has(i)&&<div className="ask">{widget(id)}</div>}
                {ans!==undefined&&id!=="review"&&<div className={"ans"+(i===cur-1&&phase==="setup"?" last":"")}>{phase==="setup"&&<button className="ans-e" onClick={()=>edit(i)}>{Ic.ed}Edit</button>}<span key={ans||"skip"} className={"ans-b"+(ans?"":" sk")}>{ans||"Skip for now"}</span></div>}
                {i===0&&cur===1&&phase==="setup"&&<div className="fix">Made a typo? Click Edit, or press <span className="kbd">esc</span> to go back.</div>}
                {id==="review"&&phase!=="setup"&&<div className="ans"><span className="ans-b">Finish setup</span></div>}
              </section>})}
            {phase!=="setup"&&<section className="turn" data-screen-label="build">
              <Typed segs={["Creating the workspace for ",{b:a.name},". This only takes a moment."]} animate={!seen.has(99)} speed={t.speed} onDone={()=>{markSeen(99)}}/>
              {seen.has(99)&&<Build a={a} speed={t.speed} onReady={()=>{setBuilt(true);setPhase("ready")}}/>}
              {built&&<div className="rdy"><span className="okr"><svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path className="draw" d="m3.5 8.5 3 3 6-7"/></svg></span><Burst/>
                <span className="rdy-t"><b>{(t.firstName||"").trim()?`You're all set, ${t.firstName.trim()}`:`${a.name} is ready`}</b><span>{a.sample?`${a.name} is live. Sample data is loaded and ready to explore.`:`${a.name} is live and ready for your first records.`}</span></span>
                <button className="bt ink">Open Trenova {Ic.ar}</button></div>}
            </section>}
          </main>
        </div>
      </div>
    </div>
    <TweaksPanel>
      <TweakSection label="Appearance"/>
      <TweakRadio label="Theme" value={t.theme} options={["dark","light"]} onChange={v=>setTweak("theme",v)}/>
      <TweakSection label="Guide"/>
      <TweakText label="User's first name" value={t.firstName} onChange={v=>setTweak("firstName",v)}/>
      <TweakSlider label="Typing speed" value={t.speed} min={0.5} max={4} step={0.25} unit="×" onChange={v=>setTweak("speed",v)}/>
      <TweakButton label="Restart onboarding" onClick={reset}/>
    </TweaksPanel>
  </div>;
}
ReactDOM.createRoot(document.getElementById("root")).render(<App/>);
