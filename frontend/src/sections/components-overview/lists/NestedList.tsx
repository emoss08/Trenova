import { useState } from 'react';

// material-ui
import { Collapse, List, ListItem, ListItemButton, ListItemIcon, ListItemText } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import { DownOutlined, LayoutOutlined, RadiusUprightOutlined, SettingOutlined, UpOutlined } from '@ant-design/icons';

// ==============================|| LIST - NESTED ||============================== //

const NestedList = () => {
  const [open, setOpen] = useState('sample');
  const [openChild, setOpenChild] = useState('');

  const handleClick = (page: string) => {
    setOpen(open !== page ? page : '');
    setOpenChild('');
  };

  const handleChildClick = (page: string) => {
    setOpenChild(openChild !== page ? page : '');
  };

  const nestedListCodeString = `<List sx={{ p: 0 }}>
  <ListItem disablePadding divider>
    <ListItemButton onClick={() => handleClick('sample')}>
      <ListItemIcon>
        <LayoutOutlined />
      </ListItemIcon>
      <ListItemText primary="Sample" />
      {open === 'sample' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
    </ListItemButton>
  </ListItem>
  <Collapse in={open === 'sample'} timeout="auto" unmountOnExit>
    <List component="div" disablePadding sx={{ bgcolor: 'secondary.100' }}>
      <ListItemButton sx={{ pl: 5 }}>
        <ListItemText primary="List item 01" />
      </ListItemButton>
      <ListItemButton sx={{ pl: 5 }} onClick={() => handleChildClick('list1')}>
        <ListItemText primary="List item 02" />
        {openChild === 'list1' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
      </ListItemButton>
      <Collapse in={openChild === 'list1'} timeout="auto" unmountOnExit>
        <List component="div" disablePadding sx={{ bgcolor: 'secondary.lighter' }}>
          <ListItemButton sx={{ pl: 7 }}>
            <ListItemText primary="List item 05" />
          </ListItemButton>
          <ListItemButton sx={{ pl: 7 }}>
            <ListItemText primary="List item 06" />
          </ListItemButton>
        </List>
      </Collapse>
    </List>
  </Collapse>
  <ListItem disablePadding divider>
    <ListItemButton onClick={() => handleClick('settings')}>
      <ListItemIcon>
        <SettingOutlined />
      </ListItemIcon>
      <ListItemText primary="Settings" />
      {open === 'settings' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
    </ListItemButton>
  </ListItem>
  <Collapse in={open === 'settings'} timeout="auto" unmountOnExit>
    <List component="div" disablePadding sx={{ bgcolor: 'secondary.100' }}>
      <ListItemButton sx={{ pl: 5 }}>
        <ListItemText primary="List item 03" />
      </ListItemButton>
      <ListItemButton sx={{ pl: 5 }}>
        <ListItemText primary="List item 04" />
      </ListItemButton>
    </List>
  </Collapse>
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
    <MainCard content={false} codeString={nestedListCodeString}>
      <List sx={{ p: 0 }}>
        <ListItem disablePadding divider>
          <ListItemButton onClick={() => handleClick('sample')}>
            <ListItemIcon>
              <LayoutOutlined />
            </ListItemIcon>
            <ListItemText primary="Sample" />
            {open === 'sample' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
          </ListItemButton>
        </ListItem>
        <Collapse in={open === 'sample'} timeout="auto" unmountOnExit>
          <List component="div" disablePadding sx={{ bgcolor: 'secondary.100' }}>
            <ListItemButton sx={{ pl: 5 }}>
              <ListItemText primary="List item 01" />
            </ListItemButton>
            <ListItemButton sx={{ pl: 5 }} onClick={() => handleChildClick('list1')}>
              <ListItemText primary="List item 02" />
              {openChild === 'list1' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
            </ListItemButton>
            <Collapse in={openChild === 'list1'} timeout="auto" unmountOnExit>
              <List component="div" disablePadding sx={{ bgcolor: 'secondary.lighter' }}>
                <ListItemButton sx={{ pl: 7 }}>
                  <ListItemText primary="List item 05" />
                </ListItemButton>
                <ListItemButton sx={{ pl: 7 }}>
                  <ListItemText primary="List item 06" />
                </ListItemButton>
              </List>
            </Collapse>
          </List>
        </Collapse>
        <ListItem disablePadding divider>
          <ListItemButton onClick={() => handleClick('settings')}>
            <ListItemIcon>
              <SettingOutlined />
            </ListItemIcon>
            <ListItemText primary="Settings" />
            {open === 'settings' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
          </ListItemButton>
        </ListItem>
        <Collapse in={open === 'settings'} timeout="auto" unmountOnExit>
          <List component="div" disablePadding sx={{ bgcolor: 'secondary.100' }}>
            <ListItemButton sx={{ pl: 5 }}>
              <ListItemText primary="List item 03" />
            </ListItemButton>
            <ListItemButton sx={{ pl: 5 }}>
              <ListItemText primary="List item 04" />
            </ListItemButton>
          </List>
        </Collapse>
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

export default NestedList;
