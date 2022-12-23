// material-ui
import { Button, Checkbox, FormControlLabel, Grid, InputLabel, Stack, TextField, Typography } from '@mui/material';

// third-party
import { useFormik } from 'formik';
import * as yup from 'yup';

// project imports
import AnimateButton from 'components/@extended/AnimateButton';

const validationSchema = yup.object({
  cardName: yup.string().required('Card Name is required'),
  cardNumber: yup.string().required('Card Number is required')
});

// ==============================|| VALIDATION WIZARD - PAYMENT ||============================== //

export type PaymentData = { cardName?: string; cardNumber?: number };
interface PaymentFormProps {
  paymentData: PaymentData;
  setPaymentData: (d: PaymentData) => void;
  handleNext: () => void;
  handleBack: () => void;
  setErrorIndex: (i: number | null) => void;
}

export default function PaymentForm({ paymentData, setPaymentData, handleNext, handleBack, setErrorIndex }: PaymentFormProps) {
  const formik = useFormik({
    initialValues: {
      cardName: paymentData.cardName,
      cardNumber: paymentData.cardNumber
    },
    validationSchema,
    onSubmit: (values) => {
      setPaymentData({
        cardName: values.cardName,
        cardNumber: values.cardNumber
      });
      handleNext();
    }
  });

  return (
    <>
      <Typography variant="h5" gutterBottom sx={{ mb: 2 }}>
        Payment method
      </Typography>
      <form onSubmit={formik.handleSubmit}>
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Stack spacing={0.5}>
              <InputLabel>Name on Card</InputLabel>
              <TextField
                id="cardName"
                name="cardName"
                value={formik.values.cardName}
                onChange={formik.handleChange}
                error={formik.touched.cardName && Boolean(formik.errors.cardName)}
                helperText={formik.touched.cardName && formik.errors.cardName}
                placeholder="Name on card"
                fullWidth
                autoComplete="cc-name"
              />
            </Stack>
          </Grid>
          <Grid item xs={12} md={6}>
            <Stack spacing={0.5}>
              <InputLabel>Card Number</InputLabel>
              <TextField
                id="cardNumber"
                name="cardNumber"
                placeholder="Card number"
                value={formik.values.cardNumber}
                onChange={formik.handleChange}
                error={formik.touched.cardNumber && Boolean(formik.errors.cardNumber)}
                helperText={formik.touched.cardNumber && formik.errors.cardNumber}
                fullWidth
                autoComplete="cc-number"
              />
            </Stack>
          </Grid>
          <Grid item xs={12} md={6}>
            <Stack spacing={0.5}>
              <InputLabel>Expiry Date</InputLabel>
              <TextField id="expDate" placeholder="Expiry date" fullWidth autoComplete="cc-exp" />
            </Stack>
          </Grid>
          <Grid item xs={12} md={6}>
            <Stack spacing={0.5}>
              <InputLabel>CVV Number</InputLabel>
              <TextField id="cvv" placeholder="CVV" helperText="Last three digits on signature strip" fullWidth autoComplete="cc-csc" />
            </Stack>
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox color="primary" name="saveCard" value="yes" />}
              label="Remember credit card details for next time"
            />
          </Grid>
          <Grid item xs={12}>
            <Stack direction="row" justifyContent="space-between">
              <Button onClick={handleBack} sx={{ my: 3, ml: 1 }}>
                Back
              </Button>
              <AnimateButton>
                <Button variant="contained" type="submit" sx={{ my: 3, ml: 1 }} onClick={() => setErrorIndex(1)}>
                  Next
                </Button>
              </AnimateButton>
            </Stack>
          </Grid>
        </Grid>
      </form>
    </>
  );
}
