// material-ui
import {
  Button,
  Chip,
  Divider,
  Grid,
  InputLabel,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography
} from '@mui/material';
import IconButton from 'components/@extended/IconButton';

// project import
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';

// assets
import { EllipsisOutlined } from '@ant-design/icons';

// table data
function createData(name: string, avatar: string, email: string, role: number, status: boolean) {
  return { name, avatar, email, role, status };
}

const avatarImage = require.context('assets/images/users', true);

const rows = [
  createData('Frozen Tek', 'avatar-1.png', 'owner@company.com', 1, true),
  createData('Eclair Dues', 'avatar-3.png', 'manager@company.com', 2, true),
  createData('Schem Lein', 'avatar-2.png', 'sl@company.com', 3, false),
  createData('Jhon Doe', 'avatar-4.png', 'jd@company.com', 3, true),
  createData('Tevoni Wug', 'avatar-5.png', 'tw@company.com', 0, false)
];

// ==============================|| ACCOUNT PROFILE - ROLE ||============================== //

const TabRole = () => (
  <Grid container spacing={3}>
    <Grid item xs={12}>
      <MainCard title="Invite Team Members" content={false}>
        <Stack spacing={2.5} sx={{ p: 2.5 }}>
          <Typography variant="h4">
            5/10{' '}
            <Typography variant="subtitle1" component="span">
              members available in your plan.
            </Typography>
          </Typography>
          <Divider />
          <Stack
            spacing={3}
            direction="row"
            justifyContent="space-between"
            alignItems="flex-end"
            sx={{ width: { xs: 1, md: '80%', lg: '60%' } }}
          >
            <Stack spacing={1} sx={{ width: `calc(100% - 110px)` }}>
              <InputLabel htmlFor="outlined-email">Email Address</InputLabel>
              <TextField fullWidth id="outlined-email" variant="outlined" placeholder="Enter your email address" />
            </Stack>
            <Button variant="contained" size="large">
              Send
            </Button>
          </Stack>
        </Stack>
        <TableContainer>
          <Table sx={{ minWidth: 350 }} aria-label="simple table">
            <TableHead>
              <TableRow>
                <TableCell sx={{ pl: 3 }}>Member</TableCell>
                <TableCell>Role</TableCell>
                <TableCell align="right">Status</TableCell>
                <TableCell align="right" />
              </TableRow>
            </TableHead>
            <TableBody>
              {rows.map((row) => (
                <TableRow hover key={row.name}>
                  <TableCell sx={{ pl: 3 }} component="th">
                    <Stack direction="row" alignItems="center" spacing={1.25}>
                      <Avatar alt="Avatar 1" src={avatarImage(`./${row.avatar}`)} />
                      <Stack spacing={0}>
                        <Typography variant="subtitle1">{row.name}</Typography>
                        <Typography variant="caption" color="secondary">
                          {row.email}
                        </Typography>
                      </Stack>
                    </Stack>
                  </TableCell>
                  <TableCell>
                    {row.role === 1 && <Chip size="small" color="primary" label="Owner" />}
                    {row.role === 2 && <Chip size="small" variant="light" color="info" label="Manager" />}
                    {row.role === 3 && <Chip size="small" variant="light" color="warning" label="Staff" />}
                    {row.role === 0 && <Chip size="small" variant="light" color="success" label="Customer" />}
                  </TableCell>
                  <TableCell align="right">
                    {!row.status && (
                      <Stack direction="row" alignItems="center" spacing={1.25} justifyContent="flex-end">
                        <Button size="small" color="error">
                          Resend
                        </Button>
                        <Chip size="small" color="info" variant="outlined" label="Invited" />
                      </Stack>
                    )}
                    {row.status && <Chip size="small" color="success" label="Joined" />}
                  </TableCell>
                  <TableCell align="right">
                    <IconButton size="small" color="secondary">
                      <EllipsisOutlined style={{ fontSize: '1.15rem' }} />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </MainCard>
    </Grid>
    <Grid item xs={12}>
      <Stack direction="row" justifyContent="flex-end" alignItems="center" spacing={2}>
        <Button color="error">Cancel</Button>
        <Button variant="contained">Update Profile</Button>
      </Stack>
    </Grid>
  </Grid>
);

export default TabRole;
