import { useContext } from 'react';

// material-ui
import { List, ListItemButton, ListItemIcon, ListItemText } from '@mui/material';

// assets
import { LogoutOutlined, UserOutlined } from '@ant-design/icons';
import { Link, useLocation } from 'react-router-dom';
import { ConfigContext } from '../../../../../contexts/ConfigContext';
import LightModeOutlinedIcon from '@mui/icons-material/LightModeOutlined';
import Brightness4OutlinedIcon from '@mui/icons-material/Brightness4Outlined';

// ==============================|| HEADER PROFILE - PROFILE TAB ||============================== //

interface Props {
  handleLogout: () => void;
}

const ProfileTab = ({ handleLogout }: Props) => {
  const location = useLocation();
  const isSelected = (path: string) => location.pathname === path;

  const config = useContext(ConfigContext);

  const handleThemeSwitcher = () => {
    config.onChangeMode(config.mode === 'dark' ? 'light' : 'dark');
  };

  return (
    <List component="nav" sx={{ p: 0, '& .MuiListItemIcon-root': { minWidth: 32 } }}>
      <ListItemButton selected={isSelected('/account/profile/personal')} component={Link} to="/account/profile/personal">
        <ListItemIcon>
          <UserOutlined />
        </ListItemIcon>
        <ListItemText primary="View Profile" />
      </ListItemButton>

      <ListItemButton onClick={handleThemeSwitcher}>
        <ListItemIcon>{config.mode === 'dark' ? <LightModeOutlinedIcon /> : <Brightness4OutlinedIcon />}</ListItemIcon>
        <ListItemText primary={config.mode === 'dark' ? 'Switch To Light Mode' : 'Switch To Dark Mode'} />
      </ListItemButton>
      <ListItemButton onClick={handleLogout}>
        <ListItemIcon>
          <LogoutOutlined />
        </ListItemIcon>
        <ListItemText primary="Logout" />
      </ListItemButton>
    </List>
  );
};

export default ProfileTab;
