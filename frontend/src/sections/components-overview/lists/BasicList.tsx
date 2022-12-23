// material-ui
import { List, ListItem, ListItemButton, ListItemIcon, ListItemText } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import { LayoutOutlined, FilePptOutlined, RadiusUprightOutlined } from '@ant-design/icons';

// ==============================|| LIST - BASIC ||============================== //

const BasicList = () => {
  const basicListCodeString = `<List sx={{ p: 0 }}>
  <ListItem disablePadding divider>
    <ListItemButton>
      <ListItemText primary="List item 01" />
    </ListItemButton>
  </ListItem>
  <ListItem disablePadding divider>
    <ListItemButton>
      <ListItemText primary="List item 02" />
    </ListItemButton>
  </ListItem>
  <ListItem disablePadding divider>
    <ListItemButton>
      <ListItemIcon>
        <LayoutOutlined />
      </ListItemIcon>
      <ListItemText primary="Sample" />
    </ListItemButton>
  </ListItem>
  <ListItem disablePadding divider>
    <ListItemButton>
      <ListItemIcon>
        <FilePptOutlined />
      </ListItemIcon>
      <ListItemText primary="Page" />
    </ListItemButton>
  </ListItem>
  <ListItem disablePadding>
    <ListItemButton>
      <ListItemIcon>
        <RadiusUprightOutlined />
      </ListItemIcon>
      <ListItemText primary="UI Elements" />
    </ListItemButton>
  </ListItem>
</List>`;

  return (
    <MainCard content={false} codeHighlight codeString={basicListCodeString}>
      <List sx={{ p: 0 }}>
        <ListItem disablePadding divider>
          <ListItemButton>
            <ListItemText primary="List item 01" />
          </ListItemButton>
        </ListItem>
        <ListItem disablePadding divider>
          <ListItemButton>
            <ListItemText primary="List item 02" />
          </ListItemButton>
        </ListItem>
        <ListItem disablePadding divider>
          <ListItemButton>
            <ListItemIcon>
              <LayoutOutlined />
            </ListItemIcon>
            <ListItemText primary="Sample" />
          </ListItemButton>
        </ListItem>
        <ListItem disablePadding divider>
          <ListItemButton>
            <ListItemIcon>
              <FilePptOutlined />
            </ListItemIcon>
            <ListItemText primary="Page" />
          </ListItemButton>
        </ListItem>
        <ListItem disablePadding>
          <ListItemButton>
            <ListItemIcon>
              <RadiusUprightOutlined />
            </ListItemIcon>
            <ListItemText primary="UI Elements" />
          </ListItemButton>
        </ListItem>
      </List>
    </MainCard>
  );
};

export default BasicList;
