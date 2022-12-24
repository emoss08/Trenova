import { NavItemType } from '../types/menu';
import { FormattedMessage } from 'react-intl';

import { UserOutlined } from '@ant-design/icons';

const icons = {
  UserOutlined
};

const pages: NavItemType = {
  id: 'group-pages',
  type: 'group',
  children: [
    {
      id: 'user',
      title: <FormattedMessage id="user" />,
      type: 'collapse',
      icon: icons.UserOutlined,
      children: [
        {
          id: 'profile',
          title: <FormattedMessage id="user-profile" />,
          type: 'item',
          url: '/user/profile'
        }
      ]
    }
  ]
};

export default pages;
