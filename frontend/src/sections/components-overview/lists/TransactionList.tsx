// material-ui
import { List, ListItemAvatar, ListItemButton, ListItemSecondaryAction, ListItemText, Stack, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import AntAvatar from 'components/@extended/Avatar';

// assets
import { GiftOutlined, MessageOutlined, SettingOutlined } from '@ant-design/icons';

// action style
const actionSX = {
  mt: 0.75,
  ml: 1,
  top: 'auto',
  right: 'auto',
  alignSelf: 'flex-start',
  transform: 'none'
};

// ==============================|| LIST - TRANSACTION ||============================== //

const TransactionList = () => {
  const transactionListCodeString = `<List
  component="nav"
  sx={{
    py: 0,
    '& .MuiListItemButton-root': {
      '& .MuiListItemSecondaryAction-root': { ...actionSX, position: 'relative' }
    }
  }}
>
  <ListItemButton divider>
    <ListItemAvatar>
      <AntAvatar alt="Basic" type="combined" color="success">
        <GiftOutlined />
      </AntAvatar>
    </ListItemAvatar>
    <ListItemText primary={<Typography variant="subtitle1">Payment from #002434</Typography>} secondary="Today, 2:00 AM" />
    <ListItemSecondaryAction>
      <Stack alignItems="flex-end">
        <Typography variant="subtitle1" noWrap>
          + $1,430
        </Typography>
        <Typography variant="h6" color="secondary" noWrap>
          78%
        </Typography>
      </Stack>
    </ListItemSecondaryAction>
  </ListItemButton>
  <ListItemButton divider>
    <ListItemAvatar>
      <AntAvatar alt="Basic" type="combined">
        <MessageOutlined />
      </AntAvatar>
    </ListItemAvatar>
    <ListItemText primary={<Typography variant="subtitle1">Payment from #984947</Typography>} secondary="5 August, 1:45 PM" />
    <ListItemSecondaryAction>
      <Stack alignItems="flex-end">
        <Typography variant="subtitle1" noWrap>
          + $302
        </Typography>
        <Typography variant="h6" color="secondary" noWrap>
          8%
        </Typography>
      </Stack>
    </ListItemSecondaryAction>
  </ListItemButton>
  <ListItemButton>
    <ListItemAvatar>
      <AntAvatar alt="Basic" type="combined" color="error">
        <SettingOutlined />
      </AntAvatar>
    </ListItemAvatar>
    <ListItemText primary={<Typography variant="subtitle1">Payment from #988784</Typography>} secondary="7 hours ago" />
    <ListItemSecondaryAction>
      <Stack alignItems="flex-end">
        <Typography variant="subtitle1" noWrap>
          + $682
        </Typography>
        <Typography variant="h6" color="secondary" noWrap>
          16%
        </Typography>
      </Stack>
    </ListItemSecondaryAction>
  </ListItemButton>
</List>`;

  return (
    <MainCard sx={{ mt: 2 }} content={false} codeString={transactionListCodeString}>
      <List
        component="nav"
        sx={{
          py: 0,
          '& .MuiListItemButton-root': {
            '& .MuiListItemSecondaryAction-root': { ...actionSX, position: 'relative' }
          }
        }}
      >
        <ListItemButton divider>
          <ListItemAvatar>
            <AntAvatar alt="Basic" type="combined" color="success">
              <GiftOutlined />
            </AntAvatar>
          </ListItemAvatar>
          <ListItemText primary={<Typography variant="subtitle1">Payment from #002434</Typography>} secondary="Today, 2:00 AM" />
          <ListItemSecondaryAction>
            <Stack alignItems="flex-end">
              <Typography variant="subtitle1" noWrap>
                + $1,430
              </Typography>
              <Typography variant="h6" color="secondary" noWrap>
                78%
              </Typography>
            </Stack>
          </ListItemSecondaryAction>
        </ListItemButton>
        <ListItemButton divider>
          <ListItemAvatar>
            <AntAvatar alt="Basic" type="combined">
              <MessageOutlined />
            </AntAvatar>
          </ListItemAvatar>
          <ListItemText primary={<Typography variant="subtitle1">Payment from #984947</Typography>} secondary="5 August, 1:45 PM" />
          <ListItemSecondaryAction>
            <Stack alignItems="flex-end">
              <Typography variant="subtitle1" noWrap>
                + $302
              </Typography>
              <Typography variant="h6" color="secondary" noWrap>
                8%
              </Typography>
            </Stack>
          </ListItemSecondaryAction>
        </ListItemButton>
        <ListItemButton>
          <ListItemAvatar>
            <AntAvatar alt="Basic" type="combined" color="error">
              <SettingOutlined />
            </AntAvatar>
          </ListItemAvatar>
          <ListItemText primary={<Typography variant="subtitle1">Payment from #988784</Typography>} secondary="7 hours ago" />
          <ListItemSecondaryAction>
            <Stack alignItems="flex-end">
              <Typography variant="subtitle1" noWrap>
                + $682
              </Typography>
              <Typography variant="h6" color="secondary" noWrap>
                16%
              </Typography>
            </Stack>
          </ListItemSecondaryAction>
        </ListItemButton>
      </List>
    </MainCard>
  );
};

export default TransactionList;
