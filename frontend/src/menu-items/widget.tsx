// third-party
import { FormattedMessage } from 'react-intl';

// assets
import { LineChartOutlined, IdcardOutlined, DatabaseOutlined } from '@ant-design/icons';

// type
import { NavItemType } from 'types/menu';

// icons
const icons = {
  LineChartOutlined,
  IdcardOutlined,
  DatabaseOutlined
};

// ==============================|| MENU ITEMS - DASHBOARD ||============================== //

const widget: NavItemType = {
  id: 'group-widget',
  title: <FormattedMessage id="widgets" />,
  type: 'group',
  children: [
    {
      id: 'statistics',
      title: <FormattedMessage id="statistics" />,
      type: 'item',
      url: '/widget/statistics',
      icon: icons.IdcardOutlined
    },
    {
      id: 'data',
      title: <FormattedMessage id="data" />,
      type: 'item',
      url: '/widget/data',
      icon: icons.DatabaseOutlined
    },
    {
      id: 'chart',
      title: <FormattedMessage id="chart" />,
      type: 'item',
      url: '/widget/chart',
      icon: icons.LineChartOutlined
    }
  ]
};

export default widget;
