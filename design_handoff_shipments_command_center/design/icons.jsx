/* Lightweight inline icons — sized by `size` prop, stroke = currentColor */
const Ic = ({ children, size = 14, ...rest }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" {...rest}>{children}</svg>
);

const IcSearch = (p) => <Ic {...p}><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></Ic>;
const IcFilter = (p) => <Ic {...p}><path d="M3 5h18M6 12h12M10 19h4"/></Ic>;
const IcSort   = (p) => <Ic {...p}><path d="M7 4v16M4 8l3-4 3 4M17 20V4M14 16l3 4 3-4"/></Ic>;
const IcPlus   = (p) => <Ic {...p}><path d="M12 5v14M5 12h14"/></Ic>;
const IcChevR  = (p) => <Ic {...p}><path d="m9 6 6 6-6 6"/></Ic>;
const IcChevD  = (p) => <Ic {...p}><path d="m6 9 6 6 6-6"/></Ic>;
const IcChevU  = (p) => <Ic {...p}><path d="m18 15-6-6-6 6"/></Ic>;
const IcDots   = (p) => <Ic {...p}><circle cx="6" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="18" cy="12" r="1"/></Ic>;
const IcBell   = (p) => <Ic {...p}><path d="M6 8a6 6 0 0 1 12 0v5l1.5 3h-15L6 13z"/><path d="M10 19a2 2 0 0 0 4 0"/></Ic>;
const IcStar   = (p) => <Ic {...p}><path d="m12 3 2.6 5.6L20 9.4l-4 4 1 5.6L12 16l-5 3 1-5.6-4-4 5.4-.8z"/></Ic>;
const IcGear   = (p) => <Ic {...p}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.8l-.1-.1A2 2 0 1 1 7 4.7l.1.1a1.7 1.7 0 0 0 1.8.3h.1a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1A2 2 0 1 1 19.7 7l-.1.1a1.7 1.7 0 0 0-.3 1.8v.1a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></Ic>;
const IcLayers = (p) => <Ic {...p}><path d="m12 3 9 5-9 5-9-5z"/><path d="m3 13 9 5 9-5"/><path d="m3 18 9 5 9-5"/></Ic>;
const IcMap    = (p) => <Ic {...p}><path d="m3 6 6-3 6 3 6-3v15l-6 3-6-3-6 3z"/><path d="M9 3v15M15 6v15"/></Ic>;
const IcTruck  = (p) => <Ic {...p}><path d="M2 8h11v8H2z"/><path d="M13 11h5l3 3v2h-8z"/><circle cx="6" cy="18" r="2"/><circle cx="17" cy="18" r="2"/></Ic>;
const IcRoute  = (p) => <Ic {...p}><circle cx="6" cy="19" r="2"/><circle cx="18" cy="5" r="2"/><path d="M8 19h6a4 4 0 0 0 0-8H10a4 4 0 0 1 0-8h6"/></Ic>;
const IcCircle = (p) => <Ic {...p}><circle cx="12" cy="12" r="9"/></Ic>;
const IcCheck  = (p) => <Ic {...p}><path d="m4 12 5 5L20 6"/></Ic>;
const IcAlert  = (p) => <Ic {...p}><path d="M12 3 1 21h22z"/><path d="M12 10v5M12 18v.5"/></Ic>;
const IcClock  = (p) => <Ic {...p}><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></Ic>;
const IcUser   = (p) => <Ic {...p}><circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/></Ic>;
const IcMessage= (p) => <Ic {...p}><path d="M4 5h16v11H8l-4 4z"/></Ic>;
const IcBolt   = (p) => <Ic {...p}><path d="M13 2 4 14h6l-1 8 9-12h-6z"/></Ic>;
const IcDollar = (p) => <Ic {...p}><path d="M12 2v20"/><path d="M17 6H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></Ic>;
const IcGrip   = (p) => <Ic {...p}><circle cx="9" cy="6" r="1"/><circle cx="15" cy="6" r="1"/><circle cx="9" cy="12" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="9" cy="18" r="1"/><circle cx="15" cy="18" r="1"/></Ic>;
const IcLayout = (p) => <Ic {...p}><rect x="3" y="3" width="18" height="18" rx="1.5"/><path d="M3 9h18M9 9v12"/></Ic>;
const IcEye    = (p) => <Ic {...p}><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/></Ic>;
const IcSun    = (p) => <Ic {...p}><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/></Ic>;
const IcMoon   = (p) => <Ic {...p}><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></Ic>;
const IcKb     = (p) => <Ic {...p}><rect x="2" y="6" width="20" height="12" rx="1.5"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M6 14h12"/></Ic>;
const IcDownload = (p) => <Ic {...p}><path d="M12 4v12M6 12l6 6 6-6M4 20h16"/></Ic>;
const IcRefresh = (p) => <Ic {...p}><path d="M3 12a9 9 0 0 1 15-6.7L21 8"/><path d="M21 4v4h-4"/><path d="M21 12a9 9 0 0 1-15 6.7L3 16"/><path d="M3 20v-4h4"/></Ic>;
const IcRadar  = (p) => <Ic {...p}><circle cx="12" cy="12" r="3"/><circle cx="12" cy="12" r="7" opacity=".4"/><path d="M12 5v7l5 3"/></Ic>;
const IcShield = (p) => <Ic {...p}><path d="M12 2 4 5v6c0 5 3.5 9 8 11 4.5-2 8-6 8-11V5z"/></Ic>;
const IcFlag   = (p) => <Ic {...p}><path d="M5 21V4M5 4h11l-2 4 2 4H5"/></Ic>;
const IcPin    = (p) => <Ic {...p}><path d="M12 22s7-7 7-12a7 7 0 1 0-14 0c0 5 7 12 7 12z"/><circle cx="12" cy="10" r="2.5"/></Ic>;
const IcThermo = (p) => <Ic {...p}><path d="M14 14V4a2 2 0 1 0-4 0v10a4 4 0 1 0 4 0z"/></Ic>;
const IcLayoutGrid = (p) => <Ic {...p}><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></Ic>;

Object.assign(window, {
  IcSearch, IcFilter, IcSort, IcPlus, IcChevR, IcChevD, IcChevU, IcDots,
  IcBell, IcStar, IcGear, IcLayers, IcMap, IcTruck, IcRoute, IcCircle, IcCheck,
  IcAlert, IcClock, IcUser, IcMessage, IcBolt, IcDollar, IcGrip, IcLayout, IcEye,
  IcSun, IcMoon, IcKb, IcDownload, IcRefresh, IcRadar, IcShield, IcFlag, IcPin,
  IcThermo, IcLayoutGrid,
});
