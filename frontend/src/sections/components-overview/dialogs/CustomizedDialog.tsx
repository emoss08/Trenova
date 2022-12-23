import { useState } from 'react';

// material-ui
import { styled } from '@mui/material/styles';
import { Button, Dialog, DialogTitle, DialogContent, DialogActions, Typography } from '@mui/material';

// project import
import IconButton from 'components/@extended/IconButton';

// assets
import { CloseOutlined } from '@ant-design/icons';

// ==============================|| CUSTOMIZED - CONTENT ||============================== //

const BootstrapDialog = styled(Dialog)(({ theme }) => ({
  '& .MuiDialogContent-root': {
    padding: theme.spacing(3)
  },
  '& .MuiDialogActions-root': {
    padding: theme.spacing(1.25),
    paddingRight: theme.spacing(2)
  }
}));

// ==============================|| CUSTOMIZED - TITLE ||============================== //

export interface DialogTitleProps {
  id: string;
  children?: React.ReactNode;
  onClose: () => void;
}

const BootstrapDialogTitle = ({ children, onClose, ...other }: DialogTitleProps) => (
  <DialogTitle sx={{ m: 0, p: 2 }} {...other}>
    {children}
    {onClose ? (
      <IconButton
        aria-label="close"
        onClick={onClose}
        color="secondary"
        sx={{
          position: 'absolute',
          right: 10,
          top: 10
        }}
      >
        <CloseOutlined />
      </IconButton>
    ) : null}
  </DialogTitle>
);

// ==============================|| DIALOG - CUSTOMIZED ||============================== //

export default function CustomizedDialogs() {
  const [open, setOpen] = useState(false);

  const handleClickOpen = () => {
    setOpen(true);
  };
  const handleClose = () => {
    setOpen(false);
  };

  return (
    <>
      <Button variant="contained" onClick={handleClickOpen}>
        Open dialog
      </Button>
      <BootstrapDialog onClose={handleClose} aria-labelledby="customized-dialog-title" open={open}>
        <BootstrapDialogTitle id="customized-dialog-title" onClose={handleClose}>
          Modal Title
        </BootstrapDialogTitle>
        <DialogContent dividers sx={{ p: 3 }}>
          <Typography variant="h6" gutterBottom>
            Cras mattis consectetur purus sit amet fermentum. Cras justo odio, dapibus ac facilisis in, egestas eget quam. Morbi leo risus,
            porta ac consectetur ac, vestibulum at eros. Praesent commodo cursus magna, vel scelerisque nisl consectetur et. Vivamus
            sagittis lacus vel augue laoreet rutrum faucibus dolor auctor.
          </Typography>
          <Typography variant="h6" gutterBottom>
            Aenean lacinia bibendum nulla sed consectetur. Praesent commodo cursus magna, vel scelerisque nisl consectetur et. Donec sed
            odio dui. Donec ullamcorper nulla non metus auctor fringilla.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button variant="contained" onClick={handleClose}>
            Save changes
          </Button>
        </DialogActions>
      </BootstrapDialog>
    </>
  );
}
