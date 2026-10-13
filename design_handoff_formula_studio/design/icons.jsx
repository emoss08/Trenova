const FSP = {
  back: <path d="M15 6l-6 6 6 6"></path>,
  chevD: <path d="M6 9l6 6 6-6"></path>,
  chevR: <path d="M9 6l6 6-6 6"></path>,
  x: <path d="M6 6l12 12M18 6L6 18"></path>,
  check: <path d="M5 12.5l4.5 4.5L19 7"></path>,
  plus: <path d="M12 5v14M5 12h14"></path>,
  minus: <path d="M5 12h14"></path>,
  trash: <path d="M4.5 7h15M10 7V5h4v2M6.5 7l1 12h9l1-12M10 11v5M14 11v5"></path>,
  search: <><circle cx="11" cy="11" r="6.5"></circle><path d="M20 20l-4-4"></path></>,
  code: <path d="M8.5 7L3.5 12l5 5M15.5 7l5 5-5 5M13.5 4.5l-3 15"></path>,
  braces: <path d="M8 4.5c-2 0-2.5 1-2.5 3v2c0 1.2-.8 2.5-2 2.5 1.2 0 2 1.3 2 2.5v2c0 2 .5 3 2.5 3M16 4.5c2 0 2.5 1 2.5 3v2c0 1.2.8 2.5 2 2.5-1.2 0-2 1.3-2 2.5v2c0 2-.5 3-2.5 3"></path>,
  beaker: <path d="M9.5 3.5h5M10.5 3.5v6l-5 8.5a1.5 1.5 0 0 0 1.3 2.5h10.4a1.5 1.5 0 0 0 1.3-2.5l-5-8.5v-6M7.5 15h9"></path>,
  play: <path d="M8 5.5v13l10-6.5z"></path>,
  pin: <path d="M9 4h6l-1 5 3 3H7l3-3zM12 15v5.5"></path>,
  book: <path d="M5 4.5h5.5a2 2 0 0 1 2 2v13a1.8 1.8 0 0 0-1.8-1.8H5zM19 4.5h-4.5a2 2 0 0 0-2 2v13a1.8 1.8 0 0 1 1.8-1.8H19z"></path>,
  layers: <path d="M12 4l8 4-8 4-8-4zM4 12l8 4 8-4M4 16l8 4 8-4"></path>,
  shield: <path d="M12 3l7 3v5.5c0 4.2-2.9 7.7-7 8.5-4.1-.8-7-4.3-7-8.5V6z"></path>,
  file: <path d="M7 3.5h7l4 4V20.5H7zM14 3.5V8h4M10 12.5h5M10 16h5"></path>,
  input: <><rect x="3.5" y="6.5" width="17" height="11" rx="2"></rect><path d="M8 10v4M11 10h1.5M11 14h1.5"></path></>,
  more: <path d="M12 6.5v.01M12 12v.01M12 17.5v.01" strokeWidth="2.6"></path>,
  clock: <><circle cx="12" cy="12" r="8.5"></circle><path d="M12 7.5V12l3 2"></path></>,
  rewind: <><path d="M4 12a8 8 0 1 0 2.3-5.6M4 4.5v4h4"></path><path d="M12 8v4l2.5 1.5"></path></>,
  branch: <><circle cx="7" cy="5.5" r="2"></circle><circle cx="7" cy="18.5" r="2"></circle><circle cx="17" cy="8" r="2"></circle><path d="M7 7.5v9M17 10c0 4-10 2.5-10 6.5"></path></>,
  download: <path d="M12 4v11M7.5 10.5L12 15l4.5-4.5M5 19.5h14"></path>,
  upload: <path d="M12 15V4M7.5 8.5L12 4l4.5 4.5M5 19.5h14"></path>,
  send: <path d="M4.5 12L20 4.5 15.5 20l-3.5-6.5zM12 13.5L20 4.5"></path>,
  users: <><circle cx="9" cy="8.5" r="3.2"></circle><path d="M3.5 19c.9-3 3-4.5 5.5-4.5s4.6 1.5 5.5 4.5M15.5 5.5a3 3 0 0 1 0 6M17.5 14.8c1.5.6 2.5 2 3 4.2"></path></>,
  alert: <path d="M12 4.5L21 19.5H3zM12 10v4M12 16.8v.01"></path>,
  info: <><circle cx="12" cy="12" r="9"></circle><path d="M12 11v5.5M12 7.5v.01"></path></>,
  okc: <><circle cx="12" cy="12" r="9"></circle><path d="M8 12.5l2.5 2.5L16 9.5"></path></>,
  xc: <><circle cx="12" cy="12" r="9"></circle><path d="M9 9l6 6M15 9l-6 6"></path></>,
  undo: <path d="M9 14l-5-5 5-5M4 9h10a6 6 0 0 1 0 12h-3"></path>,
  copy: <><rect x="8" y="8" width="11" height="11" rx="2"></rect><path d="M5 15V6a1 1 0 0 1 1-1h9"></path></>,
  truck: <path d="M3 7h11v9H3zM14 10h3.5l2.5 3v3h-6M7 18.5a1.5 1.5 0 1 0 0-.1M17 18.5a1.5 1.5 0 1 0 0-.1"></path>,
  spark: <><path d="M2.7 10.3a2.41 2.41 0 0 0 0 3.41l7.59 7.59a2.41 2.41 0 0 0 3.41 0l7.59-7.59a2.41 2.41 0 0 0 0-3.41l-7.59-7.59a2.41 2.41 0 0 0-3.41 0Z"></path><circle cx="12" cy="12" r="2" fill="currentColor" stroke="none"></circle></>,
  wand: <path d="M4 20L15 9M13 7l1.5-1.5M17 11l1.5-1.5M15 4v2M20 9h-2M18.5 4.5l-1.5 1.5"></path>,
  arrowR: <path d="M5 12h14M13 6l6 6-6 6"></path>,
  arrowUp: <path d="M12 19V5M6 11l6-6 6 6"></path>,
  refresh: <path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5v4h-4"></path>,
  eye: <><path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z"></path><circle cx="12" cy="12" r="2.8"></circle></>,
  table: <><rect x="4" y="5" width="16" height="14" rx="2"></rect><path d="M4 10h16M10 10v9"></path></>,
  msg: <path d="M5 5.5h14a1.5 1.5 0 0 1 1.5 1.5v8.5A1.5 1.5 0 0 1 19 17h-8l-4.5 3.5V17H5a1.5 1.5 0 0 1-1.5-1.5V7A1.5 1.5 0 0 1 5 5.5z"></path>,
  sliders: <path d="M4 7h9M17 7h3M4 17h3M11 17h9M15 5v4M9 15v4"></path>,
  stop: <rect x="7" y="7" width="10" height="10" rx="1.5"></rect>,
};
function FI({ n, s = 14, w = 1.7, style }) {
  return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={w} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={style}>{FSP[n]}</svg>;
}
// Nova's mark: the gradient ring used across Trenova (Triage, AI Control, Desk).
function NovaMark({ s = 14, busy }) {
  return <span className={"dm" + (busy ? " spin" : "")} style={{ width: s, height: s }}></span>;
}
Object.assign(window, { FI, NovaMark });
