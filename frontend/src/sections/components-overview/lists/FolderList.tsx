// material-ui
import { List, ListItem, ListItemText, ListItemAvatar } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import AntAvatar from 'components/@extended/Avatar';

// assets
import { AimOutlined, CameraOutlined, FileSearchOutlined } from '@ant-design/icons';

// ==============================|| LIST - FOLDER ||============================== //

export default function FolderList() {
  const folderListCodeString = `<List sx={{ width: '100%', bgcolor: 'background.paper' }}>
  <ListItem>
    <ListItemAvatar>
      <AntAvatar alt="Basic" type="combined" color="warning">
        <CameraOutlined />
      </AntAvatar>
    </ListItemAvatar>
    <ListItemText primary="Photos" secondary="Jan 9, 2014" />
  </ListItem>
  <ListItem>
    <ListItemAvatar>
      <AntAvatar alt="Basic" type="combined">
        <FileSearchOutlined />
      </AntAvatar>
    </ListItemAvatar>
    <ListItemText primary="Work" secondary="Jan 7, 2014" />
  </ListItem>
  <ListItem>
    <ListItemAvatar>
      <AntAvatar alt="Basic" type="combined" color="info">
        <AimOutlined />
      </AntAvatar>
    </ListItemAvatar>
    <ListItemText primary="Vacation" secondary="July 20, 2014" />
  </ListItem>
</List>`;

  return (
    <MainCard content={false} codeString={folderListCodeString}>
      <List sx={{ width: '100%', bgcolor: 'background.paper' }}>
        <ListItem>
          <ListItemAvatar>
            <AntAvatar alt="Basic" type="combined" color="warning">
              <CameraOutlined />
            </AntAvatar>
          </ListItemAvatar>
          <ListItemText primary="Photos" secondary="Jan 9, 2014" />
        </ListItem>
        <ListItem>
          <ListItemAvatar>
            <AntAvatar alt="Basic" type="combined">
              <FileSearchOutlined />
            </AntAvatar>
          </ListItemAvatar>
          <ListItemText primary="Work" secondary="Jan 7, 2014" />
        </ListItem>
        <ListItem>
          <ListItemAvatar>
            <AntAvatar alt="Basic" type="combined" color="info">
              <AimOutlined />
            </AntAvatar>
          </ListItemAvatar>
          <ListItemText primary="Vacation" secondary="July 20, 2014" />
        </ListItem>
      </List>
    </MainCard>
  );
}
