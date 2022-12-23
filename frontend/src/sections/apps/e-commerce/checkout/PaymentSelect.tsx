// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, FormControlLabel, Radio, Stack, Typography } from '@mui/material';

// project import
import { PaymentOptionsProps } from 'types/e-commerce';
import MainCard from 'components/MainCard';

// ==============================|| CHECKOUT PAYMENT - OPTIONS ||============================== //

const PaymentSelect = ({ item }: { item: PaymentOptionsProps }) => {
  const theme = useTheme();

  return (
    <MainCard content={false} sx={{ '&:hover': { boxShadow: theme.customShadows.primary } }}>
      <Box sx={{ p: 2, py: 1, pr: 0 }}>
        <FormControlLabel
          value={item.value}
          control={<Radio />}
          label={
            <Stack spacing={0.5} sx={{ width: '100%' }}>
              <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={2.5} sx={{ width: '100%' }}>
                <Typography variant="subtitle1">{item.title}</Typography>
                <Box
                  sx={{
                    backgroundImage: `url(${item.image})`,
                    backgroundSize: 'contain',
                    backgroundPosition: 'right',
                    borderColor: 'error.light',
                    ...item.size
                  }}
                />
              </Stack>
              <Typography variant="caption" color="textSecondary">
                {item.caption}
              </Typography>
            </Stack>
          }
          sx={{
            width: '100%',
            alignItems: 'flex-start',
            '& .MuiSvgIcon-root': { fontSize: 32 },
            '& .MuiFormControlLabel-label': { width: '100%' },
            '& .MuiRadio-root': { p: 0, pl: 1, pr: 1, pt: 0.5 }
          }}
        />
      </Box>
    </MainCard>
  );
};
export default PaymentSelect;
