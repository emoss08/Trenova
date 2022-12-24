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
      id: 'profile',
      title: <FormattedMessage id="profile" />,
      type: 'collapse',
      icon: icons.UserOutlined,
      children: [
        {
          id: 'user-profile',
          title: <FormattedMessage id="user-profile" />,
          type: 'item',
          url: '/account/profile/personal'
        },
        {
          id: 'account-profile',
          title: <FormattedMessage id="account-profile" />,
          type: 'item',
          url: '/account/profile/basic'
        },
        {
          id: 'user-list',
          title: <FormattedMessage id="user-list" />,
          type: 'item',
          url: '/apps/profiles/user-list'
        },
        {
          id: 'user-card',
          title: <FormattedMessage id="user-card" />,
          type: 'item',
          url: '/apps/profiles/user-card'
        }
      ]
    }
  ]
};

export default pages;
