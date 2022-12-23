import { useEffect, useRef, useState } from 'react';

// material-ui
import { Button, Dialog, DialogActions, DialogContent, DialogProps, DialogTitle, Grid, Typography } from '@mui/material';

// project import
import IconButton from 'components/@extended/IconButton';

// assets
import { CloseOutlined } from '@ant-design/icons';

// ==============================|| DIALOG - SCROLLING ||============================== //

export default function ScrollDialog() {
  const [open, setOpen] = useState(false);
  const [scroll, setScroll] = useState<DialogProps['scroll']>('paper');

  const handleClickOpen = (scrollType: DialogProps['scroll']) => () => {
    setOpen(true);
    setScroll(scrollType);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const descriptionElementRef = useRef<HTMLElement>(null);
  useEffect(() => {
    if (open) {
      const { current: descriptionElement } = descriptionElementRef;
      if (descriptionElement !== null) {
        descriptionElement.focus();
      }
    }
  }, [open]);

  return (
    <>
      <Button variant="contained" onClick={handleClickOpen('paper')} sx={{ mr: 1, ml: 1, mb: 1, mt: 1 }}>
        scroll=paper
      </Button>
      <Button variant="outlined" onClick={handleClickOpen('body')} sx={{ mr: 1, ml: 1, mb: 1, mt: 1 }}>
        scroll=body
      </Button>
      <Dialog
        open={open}
        onClose={handleClose}
        scroll={scroll}
        aria-labelledby="scroll-dialog-title"
        aria-describedby="scroll-dialog-description"
      >
        <Grid container spacing={2} justifyContent="space-between" alignItems="center">
          <Grid item>
            <DialogTitle>Subscribe</DialogTitle>
          </Grid>
          <Grid item sx={{ mr: 1.5 }}>
            <IconButton color="secondary" onClick={handleClose}>
              <CloseOutlined />
            </IconButton>
          </Grid>
        </Grid>
        <DialogContent dividers>
          <Grid container spacing={1.25}>
            {[...new Array(25)].map((i, index) => (
              <Grid item key={`${index}-${scroll}`}>
                <Typography variant="h6">
                  Cras mattis consectetur purus sit amet fermentum. Cras justo odio, dapibus ac in, egestas eget quam. Morbi leo risus,
                  porta ac consectetur ac, vestibulum at eros. Praesent commodo cursus magna, vel scelerisque nisl consectetur et.
                </Typography>
              </Grid>
            ))}
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button color="error" onClick={handleClose}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleClose} sx={{ mr: 1 }}>
            Subscribe
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
