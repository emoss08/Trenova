import { useState } from 'react';

// material-ui
import {
  Box,
  Button,
  Dialog,
  DialogProps,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  Grid,
  MenuItem,
  Select,
  SelectChangeEvent,
  Switch,
  Typography
} from '@mui/material';

// ==============================|| DIALOG - SIZES ||============================== //

export default function MaxWidthDialog() {
  const [fullWidth, setFullWidth] = useState(true);
  const [maxWidth, setMaxWidth] = useState<DialogProps['maxWidth']>('sm');

  const [open, setOpen] = useState(false);
  const handleClickOpen = () => {
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const handleMaxWidthChange = (event: SelectChangeEvent<typeof maxWidth>) => {
    setMaxWidth(
      // @ts-expect-error autofill of arbitrary value is not handled.
      event.target.value
    );
  };

  const handleFullWidthChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setFullWidth(event.target.checked);
  };

  return (
    <>
      <Button variant="contained" onClick={handleClickOpen}>
        Open max-width dialog
      </Button>
      <Dialog fullWidth={fullWidth} maxWidth={maxWidth} open={open} onClose={handleClose}>
        <Box sx={{ p: 1, py: 1.5 }}>
          <DialogTitle>Optional sizes</DialogTitle>
          <DialogContent>
            <DialogContentText>You can set my maximum width and whether to adapt or not.</DialogContentText>
            <Grid container spacing={1.5} alignItems="center" sx={{ mt: 1 }}>
              <Grid item>
                <Typography variant="h6">Max Width :</Typography>
              </Grid>
              <Grid item>
                <FormControl sx={{ minWidth: 120 }}>
                  <Select
                    autoFocus
                    value={maxWidth}
                    onChange={handleMaxWidthChange}
                    inputProps={{
                      name: 'max-width',
                      id: 'max-width'
                    }}
                  >
                    <MenuItem value={false as any}>false</MenuItem>
                    <MenuItem value="xs">xs</MenuItem>
                    <MenuItem value="sm">sm</MenuItem>
                    <MenuItem value="md">md</MenuItem>
                    <MenuItem value="lg">lg</MenuItem>
                    <MenuItem value="xl">xl</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
            </Grid>
            <Grid container spacing={1.5} alignItems="center" sx={{ mt: 0.25 }}>
              <Grid item>
                <Typography variant="h6">Full Width:</Typography>
              </Grid>
              <Grid item>
                <Switch checked={fullWidth} onChange={handleFullWidthChange} />
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions>
            <Button variant="outlined" color="error" onClick={handleClose}>
              Close
            </Button>
          </DialogActions>
        </Box>
      </Dialog>
    </>
  );
}
