import { Link as RouterLink } from 'react-router-dom';

// material-ui
import { Link, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';

// table data
function createData(name: string, designation: string, product: string, date?: string, badgeText?: string, badgeType?: string) {
  return { name, designation, product, date, badgeText, badgeType };
}

const rows = [
  createData('Materially', 'Powerful Admin Theme', '16,300', '$53', '$15,652'),
  createData('Photoshop', 'Design Software', '26,421', '$35', '$8,785'),
  createData('Guruable', 'Best Admin Template', '8,265', '$98', '$9,652'),
  createData('Flatable', 'Admin App', '10,652', '$20', '$7,856')
];
// =========================|| DATA WIDGET - APPLICATION SALES ||========================= //

const ApplicationSales = () => (
  <MainCard
    title="Application Sales"
    content={false}
    secondary={
      <Link component={RouterLink} to="#" color="primary">
        View all
      </Link>
    }
  >
    <TableContainer>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell sx={{ pl: 3 }}>Application</TableCell>
            <TableCell align="right">Sales</TableCell>
            <TableCell align="right">Avg. Price</TableCell>
            <TableCell align="right" sx={{ pr: 3 }}>
              Total
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((row, index) => (
            <TableRow hover key={index}>
              <TableCell sx={{ pl: 3 }}>
                <Typography align="left" variant="subtitle1">
                  {row.name}
                </Typography>
                <Typography align="left" variant="caption" color="secondary">
                  {row.designation}
                </Typography>
              </TableCell>
              <TableCell align="right">{row.product}</TableCell>
              <TableCell align="right">{row.date}</TableCell>
              <TableCell align="right" sx={{ pr: 3 }}>
                <span>{row.badgeText}</span>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  </MainCard>
);

export default ApplicationSales;
