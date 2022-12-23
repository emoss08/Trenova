// third-party
import { FormattedMessage } from 'react-intl';

// project import
import { NavItemType } from 'types/menu';

// ==============================|| MENU ITEMS - COMPONENTS ||============================== //

const menuItems: NavItemType[] = [
  {
    id: 'group-inputs',
    title: <FormattedMessage id="inputs" />,
    type: 'group',
    children: [
      {
        id: 'autocomplete',
        search: 'autocomplete, combo box, country select, grouped, multi select',
        title: <FormattedMessage id="autocomplete" />,
        type: 'item',
        url: '/components-overview/autocomplete'
      },
      {
        id: 'buttons',
        search: 'buttons, button group, icon button, toggle button, loading button',
        title: <FormattedMessage id="button" />,
        type: 'item',
        url: '/components-overview/buttons'
      },
      {
        id: 'checkbox',
        search: 'checkbox, indeterminate',
        title: <FormattedMessage id="checkbox" />,
        type: 'item',
        url: '/components-overview/checkbox'
      },
      {
        id: 'radio',
        search: 'radio',
        title: <FormattedMessage id="radio" />,
        type: 'item',
        url: '/components-overview/radio'
      },
      {
        id: 'rating',
        search: 'rating, star rating, feedback',
        title: <FormattedMessage id="rating" />,
        type: 'item',
        url: '/components-overview/rating'
      },
      {
        id: 'switch',
        search: 'switch',
        title: <FormattedMessage id="switch" />,
        type: 'item',
        url: '/components-overview/switch'
      },
      {
        id: 'select',
        search: 'select, multi-select',
        title: <FormattedMessage id="select" />,
        type: 'item',
        url: '/components-overview/select'
      },
      {
        id: 'slider',
        search: 'slider, range',
        title: <FormattedMessage id="slider" />,
        type: 'item',
        url: '/components-overview/slider'
      },
      {
        id: 'textfield',
        search: 'textfield, input, form input, search',
        title: <FormattedMessage id="text-field" />,
        type: 'item',
        url: '/components-overview/textfield'
      }
    ]
  },
  {
    id: 'data-display',
    title: <FormattedMessage id="data-display" />,
    type: 'group',
    children: [
      {
        id: 'avatars',
        search: 'avatars, fallbacks, group avatar',
        title: <FormattedMessage id="avatar" />,
        type: 'item',
        url: '/components-overview/avatars'
      },
      {
        id: 'badges',
        search: 'badges',
        title: <FormattedMessage id="badges" />,
        type: 'item',
        url: '/components-overview/badges'
      },
      {
        id: 'chips',
        search: 'chips, tags, ',
        title: <FormattedMessage id="chip" />,
        type: 'item',
        url: '/components-overview/chips'
      },
      {
        id: 'lists',
        search: 'lists, folder list, nested list',
        title: <FormattedMessage id="list" />,
        type: 'item',
        url: '/components-overview/lists'
      },
      {
        id: 'tooltip',
        search: 'tooltip',
        title: <FormattedMessage id="tooltip" />,
        type: 'item',
        url: '/components-overview/tooltip'
      },
      {
        id: 'typography',
        search: 'typography, h1, h2,h3, h4, h5, h6, caption, subtitle, body',
        title: <FormattedMessage id="typography" />,
        type: 'item',
        url: '/components-overview/typography'
      }
    ]
  },
  {
    id: 'feedback',
    title: <FormattedMessage id="feedback" />,
    type: 'group',
    children: [
      {
        id: 'alert',
        search: 'alert',
        title: <FormattedMessage id="alert" />,
        type: 'item',
        url: '/components-overview/alert'
      },
      {
        id: 'dialogs',
        search: 'dialogs, modal, sweetalert, confirmation box',
        title: <FormattedMessage id="dialogs" />,
        type: 'item',
        url: '/components-overview/dialogs'
      },
      {
        id: 'progress',
        search: 'progress, circular, linear, buffer',
        title: <FormattedMessage id="progress" />,
        type: 'item',
        url: '/components-overview/progress'
      },
      {
        id: 'snackbar',
        search: 'snackbar, notification, notify',
        title: <FormattedMessage id="snackbar" />,
        type: 'item',
        url: '/components-overview/snackbar'
      }
    ]
  },
  {
    id: 'navigation',
    title: <FormattedMessage id="navigation" />,
    type: 'group',
    children: [
      {
        id: 'breadcrumbs',
        search: 'breadcrumbs',
        title: <FormattedMessage id="breadcrumb" />,
        type: 'item',
        url: '/components-overview/breadcrumbs'
      },
      {
        id: 'pagination',
        search: 'pagination, table pagination',
        title: <FormattedMessage id="pagination" />,
        type: 'item',
        url: '/components-overview/pagination'
      },
      {
        id: 'speeddial',
        search: 'speeddial, speed dial, quick access button, fab button',
        title: <FormattedMessage id="speed-dial" />,
        type: 'item',
        url: '/components-overview/speeddial'
      },
      {
        id: 'stepper',
        search: 'stepper, form wizard, vertical stepper, vertical wizard',
        title: <FormattedMessage id="stepper" />,
        type: 'item',
        url: '/components-overview/stepper'
      },
      {
        id: 'tabs',
        search: 'tabs, vertical tab',
        title: <FormattedMessage id="tabs" />,
        type: 'item',
        url: '/components-overview/tabs'
      }
    ]
  },
  {
    id: 'surfaces',
    title: <FormattedMessage id="surfaces" />,
    type: 'group',
    children: [
      {
        id: 'accordion',
        search: 'accordion',
        title: <FormattedMessage id="accordion" />,
        type: 'item',
        url: '/components-overview/accordion'
      },
      {
        id: 'cards',
        search: 'cards',
        title: <FormattedMessage id="cards" />,
        type: 'item',
        url: '/components-overview/cards'
      }
    ]
  },
  {
    id: 'utils',
    title: <FormattedMessage id="utils" />,
    type: 'group',
    children: [
      {
        id: 'color',
        search: 'color',
        title: <FormattedMessage id="color" />,
        type: 'item',
        url: '/components-overview/color'
      },
      {
        id: 'date-time-picker',
        search: 'datetime, date, time date time, picker, date range picker',
        title: <FormattedMessage id="datetime" />,
        type: 'item',
        url: '/components-overview/date-time-picker'
      },
      {
        id: 'modal',
        search: 'modal, dialog',
        title: <FormattedMessage id="modal" />,
        type: 'item',
        url: '/components-overview/modal'
      },
      {
        id: 'shadows',
        search: 'shadows, color shadow',
        title: <FormattedMessage id="shadow" />,
        type: 'item',
        url: '/components-overview/shadows'
      },
      {
        id: 'timeline',
        search: 'timeline, list of event',
        title: <FormattedMessage id="timeline" />,
        type: 'item',
        url: '/components-overview/timeline'
      },
      {
        id: 'treeview',
        search: 'treeview, email clone',
        title: <FormattedMessage id="treeview" />,
        type: 'item',
        url: '/components-overview/treeview'
      }
    ]
  }
];

export default menuItems;
