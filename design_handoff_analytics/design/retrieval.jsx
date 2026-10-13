function RetrievalTab({ S, A }) {
  const R = S.retr; const emb = routeOf(TASKS.find(t => t.k === "Embedding"), S.providers).first;
  const model = emb ? (emb.id === "ollama" ? "nomic-embed-text" : emb.model) : null;
  const ollama = S.providers.find(p => p.id === "ollama" && p.on);
  const on = R.src.filter(s => s.on);
  const idx = on.reduce((t, s) => t + s.idx, 0), wait = on.reduce((t, s) => t + Math.max(0, s.total - s.idx - s.fail - s.skip), 0), fail = on.reduce((t, s) => t + s.fail, 0);
  const running = emb && !R.paused && wait > 0;
  const stOf = s => !s.on ? ["Off", ""] : !emb ? ["Words only", "w"] : R.paused ? ["Paused", "w"] : s.total - s.idx - s.fail - s.skip > 0 ? ["Indexing", "b"] : ["Up to date", "k"];
  return (
    <div className="tabp">
      {!emb ? (
        <section className="hero rh">
          <div className="hero-s"><span className="who"><span className="dm"></span><b>Nova</b><span>Retrieval</span></span>
            <p className="say">Agents search by <b>keyword only</b>. Nothing handles the Embedding task, so <b>{wait.toLocaleString()} items</b> are waiting to be searchable by meaning.</p></div>
          <div className="hc">{ollama ? <><button className="btn ink lg" onClick={() => { A.toggleTask("ollama", "Embedding"); A.toast("Local Ollama now handles Embedding · indexing started"); }}><Ic n="bolt" s={13}></Ic>Use Local Ollama</button><span>nomic-embed-text · on your network</span></> : <><button className="btn ink lg" onClick={() => A.go("providers", null, "routing")}>Route Embedding</button><span>Pick a provider on Providers</span></>}</div>
        </section>
      ) : (
        <section className="hero rh">
          <div className="hero-s"><span className="who"><span className={"dm" + (running ? " spin" : "")}></span><b>Nova</b><span>Retrieval</span></span>
            <p className="say" key={running ? "r" : R.paused ? "p" : "d"}>{R.paused ? <>Indexing is <b className="t-w">paused</b>. Agents search by keyword until it resumes.</> : running ? <>Indexing under <b className="mono">{model}</b>. <b>{wait.toLocaleString()}</b> items to go — searches use keywords for anything not done yet.</> : <>Everything is searchable by meaning under <b className="mono">{model}</b>{fail ? <>, except <span className="ref d" onClick={() => A.scrollTo("rfail")}>{fail} items that failed</span></> : ""}.</>}</p></div>
          <div className="hc">{R.paused ? <button className="btn ink lg" onClick={() => A.setRetr({ paused: false }, "Indexing resumed")}><Ic n="play" s={12}></Ic>Resume indexing</button> : <button className="btn lg" onClick={() => A.setRetr({ paused: true }, "Indexing paused")}><Ic n="pause" s={13} w={2.2}></Ic>Pause indexing</button>}<span>Routed to {emb.n}</span></div>
        </section>
      )}
      <section className="wk w5">
        <div className="wk-c"><span className="lbl">Indexed</span><b className="big mono">{idx.toLocaleString()}</b><span className="sub">searchable by meaning</span></div>
        <div className="wk-c"><span className="lbl">Waiting</span><b className={"big mono" + (wait ? "" : " dim")}>{wait.toLocaleString()}</b><span className="sub">{running ? "indexing now" : "to be indexed"}</span></div>
        <div className="wk-c"><span className="lbl">Failed</span><b className={"big mono" + (fail ? " t-d" : " dim")}>{fail}</b><span className="sub">{fail ? "see below" : "nothing failed"}</span></div>
        <div className="wk-c"><span className="lbl">Cost this month</span><b className="big mono">${R.spent.toFixed(2)}</b><span className="ubar bud"><i style={{ width: Math.min(100, R.spent / R.budget * 100) + "%" }}></i></span><span className="sub">of ${R.budget.toFixed(2)} budget</span></div>
        <div className="wk-c"><span className="lbl">Last run</span><b className={"big" + (R.last ? "" : " dim mono")}>{R.last || "—"}</b><span className="sub">{R.last ? "indexer round" : "nothing indexed yet"}</span></div>
      </section>
      <div className="ov">
        <div className="ov-m">
          <section className="sec">
            <SecH t="Sources" r={model ? <span className="sh2-n mono">{model}</span> : null}></SecH>
            <div className="srcs">{R.src.map(s => { const [l, c] = stOf(s), w = Math.max(0, s.total - s.idx - s.fail - s.skip), T = Math.max(1, s.total); return (
              <div key={s.k} className={"src" + (s.on ? "" : " dim")}>
                <span className="src-i"><Ic n={s.ic} s={15}></Ic></span>
                <div className="src-m">
                  <div className="src-h"><b>{s.n}</b><span className={"tg " + c}>{l === "Indexing" && <i className="spn"></i>}{l}</span></div>
                  <p>{s.d}</p>
                  <div className="sbar"><i className="ix" style={{ width: s.idx / T * 100 + "%" }}></i><i className="fx" style={{ width: s.fail / T * 100 + "%" }}></i><i className="sk" style={{ width: s.skip / T * 100 + "%" }}></i></div>
                  <div className="src-n mono"><span>{s.idx.toLocaleString()} indexed</span><span>{w.toLocaleString()} waiting</span>{s.fail > 0 && <span className="t-d">{s.fail} failed</span>}{s.skip > 0 && <span title="Retired, superseded, empty, or too sensitive to send. Still found by their words.">{s.skip} skipped</span>}<span className="dim">of {s.total.toLocaleString()}</span></div>
                </div>
                <div className="src-c">
                  <button className="btn sm" disabled={!s.on || !emb} title={!emb ? "Nothing is indexed yet" : !s.on ? "Turn the source on to index it" : ""} onClick={() => A.reindex(s.k)}><Ic n="refresh" s={12}></Ic>Re-index</button>
                  <Switch on={s.on} label={"Index " + s.n} onChange={v => A.setSrc(s.k, { on: v })}></Switch>
                </div>
              </div>); })}</div>
          </section>
          <section className="sec" id="rfail">
            <DT rows={R.fails} name="item" find={f => f.item + f.err} empty="No failed items yet" cols={[{ k: "i", l: "Item", c: f => <div className="tl"><b>{f.item}</b><span>{f.src}</span></div>, sv: f => f.item }, { k: "s", l: "Status", c: () => <Bd m={["Failed", "d"]}></Bd> }, { k: "e", l: "Error", w: "36%", c: f => <span className="mu">{f.err}</span> }, { k: "n", l: "Attempts", r: true, c: f => <span className="mono">{f.n}</span>, sv: f => f.n }, { k: "l", l: "Last attempt", c: () => <span className="mu">just now</span>, hide: 1 }, { k: "x", l: "", r: true, c: f => <button className="btn sm" onClick={e => { e.stopPropagation(); A.retryFail(f); }}><Ic n="refresh" s={12}></Ic>Retry</button> }]}></DT>
          </section>
        </div>
        <aside className="ov-a">
          <section className="sec">
            <SecH t="Settings" r={<span className="sh2-n">From the indexer's next round</span>}></SecH>
            <div className="pol">
              <div className="po"><div className="po-h"><Ic n="dollar" s={14}></Ic><b>Monthly indexing budget</b></div>
                <div className="bud-i"><span>$</span><input className="mono" defaultValue={R.budget.toFixed(2)} onBlur={e => { const v = parseFloat(e.target.value); if (v > 0 && v !== R.budget) A.setRetr({ budget: v }, "Budget set to $" + v.toFixed(2)); }}></input><em>per month</em></div>
                <p>Indexing stops for the month when it's reached. Searches count against each agent's own budget, not this one.</p></div>
              <div className="po"><div className="po-h"><Ic n="pause" s={14}></Ic><b>Pause indexing</b><span className="sp"></span><Switch on={R.paused} label="Pause indexing" onChange={v => A.setRetr({ paused: v }, v ? "Indexing paused" : "Indexing resumed")}></Switch></div>
                <p>Nothing new is indexed, and agents search by keyword until it's resumed.</p></div>
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}
Object.assign(window, { RetrievalTab });
