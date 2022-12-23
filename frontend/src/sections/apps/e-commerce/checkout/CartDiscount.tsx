import { useState } from 'react';

// material-ui
import { Button, FormHelperText, Stack, TextField, Typography } from '@mui/material';

// third-party
import { useFormik } from 'formik';
import * as yup from 'yup';

// project imports
import CouponCode from './CouponCode';
import { useDispatch, useSelector } from 'store';
import { setDiscount } from 'store/reducers/cart';
import { openSnackbar } from 'store/reducers/snackbar';

const validationSchema = yup.object({
  code: yup.string().oneOf(['MANTIS50', 'FLAT05', 'SUB150', 'UPTO200'], 'Coupon expired').required('Coupon code is required')
});

// ==============================|| CHECKOUT CART - CART DISCOUNT ||============================== //

const CartDiscount = () => {
  const dispatch = useDispatch();

  const [open, setOpen] = useState(false);
  const [coupon, setCoupon] = useState<string>('');
  const cart = useSelector((state) => state.cart);
  const handleClickOpen = () => {
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const formik = useFormik({
    enableReinitialize: true,
    initialValues: {
      code: coupon
    },
    validationSchema,
    onSubmit: (values) => {
      dispatch(setDiscount(values.code, cart.checkout.total));
      dispatch(
        openSnackbar({
          open: true,
          message: 'Coupon Add Success',
          variant: 'alert',
          alert: {
            color: 'success'
          },
          close: false
        })
      );
    }
  });

  return (
    <Stack justifyContent="flex-end" spacing={1}>
      <Typography align="left" variant="caption" color="textSecondary" sx={{ cursor: 'pointer' }} onClick={handleClickOpen}>
        Have a Promo Code?
      </Typography>
      <form onSubmit={formik.handleSubmit}>
        <Stack justifyContent="flex-end" spacing={1}>
          <Stack direction="row" justifyContent="space-between" spacing={2}>
            <TextField
              id="code"
              name="code"
              fullWidth
              placeholder="example"
              value={formik.values.code}
              onChange={formik.handleChange}
              error={Boolean(formik.errors.code)}
            />

            <Button type="submit" color="primary" variant="contained" aria-label="directions">
              Apply
            </Button>
          </Stack>
          {formik.errors.code && (
            <FormHelperText error id="standard-code">
              {formik.errors.code}
            </FormHelperText>
          )}
        </Stack>
      </form>

      <CouponCode open={open} handleClose={handleClose} setCoupon={setCoupon} />
    </Stack>
  );
};

export default CartDiscount;
