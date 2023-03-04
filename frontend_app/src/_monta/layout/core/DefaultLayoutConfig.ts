import {ILayout} from './LayoutModels'

export const DefaultLayoutConfig: ILayout = {
  main: {
    type: 'default',
    primaryColor: '#009EF7',
    darkSkinEnabled: true,
  },
  loader: {
    display: true,
    type: 'default', // Set default|spinner-message|spinner-logo to hide or show page loader
  },
  scrolltop: {
    display: true,
  },
  header: {
    width: 'fluid', // Set fixed|fluid to change width type
    fixed: {
      desktop: false,
      tabletAndMobile: true, // Set true|false to set fixed Header for tablet and mobile modes
    },
  },
  megaMenu: {
    display: false, // Set true|false to show or hide Mega Menu
  },
  aside: {
    minimized: false,
    minimize: true,
  },
  content: {
    width: 'fixed', // Set fixed|fluid to change width
  },
  footer: {
    width: 'fluid', // Set fixed|fluid to change width type
  },
}
