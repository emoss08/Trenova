import { lazy } from 'react';

// project import
import CommonLayout from 'layout/CommonLayout';
import Loadable from 'components/Loadable';

// render - inputs components page
const Autocomplete = Loadable(lazy(() => import('pages/components-overview/autocomplete')));
const Buttons = Loadable(lazy(() => import('pages/components-overview/buttons')));
const Checkbox = Loadable(lazy(() => import('pages/components-overview/checkbox')));
const Radio = Loadable(lazy(() => import('pages/components-overview/radio')));
const Rating = Loadable(lazy(() => import('pages/components-overview/rating')));
const Select = Loadable(lazy(() => import('pages/components-overview/select')));
const Slider = Loadable(lazy(() => import('pages/components-overview/slider')));
const Switch = Loadable(lazy(() => import('pages/components-overview/switch')));
const TextField = Loadable(lazy(() => import('pages/components-overview/textfield')));

// render - feedback components page
const Alert = Loadable(lazy(() => import('pages/components-overview/alert')));
const Dialogs = Loadable(lazy(() => import('pages/components-overview/dialogs')));
const Progress = Loadable(lazy(() => import('pages/components-overview/progress')));
const Snackbar = Loadable(lazy(() => import('pages/components-overview/snackbar')));

// render - data display components
const Avatars = Loadable(lazy(() => import('pages/components-overview/avatars')));
const Badges = Loadable(lazy(() => import('pages/components-overview/badges')));
const Chips = Loadable(lazy(() => import('pages/components-overview/chips')));
const Lists = Loadable(lazy(() => import('pages/components-overview/lists')));
const Tooltip = Loadable(lazy(() => import('pages/components-overview/tooltip')));
const Typography = Loadable(lazy(() => import('pages/components-overview/typography')));

// render - navigation components page
const Breadcrumbs = Loadable(lazy(() => import('pages/components-overview/breadcrumbs')));
const Pagination = Loadable(lazy(() => import('pages/components-overview/pagination')));
const Speeddial = Loadable(lazy(() => import('pages/components-overview/speeddial')));
const Stepper = Loadable(lazy(() => import('pages/components-overview/stepper')));
const Tabs = Loadable(lazy(() => import('pages/components-overview/tabs')));

// render - surfaces components page
const Accordion = Loadable(lazy(() => import('pages/components-overview/accordion')));
const Cards = Loadable(lazy(() => import('pages/components-overview/cards')));

// render - utils components page
const Color = Loadable(lazy(() => import('pages/components-overview/color')));
const DateTimePicker = Loadable(lazy(() => import('pages/components-overview/date-time-picker')));
const Modal = Loadable(lazy(() => import('pages/components-overview/modal')));
const Shadow = Loadable(lazy(() => import('pages/components-overview/shadows')));
const Timeline = Loadable(lazy(() => import('pages/components-overview/timeline')));
const TreeView = Loadable(lazy(() => import('pages/components-overview/treeview')));

// ==============================|| COMPONENTS ROUTING ||============================== //

const ComponentsRoutes = {
  path: 'components-overview',
  element: <CommonLayout layout="component" />,
  children: [
    {
      path: 'autocomplete',
      element: <Autocomplete />
    },
    {
      path: 'buttons',
      element: <Buttons />
    },
    {
      path: 'checkbox',
      element: <Checkbox />
    },
    {
      path: 'radio',
      element: <Radio />
    },
    {
      path: 'rating',
      element: <Rating />
    },
    {
      path: 'switch',
      element: <Switch />
    },
    {
      path: 'select',
      element: <Select />
    },
    {
      path: 'slider',
      element: <Slider />
    },
    {
      path: 'textfield',
      element: <TextField />
    },
    {
      path: 'avatars',
      element: <Avatars />
    },
    {
      path: 'badges',
      element: <Badges />
    },
    {
      path: 'chips',
      element: <Chips />
    },
    {
      path: 'lists',
      element: <Lists />
    },
    {
      path: 'tooltip',
      element: <Tooltip />
    },
    {
      path: 'typography',
      element: <Typography />
    },
    {
      path: 'alert',
      element: <Alert />
    },
    {
      path: 'dialogs',
      element: <Dialogs />
    },
    {
      path: 'progress',
      element: <Progress />
    },
    {
      path: 'snackbar',
      element: <Snackbar />
    },
    {
      path: 'breadcrumbs',
      element: <Breadcrumbs />
    },
    {
      path: 'pagination',
      element: <Pagination />
    },
    {
      path: 'speeddial',
      element: <Speeddial />
    },
    {
      path: 'stepper',
      element: <Stepper />
    },
    {
      path: 'tabs',
      element: <Tabs />
    },
    {
      path: 'accordion',
      element: <Accordion />
    },
    {
      path: 'cards',
      element: <Cards />
    },
    {
      path: 'color',
      element: <Color />
    },
    {
      path: 'date-time-picker',
      element: <DateTimePicker />
    },
    {
      path: 'modal',
      element: <Modal />
    },
    {
      path: 'shadows',
      element: <Shadow />
    },
    {
      path: 'timeline',
      element: <Timeline />
    },
    {
      path: 'treeview',
      element: <TreeView />
    }
  ]
};

export default ComponentsRoutes;
