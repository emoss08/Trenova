import { useState } from 'react';

// material-ui
import { Collapse, List, ListItem, ListItemButton, ListItemIcon, ListItemText } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import { DownOutlined, LayoutOutlined, RadiusUprightOutlined, SettingOutlined, UpOutlined } from '@ant-design/icons';

const selectList = {
  '&.Mui-selected': {
    bgcolor: 'transparent',
    color: 'primary.main',
    '& .MuiListItemIcon-root': { color: 'primary.main' }
  }
};

// ==============================|| LIST - SELECTED ||============================== //

const SelectedList = () => {
  const [selected, setSelected] = useState('sample');
  const [open, setOpen] = useState('sample');
  const [openChild, setOpenChild] = useState('');

  const handleSelect = (page: string) => {
    setSelected(page);
  };

  const handleClick = (page: string) => {
    setOpen(open !== page ? page : '');
    setSelected(selected !== page ? selected : '');
    setOpenChild('');
  };

  const handleChildClick = (page: string) => {
    setOpenChild(openChild !== page ? page : '');
    setSelected(selected !== page ? selected : '');
  };

  const selectedListCodeString = `<List sx={{ p: 0 }}>
  <ListItem disablePadding divider>
    <ListItemButton selected={open === 'sample'} onClick={() => handleClick('sample')} sx={selectList}>
      <ListItemIcon>
        <LayoutOutlined />
      </ListItemIcon>
      <ListItemText primary="Sample" />
      {open === 'sample' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
    </ListItemButton>
  </ListItem>
  <Collapse in={open === 'sample'} timeout="auto" unmountOnExit>
    <List component="div" disablePadding>
      <ListItemButton
        sx={{ pl: 5, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
        selected={selected === 'list1'}
        onClick={() => handleSelect('list1')}
      >
        <ListItemText primary="List item 01" />
      </ListItemButton>
      <ListItemButton
        sx={{ pl: 5, ...selectList }}
        onClick={() => handleChildClick('list2')}
        selected={selected === 'list2' || openChild === 'list2'}
      >
        <ListItemText primary="List item 02" />
        {openChild === 'list1' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
      </ListItemButton>
      <Collapse in={openChild === 'list2'} timeout="auto" unmountOnExit>
        <List component="div" disablePadding>
          <ListItemButton
            sx={{ pl: 7, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
            selected={selected === 'list5'}
            onClick={() => handleSelect('list5')}
          >
            <ListItemText primary="List item 05" />
          </ListItemButton>
          <ListItemButton
            sx={{ pl: 7, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
            selected={selected === 'list6'}
            onClick={() => handleSelect('list6')}
          >
            <ListItemText primary="List item 06" />
          </ListItemButton>
        </List>
      </Collapse>
    </List>
  </Collapse>
  <ListItem disablePadding divider>
    <ListItemButton onClick={() => handleClick('settings')} selected={selected === 'settings' || open === 'settings'} sx={selectList}>
      <ListItemIcon>
        <SettingOutlined />
      </ListItemIcon>
      <ListItemText primary="Settings" />
      {open === 'settings' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
    </ListItemButton>
  </ListItem>
  <Collapse in={open === 'settings'} timeout="auto" unmountOnExit>
    <List component="div" disablePadding>
      <ListItemButton
        sx={{ pl: 5, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
        selected={selected === 'list3'}
        onClick={() => handleSelect('list3')}
      >
        <ListItemText primary="List item 03" />
      </ListItemButton>
      <ListItemButton
        sx={{ pl: 5, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
        selected={selected === 'list4'}
        onClick={() => handleSelect('list4')}
      >
        <ListItemText primary="List item 04" />
      </ListItemButton>
    </List>
  </Collapse>
  <ListItem disablePadding>
    <ListItemButton
      sx={{ '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
      selected={selected === 'ui-elements'}
      onClick={() => handleSelect('ui-elements')}
    >
      <ListItemIcon>
        <RadiusUprightOutlined />
      </ListItemIcon>
      <ListItemText primary="UI Elements" />
    </ListItemButton>
  </ListItem>
</List>`;

  return (
    <MainCard content={false} codeString={selectedListCodeString}>
      <List sx={{ p: 0 }}>
        <ListItem disablePadding divider>
          <ListItemButton selected={open === 'sample'} onClick={() => handleClick('sample')} sx={selectList}>
            <ListItemIcon>
              <LayoutOutlined />
            </ListItemIcon>
            <ListItemText primary="Sample" />
            {open === 'sample' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
          </ListItemButton>
        </ListItem>
        <Collapse in={open === 'sample'} timeout="auto" unmountOnExit>
          <List component="div" disablePadding>
            <ListItemButton
              sx={{ pl: 5, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
              selected={selected === 'list1'}
              onClick={() => handleSelect('list1')}
            >
              <ListItemText primary="List item 01" />
            </ListItemButton>
            <ListItemButton
              sx={{ pl: 5, ...selectList }}
              onClick={() => handleChildClick('list2')}
              selected={selected === 'list2' || openChild === 'list2'}
            >
              <ListItemText primary="List item 02" />
              {openChild === 'list1' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
            </ListItemButton>
            <Collapse in={openChild === 'list2'} timeout="auto" unmountOnExit>
              <List component="div" disablePadding>
                <ListItemButton
                  sx={{ pl: 7, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
                  selected={selected === 'list5'}
                  onClick={() => handleSelect('list5')}
                >
                  <ListItemText primary="List item 05" />
                </ListItemButton>
                <ListItemButton
                  sx={{ pl: 7, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
                  selected={selected === 'list6'}
                  onClick={() => handleSelect('list6')}
                >
                  <ListItemText primary="List item 06" />
                </ListItemButton>
              </List>
            </Collapse>
          </List>
        </Collapse>
        <ListItem disablePadding divider>
          <ListItemButton onClick={() => handleClick('settings')} selected={selected === 'settings' || open === 'settings'} sx={selectList}>
            <ListItemIcon>
              <SettingOutlined />
            </ListItemIcon>
            <ListItemText primary="Settings" />
            {open === 'settings' ? <DownOutlined style={{ fontSize: '0.75rem' }} /> : <UpOutlined style={{ fontSize: '0.75rem' }} />}
          </ListItemButton>
        </ListItem>
        <Collapse in={open === 'settings'} timeout="auto" unmountOnExit>
          <List component="div" disablePadding>
            <ListItemButton
              sx={{ pl: 5, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
              selected={selected === 'list3'}
              onClick={() => handleSelect('list3')}
            >
              <ListItemText primary="List item 03" />
            </ListItemButton>
            <ListItemButton
              sx={{ pl: 5, '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
              selected={selected === 'list4'}
              onClick={() => handleSelect('list4')}
            >
              <ListItemText primary="List item 04" />
            </ListItemButton>
          </List>
        </Collapse>
        <ListItem disablePadding>
          <ListItemButton
            sx={{ '&.Mui-selected': { borderRight: '2px solid', borderColor: 'primary.main' } }}
            selected={selected === 'ui-elements'}
            onClick={() => handleSelect('ui-elements')}
          >
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

export default SelectedList;
