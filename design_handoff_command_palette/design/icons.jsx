const CPI = {
  search: <><circle cx="11" cy="11" r="6.5"></circle><path d="M20 20l-4-4"></path></>,
  x: <path d="M6 6l12 12M18 6L6 18"></path>,
  check: <path d="M5 12.5l4.5 4.5L19 7"></path>,
  plus: <path d="M12 5v14M5 12h14"></path>,
  chevR: <path d="M9 6l6 6-6 6"></path>,
  arrowR: <path d="M5 12h14M13 6l6 6-6 6"></path>,
  enter: <path d="M19 5v6a3 3 0 0 1-3 3H6M10 10l-4 4 4 4"></path>,
  upRight: <path d="M7 17L17 7M8.5 7H17v8.5"></path>,
  link: <path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"></path>,
  copy: <><rect x="8" y="8" width="11" height="11" rx="2"></rect><path d="M5 15V6a1 1 0 0 1 1-1h9"></path></>,
  pin: <path d="M9 4h6l-1 5 3 3H7l3-3zM12 15v5.5"></path>,
  truck: <path d="M3 7h11v9H3zM14 10h3.5l2.5 3v3h-6M7 18.5a1.5 1.5 0 1 0 0-.1M17 18.5a1.5 1.5 0 1 0 0-.1"></path>,
  building: <path d="M4 20.5h16M6 20.5V5.5a1 1 0 0 1 1-1h7a1 1 0 0 1 1 1v15M15 9.5h3a1 1 0 0 1 1 1v10M9 8.5h3M9 12h3M9 15.5h3"></path>,
  user: <><circle cx="12" cy="8.5" r="3.5"></circle><path d="M5 20c1-3.6 3.7-5.5 7-5.5s6 1.9 7 5.5"></path></>,
  users: <><circle cx="9" cy="8.5" r="3.2"></circle><path d="M3.5 19c.9-3 3-4.5 5.5-4.5s4.6 1.5 5.5 4.5M15.5 5.5a3 3 0 0 1 0 6M17.5 14.8c1.5.6 2.5 2 3 4.2"></path></>,
  file: <path d="M7 3.5h7l4 4V20.5H7zM14 3.5V8h4M10 12.5h5M10 16h5"></path>,
  page: <><rect x="4" y="4" width="16" height="16" rx="2.5"></rect><path d="M4 9h16M9 9v11"></path></>,
  gear: <><circle cx="12" cy="12" r="3"></circle><path d="M12 3.5v2.2M12 18.3v2.2M20.5 12h-2.2M5.7 12H3.5M18 6l-1.6 1.6M7.6 16.4L6 18M18 18l-1.6-1.6M7.6 7.6L6 6"></path></>,
  alert: <path d="M12 4.5L21 19.5H3zM12 10v4M12 16.8v.01"></path>,
  dollar: <path d="M12 3.5v17M16.5 7.5c-.6-1.5-2.3-2.5-4.5-2.5-2.5 0-4.5 1.3-4.5 3.4 0 4.6 9 2.4 9 7.1 0 2.1-2 3.5-4.5 3.5-2.4 0-4.2-1.1-4.8-2.8"></path>,
  book: <path d="M5 4.5h5.5a2 2 0 0 1 2 2v13a1.8 1.8 0 0 0-1.8-1.8H5zM19 4.5h-4.5a2 2 0 0 0-2 2v13a1.8 1.8 0 0 1 1.8-1.8H19z"></path>,
  formula: <path d="M8.5 7L3.5 12l5 5M15.5 7l5 5-5 5M13.5 4.5l-3 15"></path>,
  clock: <><circle cx="12" cy="12" r="8.5"></circle><path d="M12 7.5V12l3 2"></path></>,
  trailer: <><rect x="3" y="6.5" width="18" height="9" rx="1.5"></rect><path d="M7 9.5v3M11 9.5v3M15 9.5v3M7 18.5a1.5 1.5 0 1 0 0-.1M17 18.5a1.5 1.5 0 1 0 0-.1"></path></>,
  inbox: <path d="M3.5 13.5l2.5-8h12l2.5 8v5a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1zM3.5 13.5h5l1 2.5h5l1-2.5h5"></path>,
  chart: <path d="M4 20h16M7 16v-5M12 16V6M17 16v-8"></path>,
  plug: <path d="M9 3.5v4M15 3.5v4M6.5 7.5h11v3a5.5 5.5 0 0 1-11 0zM12 16v4.5"></path>,
  mail: <><rect x="3.5" y="5.5" width="17" height="13" rx="2"></rect><path d="M4 7l8 6 8-6"></path></>,
  shield: <path d="M12 3l7 3v5.5c0 4.2-2.9 7.7-7 8.5-4.1-.8-7-4.3-7-8.5V6z"></path>,
  desk: <path d="M5 5.5h14a1.5 1.5 0 0 1 1.5 1.5v8.5A1.5 1.5 0 0 1 19 17h-8l-4.5 3.5V17H5a1.5 1.5 0 0 1-1.5-1.5V7A1.5 1.5 0 0 1 5 5.5z"></path>,
  layout: <><rect x="3.5" y="4.5" width="17" height="15" rx="2"></rect><path d="M9 4.5v15"></path></>,
  keyboard: <><rect x="2.5" y="6" width="19" height="12" rx="2"></rect><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10"></path></>,
  bell: <path d="M6 16.5V11a6 6 0 0 1 12 0v5.5l1.5 2h-15zM10 20.5a2 2 0 0 0 4 0"></path>,
  checks: <path d="M2.5 12.5l4 4L15 8M10.5 16.5l1 0L21 7"></path>,
  sun: <><circle cx="12" cy="12" r="3.8"></circle><path d="M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M5.3 18.7l1.4-1.4M17.3 6.7l1.4-1.4"></path></>,
  moon: <path d="M19.5 14.5A8 8 0 0 1 9.5 4.5a8 8 0 1 0 10 10z"></path>,
  monitor: <><rect x="3" y="4.5" width="18" height="12" rx="2"></rect><path d="M8.5 20h7M12 16.5V20"></path></>,
  logout: <path d="M14 4.5h4a1.5 1.5 0 0 1 1.5 1.5v12a1.5 1.5 0 0 1-1.5 1.5h-4M10 16.5L5.5 12 10 7.5M5.5 12H15"></path>,
  upload: <path d="M12 15V4M7.5 8.5L12 4l4.5 4.5M5 19.5h14"></path>,
  download: <path d="M12 4v11M7.5 10.5L12 15l4.5-4.5M5 19.5h14"></path>,
  eye: <><path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z"></path><circle cx="12" cy="12" r="2.8"></circle></>,
  calendar: <><rect x="4" y="5.5" width="16" height="14" rx="2"></rect><path d="M4 10h16M8.5 3.5v4M15.5 3.5v4"></path></>,
  folder: <path d="M3.5 7a1.5 1.5 0 0 1 1.5-1.5h4l2 2h8a1.5 1.5 0 0 1 1.5 1.5v8.5A1.5 1.5 0 0 1 19 19H5a1.5 1.5 0 0 1-1.5-1.5z"></path>,
  user2: <><circle cx="12" cy="9" r="3.5"></circle><path d="M12 21a9 9 0 1 1 0-18 9 9 0 0 1 0 18zM6.5 18.5c1.2-2.2 3-3.3 5.5-3.3s4.3 1.1 5.5 3.3"></path></>,
  map: <path d="M9 4.5L3.5 6.5v13L9 17.5l6 2 5.5-2v-13L15 6.5zM9 4.5v13M15 6.5v13"></path>,
  spark: <path d="M12 3.5l1.8 5.2 5.2 1.8-5.2 1.8L12 17.5l-1.8-5.2L5 10.5l5.2-1.8z"></path>,
  cmd: <path d="M9 6.5a2.5 2.5 0 1 0-2.5 2.5H9zM9 9v6M9 15H6.5A2.5 2.5 0 1 0 9 17.5zM15 9H9M15 9V6.5A2.5 2.5 0 1 1 17.5 9zM15 9v6M15 15h2.5a2.5 2.5 0 1 1-2.5 2.5zM9 15h6"></path>,
  more: <path d="M6 12h.01M12 12h.01M18 12h.01" strokeWidth="2.6"></path>,
  back: <path d="M15 6l-6 6 6 6"></path>,
};
function Ic({ n, s = 15, w = 1.7, style }) {
  return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={w} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={style}>{CPI[n]}</svg>;
}
function NovaMark({ s = 14, busy }) {
  return <span className={"dm" + (busy ? " spin" : "")} style={{ width: s, height: s }}></span>;
}
function Keys({ k }) {
  return <span className="kg">{k.map((x, i) => <span key={i} className="kbd">{x}</span>)}</span>;
}
Object.assign(window, { Ic, NovaMark, Keys });
