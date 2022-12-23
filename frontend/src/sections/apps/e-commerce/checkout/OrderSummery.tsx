// material-ui
import { Stack, Table, TableBody, TableCell, TableContainer, TableRow, Typography } from '@mui/material';

// third-party
import currency from 'currency.js';

// types
import { CartCheckoutStateProps } from 'types/cart';

// project imports
import MainCard from 'components/MainCard';

// ==============================|| CHECKOUT CART - ORDER SUMMARY ||============================== //

const OrderSummary = ({ checkout, show }: { checkout: CartCheckoutStateProps; show?: boolean }) => (
  <Stack spacing={3}>
    <MainCard content={false} sx={{ borderRadius: show ? '4px' : '0 0 4px 4px', borderTop: show ? '1px inherit' : 'none' }}>
      <TableContainer>
        <Table sx={{ minWidth: 'auto' }} size="small" aria-label="simple table">
          <TableBody>
            {show && (
              <TableRow>
                <TableCell>
                  <Typography variant="subtitle1">Order Summary</Typography>
                </TableCell>
                <TableCell />
              </TableRow>
            )}
            <TableRow>
              <TableCell sx={{ borderBottom: 'none', opacity: 0.5 }}>Sub Total</TableCell>
              <TableCell align="right" sx={{ borderBottom: 'none' }}>
                {checkout.subtotal && <Typography variant="subtitle1">{currency(checkout.subtotal).format()}</Typography>}
              </TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={{ borderBottom: 'none', opacity: 0.5 }}>Estimated Delivery</TableCell>
              <TableCell align="right" sx={{ borderBottom: 'none' }}>
                {checkout.shipping && (
                  <Typography variant="subtitle1">{checkout.shipping <= 0 ? '-' : currency(checkout.shipping).format()}</Typography>
                )}
              </TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={{ borderBottom: 'none', opacity: 0.5 }}>Voucher</TableCell>
              <TableCell align="right" sx={{ borderBottom: 'none' }}>
                {checkout.discount && (
                  <Typography variant="subtitle1" sx={{ color: 'success.main' }}>
                    {checkout.discount <= 0 ? '-' : currency(checkout.discount).format()}
                  </Typography>
                )}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>
    </MainCard>
    <MainCard>
      <Stack direction="row" justifyContent="space-between" alignItems="center">
        <Typography variant="subtitle1">Total</Typography>
        {checkout.total && (
          <Typography variant="subtitle1" align="right">
            {currency(checkout.total).format()}
          </Typography>
        )}
      </Stack>
    </MainCard>
  </Stack>
);

export default OrderSummary;
