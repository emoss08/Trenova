const SECTIONS = [
  { id: "before", title: "Before the reply", sub: "The question is checked by the guard before any model sees it.", frames: [
    { label: "Checking the question", sub: "turn.status = guarding · usually under a second", el: (
      <Frame label="" dock={<EComp ring="check" send="stop" status={<><span className="ec-shield"><Ic n="shield" s={13} w={2} /></span><span className="shim">Checking the question…</span></>} />}>
        <Q>Post all 15 invoices and email the customers their totals</Q>
      </Frame>) },
    { label: "Question not accepted", sub: "event: refused · the guard's reason is kept", el: (
      <Frame label="" dock={<EComp value="" placeholder="Rephrase, or ask another agent…" />}>
        <Q muted tag="Not sent to the agent">Can you change Lena Park's pay rate to $32/hour?</Q>
        <ECard tone="neutral" icon="shield" title="This is outside what Billing Specialist can do" sub="Pay rates are handled by the Workforce Coordinator. Nothing was looked up or changed."
          actions={<><Btn k="ink">Ask Workforce Coordinator</Btn><Btn>Rephrase</Btn></>} />
      </Frame>) },
  ] },
  { id: "model", title: "When the model fails", sub: "Auto walks the organization's provider order; a pick is a preference, not a pin.", frames: [
    { label: "Retrying", sub: "event: retrying · reply starts over", el: (
      <Frame label="" dock={<EComp ring="retry" send="stop" status={<><Countdown from={4} /><span>Anthropic is overloaded · retrying</span><span className="ec-sp"></span><span className="ec-att">Attempt 2 of 3</span><button className="ec-link">Switch model</button></>} />}>
        <Q>Which customers have invoices over 60 days?</Q>
      </Frame>) },
    { label: "Answered by a fallback model", sub: "the reply says which model actually answered", el: (
      <Frame label="" dock={<EComp />}>
        <Q>Which customers have invoices over 60 days?</Q>
        <R><p>Three customers are past 60 days: <strong>Acme Manufacturing</strong> ($9,800), <strong>Harbor Supply Co.</strong> ($5,600) and <strong>Lakeside Grocers</strong> ($1,850).</p></R>
        <div className="ec-fb"><span className="ec-fbm off"><BrandMark id="anthropic" s={11} /></span><Ic n="chevR" s={10} /><span className="ec-fbm"><BrandMark id="openai" s={11} /></span><span>Answered by <b>GPT-5</b> · Claude Sonnet 4.5 didn't respond</span></div>
      </Frame>) },
    { label: "Stopped partway", sub: "event: error after text arrived · partial reply kept", el: (
      <Frame label="" dock={<EComp />}>
        <Q>Summarize this week's detention by customer</Q>
        <R fade><p>Detention ran <strong>114.9 hours</strong> last week across six customers. Acme Manufacturing accounts for over a third of it, mostly from two long holds at their Columbus plant on Tuesday and</p></R>
        <ECard tone="err" compact title="Reply stopped partway" sub="The connection to Anthropic dropped after 41 words. Nothing was changed."
          actions={<><Btn k="ink"><Ic n="replay" s={12} />Try again</Btn><Btn>Continue from here</Btn><Btn>Copy what arrived</Btn></>} />
      </Frame>) },
    { label: "No model could answer", sub: "every endpoint in the order failed", el: (
      <Frame label="" dock={<EComp value="Which customers have invoices over 60 days?" />}>
        <Q>Which customers have invoices over 60 days?</Q>
        <ECard tone="err" title="No model could answer" sub="Desk tried every model your organization set up. Your message is saved.">
          <div className="ec-provs">{PROV.map(([p, m, s, d]) => <div key={p}><BrandMark id={p} s={13} /><b>{m}</b><span className="ec-pst">{s}</span><em>{d}</em></div>)}</div>
          <div className="ec-acts"><Btn k="ink"><Ic n="replay" s={12} />Try again</Btn><Btn>Check provider status</Btn></div>
        </ECard>
      </Frame>) },
  ] },
  { id: "tools", title: "Tools", sub: "Each refusal is a stop you can act on, so each is said in its own words instead of “failed”.", frames: [
    { label: "Steps that didn't go through", sub: "verdicts: failed · denied · over_budget · invalid · duplicate", el: (
      <Frame label="" dock={<EComp />}>
        <R><p>I found 14 invoices ready to post, but couldn't check credit holds or the customers' payment history, so I haven't drafted the posting yet.</p></R>
        <StepFails />
      </Frame>) },
    { label: "The change didn't go through", sub: "a write failed after you approved it", el: (
      <Frame label="" dock={<EComp />}>
        <ECard tone="err" title="Posted 73 of 120 invoices · 47 didn't go through" sub="The 47 stayed as drafts. The 73 that posted are final.">
          <BulkList items={FAILED_POST} noun="invoices" onOpenTable={() => {}} />
          <div className="ec-acts"><Btn k="ink">Ask agent to fix these 47</Btn><Btn>Open in billing</Btn></div>
        </ECard>
      </Frame>) },
  ] },
  { id: "approvals", title: "Approvals", sub: "The preview runs the write as a dry run before you can approve it.", frames: [
    { label: "Would be refused as it stands", sub: "preview code: would_fail", el: (
      <Frame label="" dock={<>
        <div className="dcx ec-would"><span className="dcx-i"><Ic n="alert" s={15} w={2} /></span><span className="dcx-t"><b>Post 15 invoices <span className="dcx-s">would be refused as it stands</span></b><span className="dcx-d">38 of 120 · 22 locked by a dispute · 11 in a closed period · 5 already billed</span></span><button className="bt sm">Review 38</button><button className="apv-b ec-fix">Approve 82, skip 38</button></div>
        <EComp />
      </>}>
        <R><p>Everything's ready. I've drafted the posting for all <strong>15 invoices</strong> totaling <strong>$47,030.50</strong>.</p></R>
      </Frame>) },
    { label: "Proposal is out of date", sub: "proposal hold · records changed since it was drafted", el: (
      <Frame label="" dock={<>
        <div className="dcx ec-stale"><span className="dcx-i"><Ic n="undo" s={15} w={2} /></span><span className="dcx-t"><b>Assign biller <span className="dcx-s">on 11 items</span></b><span className="dcx-d">3 of these items changed after it was drafted · Jordan Pike assigned them</span></span><button className="bt sm">Not now</button><button className="apv-b">Redraft with 8 items</button></div>
        <EComp />
      </>}>
        <R><p>I've drafted assigning you as biller on the <strong>11 items</strong> that don't have one.</p></R>
      </Frame>) },
  ] },
  { id: "limits", title: "Limits & usage", sub: "Every limit says which limit, when it resets, and who can raise it.", frames: [
    { label: "Sending too fast", sub: "429 · rate limited", el: (
      <Frame label="" dock={<EComp value="And the ones over 90?" send="count" status={<><Countdown from={18} /><span>You're sending messages quickly · send again in a moment</span></>} />}>
        <Q>Which customers have invoices over 60 days?</Q>
        <R><p>Three customers: Acme Manufacturing, Harbor Supply Co. and Lakeside Grocers.</p></R>
      </Frame>) },
    { label: "Agent's daily request limit", sub: "dailyRequestLimit reached", el: (
      <Frame label="" dock={<EComp off={<><Ic n="lock" s={13} w={2} /><span><b>Billing Specialist has reached today's 200-request limit.</b> It resets at midnight.</span><button className="ec-link">Ask General Assistant</button></>} />}>
        <R><p>Done. Every item is ready, so I've drafted the posting.</p></R>
      </Frame>) },
    { label: "Budget almost used", sub: "agent budget status ≥ 90%", el: (
      <Frame label="" dock={<>
        <div className="ec-meter"><div className="ec-mt"><span>Billing Specialist has used <b>92%</b> of October's budget</span><span className="ax-num">$460 / $500</span></div><div className="ec-mb"><i style={{ width: "92%" }}></i></div></div>
        <EComp />
      </>}>
        <R><p>Three customers are past 60 days: Acme Manufacturing, Harbor Supply Co. and Lakeside Grocers.</p></R>
      </Frame>) },
    { label: "Budget used up", sub: "monthlyBudgetUsd reached", el: (
      <Frame label="" dock={<EComp off={<><Ic n="alert" s={13} w={2} /><span><b>October's budget for Billing Specialist is used up.</b> It resets Nov 1, or an admin can raise it.</span><button className="ec-link">Ask an admin</button></>} />}>
        <R><p>Three customers are past 60 days: Acme Manufacturing, Harbor Supply Co. and Lakeside Grocers.</p></R>
      </Frame>) },
    { label: "Your usage limit", sub: "per-person allowance for the period", el: (
      <Frame label="" dock={<EComp off={<><Ic n="lock" s={13} w={2} /><span><b>You've used your AI allowance for this period.</b> It refreshes in 4 days.</span><button className="ec-link">Request more</button></>} note={<span>1,000 of 1,000 messages · resets Oct 6</span>} />}>
        <Q>Draft this week's AR summary for Acme</Q>
      </Frame>) },
    { label: "Conversation too long", sub: "context window exceeded for the model", el: (
      <Frame label="" dock={<>
        <ECard tone="info" icon="info" compact title="This conversation is too long for Claude Sonnet 4.5" sub="Start fresh and Desk carries over a summary and your 3 pinned artifacts."
          actions={<><Btn k="ink">Continue in a new conversation</Btn><Btn><BrandMark id="gemini" s={11} />Switch to Gemini 2.5 Pro</Btn></>} />
        <EComp />
      </>}>
        <R><p>That's the full list for September.</p></R>
      </Frame>) },
  ] },
  { id: "docs", title: "Documents", sub: "Attachments upload before you send: up to 5 files per message, 25 MB each. Send waits until every file is ready.", frames: [
    { label: "Every upload state", sub: "uploading · ready · failed · too large · unsupported · password-protected", el: (
      <Frame label="" dock={<div className="cmp ec"><div className="ar"><div className="ar-l">
        {[
          { id: "1", name: "BOL-778.pdf", ext: "pdf", size: 1240000, status: "uploading", progress: 0.55 },
          { id: "2", name: "rate-con-acme-77812.pdf", ext: "pdf", size: 412000, status: "ready" },
          { id: "3", name: "invoice-sept.pdf", ext: "pdf", size: 412000, status: "error", err: "Upload failed · connection dropped", retry: true },
          { id: "4", name: "scan-batch-sept.pdf", ext: "pdf", size: 61000000, status: "error", err: "Too large · max 25 MB" },
          { id: "5", name: "carrier-packet.zip", ext: "zip", size: 8000000, status: "error", err: "Desk can't read .zip files" },
          { id: "6", name: "locked-invoice.pdf", ext: "pdf", size: 320000, status: "error", err: "Password-protected · Desk can't open it" },
        ].map(a => <FileChip key={a.id} a={a} onRemove={() => {}} onRetry={() => {}} />)}
      </div></div>
      <textarea readOnly rows={2} placeholder="Ask about these files…"></textarea>
      <div className="cmp-b"><button className="ib"><Ic n="plus" s={16} /></button><span className="apill"><AgentMark s={10} />Billing Specialist</span><span style={{ flex: 1 }}></span><button className="send" disabled title="Waiting for the files to finish uploading"><span className="send-wait"></span></button></div></div>} >
        <R><p>Drop rate confirmations, BOLs or PODs here and I'll read them.</p></R>
      </Frame>) },
    { label: "Unreadable scan", sub: "extraction ran but confidence is too low to use", el: (
      <Frame label="" dock={<EComp />}>
        <Q>What's in this?</Q>
        <ECard tone="warn" icon="alert" title="I couldn't read most of pod-photo.jpg" sub="The photo is blurred and taken at an angle. I could only make out the shipment number, SEED-SHP-008."
          actions={<><Btn k="ink">Upload a clearer photo</Btn><Btn>Use what I could read</Btn></>} />
      </Frame>) },
  ] },
  { id: "access", title: "Access & availability", sub: "cannotContinueReason: AgentDeleted · AgentDisabled · AgentNotConversational · NoAccess", frames: [
    { label: "Agent turned off", sub: "AgentDisabled · read-only", el: (
      <Frame label="" dock={<EComp off={<><Ic n="lock" s={13} w={2} /><span><b>Jordan Pike turned off Billing Specialist on Sep 30.</b> You can still read this conversation.</span><button className="ec-link">Start with another agent</button></>} />}>
        <R><p>Done. Every item is ready, so I've drafted the posting.</p></R>
      </Frame>) },
    { label: "No access to this agent", sub: "NoAccess · 403", el: (
      <Frame label="" dock={<EComp off={<><Ic n="lock" s={13} w={2} /><span><b>You no longer have access to Billing Specialist.</b> Your role changed on Oct 1.</span><button className="ec-link">Request access</button></>} />}>
        <R><p>Here's the full list with bill-to customers and amounts.</p></R>
      </Frame>) },
    { label: "No model set up", sub: "organization has no assistant provider", el: (
      <Frame label="" dock={<EComp off={<><Ic n="info" s={13} w={2} /><span><b>No AI model is set up for your organization yet.</b> An admin can connect one in Agent Control.</span><button className="ec-link">Open Agent Control</button></>} />}>
        <div className="ec-empty">Desk</div>
      </Frame>) },
    { label: "You're offline", sub: "network lost · messages queue", el: (
      <Frame label="" top={<div className="ec-off"><span className="ec-offd"></span>You're offline · messages will send when you reconnect</div>} dock={<EComp />}>
        <Q>Which loads are at risk from the storm today?</Q>
        <div className="ec-queued"><Ic n="undo" s={11} w={2.2} />Waiting to send</div>
      </Frame>) },
  ] },
];

function ErrorsGallery() {
  const [t, setTweak] = useTweaks(/*EDITMODE-BEGIN*/{ "theme": "dark" }/*EDITMODE-END*/);
  return (
    <div className={"dsk ec-page" + (t.theme === "dark" ? " dk" : "")}>
      <header className="ec-head">
        <h1>Desk · errors, warnings & limits</h1>
        <p>Every way a turn can stop, built from the states in the Trenova assistant code: the guard, the stream events, tool verdicts, previews, budgets and access.</p>
        <nav>{SECTIONS.map(s => <a key={s.id} href={"#" + s.id}>{s.title}</a>)}</nav>
      </header>
      {SECTIONS.map(s => (
        <section key={s.id} id={s.id} className="ec-sec">
          <div className="ec-sech"><h2>{s.title}</h2><p>{s.sub}</p></div>
          <div className="ec-grid">{s.frames.map(f => <div key={f.label} className="ec-cell"><div className="ec-cap"><b>{f.label}</b><span>{f.sub}</span></div>{f.el}</div>)}</div>
        </section>
      ))}
      <TweaksPanel>
        <TweakSection label="Appearance" />
        <TweakRadio label="Theme" value={t.theme} options={["dark", "light"]} onChange={v => setTweak("theme", v)} />
      </TweaksPanel>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<ErrorsGallery />);
