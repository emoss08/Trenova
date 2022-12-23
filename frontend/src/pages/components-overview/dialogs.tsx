// material-ui
import { Grid } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import SimpleDialog from 'sections/components-overview/dialogs/SimpleDialog';
import AlertDialog from 'sections/components-overview/dialogs/AlertDialog';
import FormDialog from 'sections/components-overview/dialogs/FormDialog';
import TransitionsDialog from 'sections/components-overview/dialogs/TransitionsDialog';
import CustomizedDialog from 'sections/components-overview/dialogs/CustomizedDialog';
import FullScreenDialog from 'sections/components-overview/dialogs/FullScreenDialog';
import SizesDialog from 'sections/components-overview/dialogs/SizesDialog';
import ResponsiveDialog from 'sections/components-overview/dialogs/ResponsiveDialog';
import DraggableDialog from 'sections/components-overview/dialogs/DraggableDialog';
import ScrollDialog from 'sections/components-overview/dialogs/ScrollDialog';
import ConfirmationDialog from 'sections/components-overview/dialogs/ConfirmationDialog';

// ==============================|| COMPONENTS - DIALOGS ||============================== //

const Dialogs = () => {
  const basicDialogCodeString = `// SimpleDialog.tsx
<Button variant="contained" onClick={handleClickOpen}>
  Open simple dialog
</Button>
<Dialog onClose={handleClose} open={open}>
  <Grid
    container
    spacing={2}
    justifyContent="space-between"
    alignItems="center"
    sx={{ borderBottom: '1px solid {theme.palette.divider}' }}
  >
    <Grid item>
      <DialogTitle>Set backup account</DialogTitle>
    </Grid>
    <Grid item sx={{ mr: 1.5 }}>
      <IconButton color="secondary" onClick={handleClose}>
        <CloseOutlined />
      </IconButton>
    </Grid>
  </Grid>

  <List sx={{ p: 2.5 }}>
    {emails.map((email, index) => (
      <ListItem button onClick={() => handleListItemClick(email)} key={email} selected={selectedValue === email} sx={{ p: 1.25 }}>
        <ListItemAvatar>
          <Avatar src={avatarImage('./avatar-{index + 1}.png')} />
        </ListItemAvatar>
        <ListItemText primary={email} />
      </ListItem>
    ))}
    <ListItem autoFocus button onClick={() => handleListItemClick('addAccount')} sx={{ p: 1.25 }}>
      <ListItemAvatar>
        <Avatar sx={{ bgcolor: 'primary.lighter', color: 'primary.main', width: 32, height: 32 }}>
          <PlusOutlined style={{ fontSize: '0.625rem' }} />
        </Avatar>
      </ListItemAvatar>
      <ListItemText primary="Add Account" />
    </ListItem>
  </List>
</Dialog>`;

  const alertcDialogCodeString = `// AlertDialog.tsx
<Button variant="contained" onClick={handleClickOpen}>
  Open alert dialog
</Button>
<Dialog open={open} onClose={handleClose} aria-labelledby="alert-dialog-title" aria-describedby="alert-dialog-description">
<Box sx={{ p: 1, py: 1.5 }}>
  <DialogTitle id="alert-dialog-title">Use Google&apos;s location service?</DialogTitle>
  <DialogContent>
    <DialogContentText id="alert-dialog-description">
      Let Google help apps determine location. This means sending anonymous location data to Google, even when no apps are running.
    </DialogContentText>
  </DialogContent>
  <DialogActions>
    <Button color="error" onClick={handleClose}>
      Disagree
    </Button>
    <Button variant="contained" onClick={handleClose} autoFocus>
      Agree
    </Button>
  </DialogActions>
</Box>
</Dialog>`;

  const formDialogCodeString = `// FormDialog.tsx
<Button variant="contained" onClick={handleClickOpen}>
  Open form dialog
</Button>
<Dialog open={open} onClose={handleClose}>
  <Box sx={{ p: 1, py: 1.5 }}>
    <DialogTitle>Subscribe</DialogTitle>
    <DialogContent>
      <DialogContentText sx={{ mb: 2 }}>
        To subscribe to this website, please enter your email address here. We will send updates occasionally.
      </DialogContentText>
      <TextField id="name" placeholder="Email Address" type="email" fullWidth variant="outlined" />
    </DialogContent>
    <DialogActions>
      <Button color="error" onClick={handleClose}>
        Cancel
      </Button>
      <Button variant="contained" onClick={handleClose}>
        Subscribe
      </Button>
    </DialogActions>
  </Box>
</Dialog>`;

  const transitionsDialogCodeString = ` // TransitionsDialog.tsx
<Button variant="contained" onClick={handleClickOpen}>
  Slide in dialog
</Button>
<Dialog
  open={open}
  TransitionComponent={Transition}
  keepMounted
  onClose={handleClose}
  aria-describedby="alert-dialog-slide-description"
>
  <Box sx={{ p: 1, py: 1.5 }}>
    <DialogTitle>Use Google&apos;ss location service?</DialogTitle>
    <DialogContent>
      <DialogContentText id="alert-dialog-slide-description">
        Let Google help apps determine location. This means sending anonymous location data to Google, even when no apps are running.
      </DialogContentText>
    </DialogContent>
    <DialogActions>
      <Button color="error" onClick={handleClose}>
        Disagree
      </Button>
      <Button variant="contained" onClick={handleClose}>
        Agree
      </Button>
    </DialogActions>
  </Box>
</Dialog>`;

  const customizedDialogCodeString = `// CustomizedDialog.tsx
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
    <Button variant="contained" autoFocus onClick={handleClose}>
      Save changes
    </Button>
  </DialogActions>
</BootstrapDialog>`;

  const fullscreenDialogCodeString = `// FullScreenDialog.tsx
<Button variant="contained" onClick={handleClickOpen}>
  Open full-screen dialog
</Button>
<Dialog fullScreen open={open} onClose={handleClose} TransitionComponent={Transition}>
  <AppBar sx={{ position: 'relative', boxShadow: 'none' }}>
    <Toolbar>
      <IconButton edge="start" color="inherit" onClick={handleClose} aria-label="close">
        <CloseOutlined />
      </IconButton>
      <Typography sx={{ ml: 2, flex: 1 }} variant="h6" component="div">
        Set Backup Account
      </Typography>
      <Button autoFocus color="inherit" onClick={handleClose}>
        save
      </Button>
    </Toolbar>
  </AppBar>
  <List sx={{ p: 3 }}>
    <ListItem button>
      <ListItemAvatar>
        <Avatar src={avatarImage('./avatar-1.png')} />
      </ListItemAvatar>
      <ListItemText primary="Phone ringtone" secondary="Default" />
    </ListItem>
    <Divider />
    <ListItem button>
      <ListItemAvatar>
        <Avatar src={avatarImage('./avatar-2.png')} />
      </ListItemAvatar>
      <ListItemText primary="Default notification ringtone" secondary="Tethys" />
    </ListItem>
  </List>
</Dialog>`;

  const sizesDialogCodeString = `// SizesDialog.tsx
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
</Dialog>`;

  const responsiveDialogCodeString = `// ResponsiveDialog.tsx
<Button variant="contained" onClick={handleClickOpen}>
  Open responsive dialog
</Button>
<Dialog fullScreen={fullScreen} open={open} onClose={handleClose} aria-labelledby="responsive-dialog-title">
  <Box sx={{ p: 1, py: 1.5 }}>
    <DialogTitle id="responsive-dialog-title">Use Google&apos;s location service?</DialogTitle>
    <DialogContent>
      <DialogContentText>
        Let Google help apps determine location. This means sending anonymous location data to Google, even when no apps are running.
      </DialogContentText>
    </DialogContent>
    <DialogActions>
      <Button color="error" onClick={handleClose}>
        Disagree
      </Button>
      <Button variant="contained" onClick={handleClose} autoFocus>
        Agree
      </Button>
    </DialogActions>
  </Box>
</Dialog>`;

  const draggaleDialogCodeString = `// DraggableDialog
<Button variant="contained" onClick={handleClickOpen}>
  Open draggable dialog
</Button>
<Dialog open={open} onClose={handleClose} PaperComponent={PaperComponent} aria-labelledby="draggable-dialog-title">
  <Box sx={{ p: 1, py: 1.5 }}>
    <DialogTitle style={{ cursor: 'move' }} id="draggable-dialog-title">
      Subscribe
    </DialogTitle>
    <DialogContent>
      <DialogContentText sx={{ mb: 2 }}>
        To subscribe to this website, please enter your email address here. We will send updates occasionally.
      </DialogContentText>
      <TextField id="name" placeholder="Email Address" type="email" fullWidth variant="outlined" />
    </DialogContent>
    <DialogActions>
      <Button color="error" onClick={handleClose}>
        Cancel
      </Button>
      <Button variant="contained" onClick={handleClose}>
        Subscribe
      </Button>
    </DialogActions>
  </Box>
</Dialog>`;

  const scrollingDialogCodeString = `// ScrollDialog.tsx
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
        <Grid item key={'{index}-{scroll}'}>
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
</Dialog>`;

  const confirmDialogCodeString = `// ConfirmationDialog.tsx
<Box sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}>
  <List component="div" role="group">
    <ListItem button divider disabled>
      <ListItemText primary="Interruptions" />
    </ListItem>
    <ListItem
      button
      divider
      aria-haspopup="true"
      aria-controls="ringtone-menu"
      aria-label="phone ringtone"
      onClick={handleClickListItem}
    >
      <ListItemText primary="Phone Ringtone" secondary={value} />
    </ListItem>
    <ListItem button divider disabled>
      <ListItemText primary="Default Notification Ringtone" secondary="Tethys" />
    </ListItem>
    <ConfirmationDialogRaw id="ringtone-menu" keepMounted open={open} onClose={handleClose} value={value} />
  </List>
</Box>
<Dialog
  sx={{ '& .MuiDialog-paper': { width: '80%', maxHeight: 435 } }}
  maxWidth={matchDownMD ? 'sm' : 'lg'}
  TransitionProps={{ onEntering: handleEntering }}
  open={open}
  {...other}
>
  <DialogTitle>Phone Ringtone</DialogTitle>
  <DialogContent dividers>
    <RadioGroup row={!matchDownMD} ref={radioGroupRef} aria-label="ringtone" name="ringtone" value={value} onChange={handleChange}>
      {options.map((option) => (
        <FormControlLabel value={option} key={option} control={<Radio />} label={option} />
      ))}
    </RadioGroup>
  </DialogContent>
  <DialogActions>
    <Button color="error" autoFocus onClick={handleCancel}>
      Cancel
    </Button>
    <Button variant="contained" onClick={handleOk} sx={{ mr: 0.5 }}>
      Done
    </Button>
  </DialogActions>
</Dialog>`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Dialog"
        caption="Dialogs inform users about a task and can contain critical information, require decisions, or involve multiple tasks."
        directory="src/pages/components-overview/dialogs"
        link="https://mui.com/material-ui/react-dialog/"
      />
      <ComponentWrapper sx={{ '& .MuiCardContent-root': { textAlign: 'center' } }}>
        <Grid container spacing={3}>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Basic" codeString={basicDialogCodeString}>
              <SimpleDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Alert" codeString={alertcDialogCodeString}>
              <AlertDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Form" codeString={formDialogCodeString}>
              <FormDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Transitions" codeString={transitionsDialogCodeString}>
              <TransitionsDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Customized" codeString={customizedDialogCodeString}>
              <CustomizedDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Full Screen" codeString={fullscreenDialogCodeString}>
              <FullScreenDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Sizes" codeString={sizesDialogCodeString}>
              <SizesDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Responsive" codeString={responsiveDialogCodeString}>
              <ResponsiveDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Draggable" codeString={draggaleDialogCodeString}>
              <DraggableDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Scrolling" codeString={scrollingDialogCodeString}>
              <ScrollDialog />
            </MainCard>
          </Grid>
          <Grid item xs={12} sm={6} lg={4}>
            <MainCard title="Confirmation" codeString={confirmDialogCodeString}>
              <ConfirmationDialog />
            </MainCard>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default Dialogs;
