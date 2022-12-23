import { Link as RouterLink } from 'react-router-dom';

// material-ui
import { Avatar, Grid, Link, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import Avatar1 from 'assets/images/users/avatar-1.png';
import Avatar2 from 'assets/images/users/avatar-2.png';
import Avatar3 from 'assets/images/users/avatar-3.png';
import Avatar4 from 'assets/images/users/avatar-4.png';

// table data
function createData(time: string, subTime: string, avatar: string, name: string, title: string, subtext: string) {
  return { time, subTime, avatar, name, title, subtext };
}

const rows = [
  createData(
    '12',
    'hours',
    Avatar1,
    'John Deo',
    '[#1183] Workaround for OS X selects printing bug',
    'Chrome fixed the bug several versions ago, thus rendering this...'
  ),
  createData(
    '16',
    'hours',
    Avatar2,
    'Jems Win',
    '[#1249] Vertically center carousel controls',
    'Try any carousel control and reduce the screen width below...'
  ),
  createData(
    '40',
    'hours',
    Avatar3,
    'Jeny Wiliiam',
    '[#1254] Inaccurate small pagination height',
    'The height of pagination elements is not consistent with...'
  ),
  createData(
    '12',
    'hours',
    Avatar4,
    'Jems Win',
    '[#1249] Vertically center carousel controls',
    'Try any carousel control and reduce the screen width below...'
  )
];

// ==========================|| DATA WIDGET - ACTIVE TICKETS ||========================== //

const ActiveTickets = () => (
  <MainCard
    title="Active Tickets"
    content={false}
    secondary={
      <Link component={RouterLink} to="#" color="primary">
        View all
      </Link>
    }
  >
    <TableContainer>
      <Table sx={{ minWidth: 560 }}>
        <TableHead>
          <TableRow>
            <TableCell align="center">Due</TableCell>
            <TableCell>Name</TableCell>
            <TableCell sx={{ pr: 3 }}>Position</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((row, index) => (
            <TableRow hover key={index}>
              <TableCell align="center">
                <Typography variant="subtitle1">{row.time}</Typography>
                <Typography variant="subtitle2" color="secondary">
                  {row.subTime}
                </Typography>
              </TableCell>
              <TableCell>
                <Grid container spacing={2} alignItems="center" sx={{ flexWrap: 'nowrap' }}>
                  <Grid item>
                    <Avatar alt="User 1" src={row.avatar} />
                  </Grid>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="subtitle1">
                      {row.name}
                    </Typography>
                  </Grid>
                </Grid>
              </TableCell>
              <TableCell sx={{ pr: 3 }}>
                <Typography align="left" variant="subtitle1">
                  {row.title}
                </Typography>
                <Typography align="left" variant="caption" color="secondary">
                  {row.subtext}
                </Typography>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  </MainCard>
);

export default ActiveTickets;
