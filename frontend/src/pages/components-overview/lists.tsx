// material-ui
import { Grid, Stack, Typography } from '@mui/material';

// project import
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import BasicList from 'sections/components-overview/lists/BasicList';
import InteractiveList from 'sections/components-overview/lists/InteractiveList';
import NestedList from 'sections/components-overview/lists/NestedList';
import SelectedList from 'sections/components-overview/lists/SelectedList';
import AlignList from 'sections/components-overview/lists/AlignList';
import ScrollableList from 'sections/components-overview/lists/ScrollableList';
import FolderList from 'sections/components-overview/lists/FolderList';
import TransactionList from 'sections/components-overview/lists/TransactionList';
import NotificationList from 'sections/components-overview/lists/NotificationList';
import UserList from 'sections/components-overview/lists/UserList';

// ==============================|| COMPONENTS - LIST ||============================== //

const ComponentList = () => (
  <ComponentSkeleton>
    <ComponentHeader
      title="Lists"
      caption="Lists are continuous, vertical indexes of text or images."
      directory="src/pages/components-overview/lists"
      link="https://mui.com/material-ui/react-list/"
    />
    <ComponentWrapper>
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Stack spacing={3}>
            <Stack spacing={1}>
              <Typography variant="h5">Basic</Typography>
              <BasicList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Interactive</Typography>
              <InteractiveList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Scrollable</Typography>
              <ScrollableList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Notification</Typography>
              <NotificationList />
            </Stack>
          </Stack>
        </Grid>
        <Grid item xs={12} md={6}>
          <Stack spacing={3}>
            <Stack spacing={1}>
              <Typography variant="h5">Nested</Typography>
              <NestedList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Selected</Typography>
              <SelectedList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Align Item</Typography>
              <AlignList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Folder</Typography>
              <FolderList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Transaction History</Typography>
              <TransactionList />
            </Stack>
            <Stack spacing={1}>
              <Typography variant="h5">Users</Typography>
              <UserList />
            </Stack>
          </Stack>
        </Grid>
      </Grid>
    </ComponentWrapper>
  </ComponentSkeleton>
);

export default ComponentList;
