import Loadable from '../components/Loadable';
import { lazy } from 'react';
import MainLayout from '../layout/MainLayout';
import AuthGuard from 'utils/route-guard/AuthGuard';

const AccountProfile = Loadable(lazy(() => import('pages/apps/profiles/account')));
const AccountTabProfile = Loadable(lazy(() => import('sections/apps/profiles/account/TabProfile')));
const AccountTabPersonal = Loadable(lazy(() => import('sections/apps/profiles/account/TabPersonal')));
const AccountTabAccount = Loadable(lazy(() => import('sections/apps/profiles/account/TabAccount')));
const AccountTabPassword = Loadable(lazy(() => import('sections/apps/profiles/account/TabPassword')));
const AccountTabRole = Loadable(lazy(() => import('sections/apps/profiles/account/TabRole')));
const AccountTabSettings = Loadable(lazy(() => import('sections/apps/profiles/account/TabSettings')));

const AccountRoutes = {
  path: 'account',
  element: (
    <AuthGuard>
      <MainLayout />
    </AuthGuard>
  ),
  children: [
    {
      path: 'profile',
      element: <AccountProfile />,
      children: [
        {
          path: 'basic',
          element: <AccountTabProfile />
        },
        {
          path: 'personal',
          element: <AccountTabPersonal />
        },
        {
          path: 'my-account',
          element: <AccountTabAccount />
        },
        {
          path: 'password',
          element: <AccountTabPassword />
        },
        {
          path: 'role',
          element: <AccountTabRole />
        },
        {
          path: 'settings',
          element: <AccountTabSettings />
        }
      ]
    }
  ]
};

export default AccountRoutes;
