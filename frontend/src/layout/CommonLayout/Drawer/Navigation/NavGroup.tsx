// material-ui
import { List, Typography } from '@mui/material';

// project import
import NavItem from './NavItem';

// types
import { NavItemType } from 'types/menu';

// ==============================|| NAVIGATION - LIST GROUP ||============================== //

interface Props {
  item: NavItemType;
}

const NavGroup = ({ item }: Props) => {
  const navCollapse = item.children?.map((menu) => {
    switch (menu.type) {
      case 'item':
        return <NavItem key={menu.id} item={menu} level={1} />;
      default:
        return (
          <Typography key={menu.id} variant="h6" color="error" align="center">
            Fix - Group Collapse or Items
          </Typography>
        );
    }
  });

  return (
    <List
      subheader={
        item.title && (
          <Typography variant="subtitle1" color="text.primary" sx={{ pl: 3, mb: 1.5 }}>
            {item.title}
          </Typography>
        )
      }
      sx={{ mb: 1 }}
    >
      {navCollapse}
    </List>
  );
};

export default NavGroup;
