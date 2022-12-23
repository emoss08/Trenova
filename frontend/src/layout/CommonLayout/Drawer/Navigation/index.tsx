// material-ui
import { Box, Typography } from '@mui/material';

// project import
import NavGroup from './NavGroup';
import menuItem from 'menu-items/components';
import { NavItemType } from 'types/menu';

// ==============================|| DRAWER CONTENT - NAVIGATION ||============================== //

const Navigation = ({ searchValue }: { searchValue?: string }) => {
  let filteredMenuItems: NavItemType[] = [];

  // if no value searched, we will render all menu items
  if (searchValue === null || searchValue === undefined || searchValue === '') {
    filteredMenuItems = menuItem;
  } else {
    menuItem.forEach((parentMenu) => {
      const matchedChildren: any[] = [];

      parentMenu.children?.forEach((child) => {
        if (child.search?.trim().toLowerCase().includes(searchValue!)) {
          // todo: consider other filed then id
          // if match
          matchedChildren.push(child); // get the filter menuitem and push it to children
        }
      });

      const parent = filteredMenuItems.filter((xx) => xx === parentMenu); // get the parent menu item/header
      if (parent.length === 0 && matchedChildren.length > 0) {
        const clonedParent = { ...parentMenu }; // clone children as we dont want entire children but just filtered
        clonedParent.children = matchedChildren;
        filteredMenuItems.push(clonedParent);
      }
    });
  }

  const navGroups = filteredMenuItems.map((item) => {
    switch (item.type) {
      case 'group':
        return <NavGroup key={item.id} item={item} />;
      default:
        return (
          <Typography key={item.id} variant="h6" color="error" align="center">
            Fix - Navigation Group
          </Typography>
        );
    }
  });

  return <Box sx={{ pt: 1 }}>{navGroups}</Box>;
};

export default Navigation;
