import { Link as RouterLink } from 'react-router-dom';

// material-ui
import { Divider, Link, List, ListItemButton, ListItemIcon, ListItemText } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';
import SimpleBar from 'components/third-party/SimpleBar';
import Dot from 'components/@extended/Dot';

// =========================|| DATA WIDGET - INCOMING REQUESTS ||========================= //

const IncomingRequests = () => (
  <MainCard
    title="Incoming Requests"
    content={false}
    secondary={
      <Link component={RouterLink} to="#" color="primary">
        View all
      </Link>
    }
  >
    <SimpleBar sx={{ height: 334 }}>
      <List component="nav" aria-label="main mailbox folders">
        <ListItemButton>
          <ListItemIcon>
            <Dot color="success" size={12} />
          </ListItemIcon>
          <ListItemText primary="Incoming requests" />
        </ListItemButton>
        <Divider />
        <ListItemButton>
          <ListItemIcon>
            <Dot color="error" size={12} />
          </ListItemIcon>
          <ListItemText primary="You have 2 pending requests.." />
        </ListItemButton>
        <Divider />
        <ListItemButton>
          <ListItemIcon>
            <Dot color="warning" size={12} />
          </ListItemIcon>
          <ListItemText primary="You have 3 pending tasks" />
        </ListItemButton>
        <Divider />
        <ListItemButton>
          <ListItemIcon>
            <Dot size={12} />
          </ListItemIcon>
          <ListItemText primary="New order received" />
        </ListItemButton>
        <Divider />
        <ListItemButton>
          <ListItemIcon>
            <Dot color="success" size={12} />
          </ListItemIcon>
          <ListItemText primary="Incoming requests" />
        </ListItemButton>
        <Divider />
        <ListItemButton>
          <ListItemIcon>
            <Dot size={12} />
          </ListItemIcon>
          <ListItemText primary="You have 2 pending requests.." />
        </ListItemButton>
        <Divider />
        <ListItemButton>
          <ListItemIcon>
            <Dot color="warning" size={12} />
          </ListItemIcon>
          <ListItemText primary="You have 3 pending tasks" />
        </ListItemButton>
        <Divider />
        <ListItemButton>
          <ListItemIcon>
            <Dot color="error" size={12} />
          </ListItemIcon>
          <ListItemText primary="New order received" />
        </ListItemButton>
      </List>
    </SimpleBar>
  </MainCard>
);

export default IncomingRequests;
