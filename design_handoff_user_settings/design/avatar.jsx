// Gradient avatars. Seeded from the user id so everyone gets a stable default
// (like avatar.vercel.sh); users can pick a preset instead.
const avHash = str => { let h = 2166136261; for (let i = 0; i < str.length; i++) { h ^= str.charCodeAt(i); h = Math.imul(h, 16777619); } return h >>> 0; };
const AV_PRESETS = [
  ["dusk", "Dusk", [258, 306, 20]], ["lagoon", "Lagoon", [190, 232, 148]], ["citrus", "Citrus", [76, 40, 120]],
  ["orchid", "Orchid", [306, 340, 258]], ["ember", "Ember", [25, 55, 350]], ["mint", "Mint", [160, 120, 200]], ["slate", "Slate", [240, 270, 210], 0.06],
];
function avMesh([a, b, c], chroma = 0.17) {
  const k = chroma;
  return [
    `radial-gradient(at 18% 22%, oklch(0.82 ${k} ${a}) 0, transparent 58%)`,
    `radial-gradient(at 86% 18%, oklch(0.74 ${k + 0.02} ${b}) 0, transparent 55%)`,
    `radial-gradient(at 70% 88%, oklch(0.66 ${k + 0.02} ${c}) 0, transparent 60%)`,
    `radial-gradient(at 12% 92%, oklch(0.88 ${k * 0.7} ${(a + b) / 2}) 0, transparent 50%)`,
    `linear-gradient(135deg, oklch(0.8 ${k} ${a}), oklch(0.68 ${k} ${c}))`,
  ].join(",");
}
function avSeedHues(seed) { const h = avHash(seed); const a = h % 360; return [a, (a + 40 + (h >> 9) % 50) % 360, (a + 300 - (h >> 17) % 60) % 360]; }
function avBackground(s) {
  const p = AV_PRESETS.find(x => x[0] === s.avatar);
  return p ? avMesh(p[2], p[3]) : avMesh(avSeedHues(s.username || s.name));
}
function Avatar({ s, size = "md", children }) {
  return (
    <span className={"us-av " + size} style={{ backgroundImage: s.photo ? undefined : avBackground(s) }}>
      {s.photo && <span className="ph"></span>}
      {children}
    </span>
  );
}
function AvatarPicker({ s, set }) {
  return (
    <div className="avp" role="radiogroup" aria-label="Avatar style">
      <button role="radio" aria-checked={s.avatar === "auto"} className={s.avatar === "auto" ? "on" : ""} title="Auto — from your username" style={{ backgroundImage: avMesh(avSeedHues(s.username || s.name)) }} onClick={() => set("avatar", "auto")}><span>Auto</span></button>
      {AV_PRESETS.map(([k, l, h, c]) => <button key={k} role="radio" aria-checked={s.avatar === k} className={s.avatar === k ? "on" : ""} title={l} aria-label={l} style={{ backgroundImage: avMesh(h, c) }} onClick={() => set("avatar", k)}></button>)}
      <button className="avp-r" title="Shuffle" aria-label="Shuffle" onClick={() => { const opts = AV_PRESETS.map(p => p[0]).filter(k => k !== s.avatar); set("avatar", opts[Math.floor(Math.random() * opts.length)]); }}><UI n="shuffle" s={13} /></button>
    </div>
  );
}
Object.assign(window, { Avatar, AvatarPicker, avBackground, avMesh, avSeedHues });
