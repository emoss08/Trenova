import { useState } from 'react';

// material-ui
import {
  Box,
  Button,
  Divider,
  Drawer,
  Grid,
  Typography,
  Autocomplete,
  FormControl,
  FormControlLabel,
  MenuItem,
  Radio,
  RadioGroup,
  Select,
  Stack,
  TextField,
  InputLabel,
  FormHelperText,
  Tooltip
} from '@mui/material';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { DesktopDatePicker } from '@mui/x-date-pickers/DesktopDatePicker';

// third party
import * as yup from 'yup';
import { useFormik } from 'formik';

// project imports
import AddStoryComment from './AddStoryComment';
import StoryComment from './StoryComment';
import AlertStoryDelete from './AlertStoryDelete';
import AnimateButton from 'components/@extended/AnimateButton';
import UploadMultiFile from 'components/third-party/dropzone/MultiFile';
import { openSnackbar } from 'store/reducers/snackbar';
import { useDispatch, useSelector } from 'store';
import { editStory, deleteStory } from 'store/reducers/kanban';

// assets
import { CloseOutlined, DeleteFilled } from '@ant-design/icons';

// types
import { KanbanColumn, KanbanProfile, KanbanUserStory } from 'types/kanban';
import IconButton from 'components/@extended/IconButton';
import { DropzopType } from 'types/dropzone';

interface Props {
  story: KanbanUserStory;
  open: boolean;
  handleDrawerOpen: () => void;
}

const avatarImage = require.context('assets/images/users', true);
const validationSchema = yup.object({
  title: yup.string().required('User story title is required'),
  dueDate: yup.date()
});

// ==============================|| KANBAN BACKLOGS - EDIT STORY ||============================== //

const EditStory = ({ story, open, handleDrawerOpen }: Props) => {
  const dispatch = useDispatch();
  const kanban = useSelector((state) => state.kanban);
  const { profiles, columns, comments, userStory, userStoryOrder } = kanban;

  const formik = useFormik({
    enableReinitialize: true,
    initialValues: {
      id: story.id,
      title: story.title,
      assign: story.assign,
      columnId: story.columnId,
      priority: story.priority,
      dueDate: story.dueDate ? new Date(story.dueDate) : new Date(),
      acceptance: story.acceptance,
      description: story.description,
      commentIds: story.commentIds,
      image: false,
      itemIds: story.itemIds,
      files: []
    },
    validationSchema,
    onSubmit: (values) => {
      dispatch(editStory(values, userStory));
      dispatch(
        openSnackbar({
          open: true,
          message: 'Submit Success',
          variant: 'alert',
          alert: {
            color: 'success'
          },
          close: false
        })
      );
      handleDrawerOpen();
    }
  });

  const [openModal, setOpenModal] = useState(false);
  const handleModalClose = (status: boolean) => {
    setOpenModal(false);
    if (status) {
      handleDrawerOpen();
      dispatch(deleteStory(story.id, userStory, userStoryOrder));
      dispatch(
        openSnackbar({
          open: true,
          message: 'Story Deleted successfully',
          anchorOrigin: { vertical: 'top', horizontal: 'right' },
          variant: 'alert',
          alert: {
            color: 'success'
          },
          close: false
        })
      );
    }
  };

  return (
    <Drawer
      sx={{
        ml: open ? 3 : 0,
        flexShrink: 0,
        zIndex: 1200,
        overflowX: 'hidden',
        width: { xs: 320, md: 450 },
        '& .MuiDrawer-paper': {
          height: '100vh',
          width: { xs: 320, md: 450 },
          position: 'fixed',
          border: 'none',
          borderRadius: '0px'
        }
      }}
      variant="temporary"
      anchor="right"
      open={open}
      ModalProps={{ keepMounted: true }}
      onClose={() => {
        handleDrawerOpen();
        formik.resetForm();
      }}
    >
      {open && (
        <>
          <Box sx={{ p: 3 }}>
            <Grid container alignItems="center" spacing={0.5} justifyContent="space-between">
              <Grid item sx={{ width: 'calc(100% - 64px)' }}>
                <Stack direction="row" spacing={0.5} alignItems="center">
                  <Typography
                    variant="h4"
                    sx={{
                      display: 'inline-block',
                      width: 'calc(100% - 34px)',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      verticalAlign: 'middle'
                    }}
                  >
                    {story.title}
                  </Typography>
                </Stack>
              </Grid>

              <Grid item>
                <Stack direction="row" alignItems="center">
                  <Tooltip title="Delete Task">
                    <IconButton color="error" onClick={() => setOpenModal(true)} size="small" sx={{ fontSize: '0.875rem' }}>
                      <DeleteFilled />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Close">
                    <IconButton color="secondary" onClick={handleDrawerOpen} size="small" sx={{ fontSize: '0.875rem' }}>
                      <CloseOutlined />
                    </IconButton>
                  </Tooltip>
                </Stack>
                {openModal && <AlertStoryDelete title={story.title} open={openModal} handleClose={handleModalClose} />}
              </Grid>
            </Grid>
          </Box>
          <Divider />
          <Box sx={{ p: 3 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <form onSubmit={formik.handleSubmit}>
                  <LocalizationProvider dateAdapter={AdapterDateFns}>
                    <Grid container spacing={2.5}>
                      <Grid item xs={12}>
                        <Stack spacing={1}>
                          <InputLabel>Title</InputLabel>
                          <TextField
                            fullWidth
                            id="title"
                            name="title"
                            placeholder="Title"
                            value={formik.values.title}
                            onChange={formik.handleChange}
                            error={formik.touched.title && Boolean(formik.errors.title)}
                            helperText={formik.touched.title && formik.errors.title}
                          />
                        </Stack>
                      </Grid>
                      <Grid item xs={12}>
                        <Stack spacing={1}>
                          <InputLabel>Assign to</InputLabel>
                          <Autocomplete
                            id="assign"
                            value={profiles.find((profile: KanbanProfile) => profile.id === formik.values.assign) || null}
                            onChange={(event, value) => {
                              formik.setFieldValue('assign', value?.id);
                            }}
                            options={profiles}
                            fullWidth
                            autoHighlight
                            getOptionLabel={(option) => option.name}
                            isOptionEqualToValue={(option) => option.id === formik.values.assign}
                            renderOption={(props, option) => (
                              <Box component="li" sx={{ '& > img': { mr: 2, flexShrink: 0 } }} {...props}>
                                <img loading="lazy" width="20" src={avatarImage(`./${option.avatar}`)} alt="" />
                                {option.name}
                              </Box>
                            )}
                            renderInput={(params) => (
                              <TextField
                                {...params}
                                placeholder="Choose a assignee"
                                inputProps={{
                                  ...params.inputProps,
                                  autoComplete: 'new-password' // disable autocomplete and autofill
                                }}
                              />
                            )}
                          />
                        </Stack>
                      </Grid>
                      <Grid item xs={12}>
                        <Stack spacing={1}>
                          <InputLabel>Prioritize</InputLabel>
                          <FormControl>
                            <RadioGroup
                              row
                              aria-label="color"
                              value={formik.values.priority}
                              onChange={formik.handleChange}
                              name="priority"
                              id="priority"
                            >
                              <FormControlLabel
                                value="low"
                                control={<Radio color="primary" sx={{ color: 'primary.main' }} />}
                                label="Low"
                              />
                              <FormControlLabel
                                value="medium"
                                control={<Radio color="warning" sx={{ color: 'warning.main' }} />}
                                label="Medium"
                              />
                              <FormControlLabel value="high" control={<Radio color="error" sx={{ color: 'error.main' }} />} label="High" />
                            </RadioGroup>
                          </FormControl>
                        </Stack>
                      </Grid>
                      <Grid item xs={12}>
                        <Stack spacing={1}>
                          <InputLabel>Due date</InputLabel>
                          <DesktopDatePicker
                            value={formik.values.dueDate}
                            inputFormat="dd/MM/yyyy"
                            onChange={(date) => {
                              formik.setFieldValue('dueDate', date);
                            }}
                            renderInput={(props) => <TextField fullWidth {...props} placeholder="Due Date" />}
                          />
                        </Stack>
                      </Grid>
                      <Grid item xs={12}>
                        <Stack spacing={1}>
                          <InputLabel>Acceptance</InputLabel>
                          <TextField
                            fullWidth
                            id="acceptance"
                            name="acceptance"
                            multiline
                            rows={3}
                            value={formik.values.acceptance}
                            onChange={formik.handleChange}
                            error={formik.touched.acceptance && Boolean(formik.errors.acceptance)}
                            helperText={formik.touched.acceptance && formik.errors.acceptance}
                          />
                        </Stack>
                      </Grid>
                      <Grid item xs={12}>
                        <Stack spacing={1}>
                          <InputLabel>Description</InputLabel>
                          <TextField
                            fullWidth
                            id="description"
                            name="description"
                            multiline
                            rows={3}
                            value={formik.values.description}
                            onChange={formik.handleChange}
                            error={formik.touched.description && Boolean(formik.errors.description)}
                            helperText={formik.touched.description && formik.errors.description}
                          />
                        </Stack>
                      </Grid>
                      <Grid item xs={12}>
                        <Stack spacing={1}>
                          <InputLabel>State</InputLabel>
                          <FormControl fullWidth sx={{ m: 1 }}>
                            <Select
                              id="columnId"
                              name="columnId"
                              displayEmpty
                              value={formik.values.columnId}
                              onChange={formik.handleChange}
                              inputProps={{ 'aria-label': 'Without label' }}
                            >
                              {columns.map((column: KanbanColumn, index: number) => (
                                <MenuItem key={index} value={column.id}>
                                  {column.title}
                                </MenuItem>
                              ))}
                            </Select>
                          </FormControl>
                        </Stack>
                      </Grid>
                      <Grid item xs={12}>
                        <Grid container spacing={1}>
                          <Grid item xs={12}>
                            <InputLabel sx={{ mt: 0.5 }}>Attachments:</InputLabel>
                          </Grid>
                          <Grid item xs={12}>
                            <UploadMultiFile
                              type={DropzopType.standard}
                              showList={true}
                              setFieldValue={formik.setFieldValue}
                              files={formik.values.files}
                              error={formik.touched.files && !!formik.errors.files}
                            />
                          </Grid>
                          {formik.touched.files && formik.errors.files && (
                            <Grid item xs={12}>
                              <FormHelperText error id="standard-weight-helper-text-password-login">
                                {formik.errors.files}
                              </FormHelperText>
                            </Grid>
                          )}
                        </Grid>
                      </Grid>

                      <Grid item xs={12}>
                        <AnimateButton>
                          <Button fullWidth variant="contained" type="submit">
                            Save
                          </Button>
                        </AnimateButton>
                      </Grid>
                    </Grid>
                  </LocalizationProvider>
                </form>
              </Grid>
              <Grid item xs={12}>
                {story?.commentIds &&
                  [...story?.commentIds].reverse().map((commentId, index) => {
                    const commentData = comments.filter((comment) => comment.id === commentId)[0];
                    const profile = profiles.filter((item) => item.id === commentData.profileId)[0];
                    return <StoryComment key={index} comment={commentData} profile={profile} />;
                  })}
              </Grid>
              <Grid item xs={12}>
                <AddStoryComment storyId={story.id} />
              </Grid>
            </Grid>
          </Box>
        </>
      )}
    </Drawer>
  );
};
export default EditStory;
