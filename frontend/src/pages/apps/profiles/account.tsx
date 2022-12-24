import { useState } from 'react';
import { useLocation, Link, Outlet } from 'react-router-dom';

// material-ui
import { Box, Tab, Tabs } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import { ContainerOutlined, FileTextOutlined, LockOutlined, SettingOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';

// ==============================|| PROFILE - ACCOUNT ||============================== //

const AccountProfile = () => {
  const { pathname } = useLocation();

  let selectedTab = 0;
  switch (pathname) {
    case '/apps/profiles/account/personal':
      selectedTab = 1;
      break;
    case '/apps/profiles/account/my-account':
      selectedTab = 2;
      break;
    case '/apps/profiles/account/password':
      selectedTab = 3;
      break;
    case '/apps/profiles/account/role':
      selectedTab = 4;
      break;
    case '/apps/profiles/account/settings':
      selectedTab = 5;
      break;
    case '/apps/profiles/account/basic':
    default:
      selectedTab = 0;
  }

  const [value, setValue] = useState(selectedTab);

  const handleChange = (event: React.SyntheticEvent, newValue: number) => {
    setValue(newValue);
  };

  return (
    <MainCard border={false} boxShadow>
      <Box sx={{ borderBottom: 1, borderColor: 'divider', width: '100%' }}>
        <Tabs value={value} onChange={handleChange} variant="scrollable" scrollButtons="auto" aria-label="account profile tab">
          <Tab label="Profile" component={Link} to="/apps/profiles/account/basic" icon={<UserOutlined />} iconPosition="start" />
          <Tab label="Personal" component={Link} to="/apps/profiles/account/personal" icon={<FileTextOutlined />} iconPosition="start" />
          <Tab
            label="My Account"
            component={Link}
            to="/apps/profiles/account/my-account"
            icon={<ContainerOutlined />}
            iconPosition="start"
          />
          <Tab label="Change Password" component={Link} to="/apps/profiles/account/password" icon={<LockOutlined />} iconPosition="start" />
          <Tab label="Role" component={Link} to="/apps/profiles/account/role" icon={<TeamOutlined />} iconPosition="start" />
          <Tab label="Settings" component={Link} to="/apps/profiles/account/settings" icon={<SettingOutlined />} iconPosition="start" />
        </Tabs>
      </Box>
      <Box sx={{ mt: 2.5 }}>
        <Outlet />
      </Box>
    </MainCard>
  );
};

export default AccountProfile;
