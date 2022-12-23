import { useState } from 'react';

// material-ui
import { Button, FormHelperText, Grid, Stack, Typography } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';
import UploadAvatar from 'components/third-party/dropzone/Avatar';
import UploadSingleFile from 'components/third-party/dropzone/SingleFile';
import UploadMultiFile from 'components/third-party/dropzone/MultiFile';

// third-party
import { Formik } from 'formik';
import * as yup from 'yup';
import IconButton from 'components/@extended/IconButton';

// assets
import { UnorderedListOutlined, AppstoreOutlined } from '@ant-design/icons';

// ==============================|| PLUGINS - DROPZONE ||============================== //

const DropzonePage = () => {
  const [list, setList] = useState(false);

  return (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <MainCard
          title="Upload Multiple File"
          secondary={
            <Stack direction="row" alignItems="center" spacing={1.25}>
              <IconButton color={list ? 'secondary' : 'primary'} size="small" onClick={() => setList(false)}>
                <UnorderedListOutlined style={{ fontSize: '1.15rem' }} />
              </IconButton>
              <IconButton color={list ? 'primary' : 'secondary'} size="small" onClick={() => setList(true)}>
                <AppstoreOutlined style={{ fontSize: '1.15rem' }} />
              </IconButton>
            </Stack>
          }
        >
          <Formik
            initialValues={{ files: null }}
            onSubmit={(values: any) => {
              // submit form
            }}
            validationSchema={yup.object().shape({
              files: yup.mixed().required('Avatar is a required.')
            })}
          >
            {({ values, handleSubmit, setFieldValue, touched, errors }) => (
              <form onSubmit={handleSubmit}>
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <Stack spacing={1.5} alignItems="center">
                      <UploadMultiFile
                        showList={list}
                        setFieldValue={setFieldValue}
                        files={values.files}
                        error={touched.files && !!errors.files}
                      />
                      {touched.files && errors.files && (
                        <FormHelperText error id="standard-weight-helper-text-password-login">
                          {errors.files}
                        </FormHelperText>
                      )}
                    </Stack>
                  </Grid>
                </Grid>
              </form>
            )}
          </Formik>
        </MainCard>
      </Grid>
      <Grid item xs={12}>
        <MainCard title="Upload Single File">
          <Formik
            initialValues={{ files: null }}
            onSubmit={(values: any) => {
              // submit form
            }}
            validationSchema={yup.object().shape({
              files: yup.mixed().required('Avatar is a required.')
            })}
          >
            {({ values, handleSubmit, setFieldValue, touched, errors }) => (
              <form onSubmit={handleSubmit}>
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <Stack spacing={1.5} alignItems="center">
                      <UploadSingleFile setFieldValue={setFieldValue} file={values.files} error={touched.files && !!errors.files} />
                      {touched.files && errors.files && (
                        <FormHelperText error id="standard-weight-helper-text-password-login">
                          {errors.files}
                        </FormHelperText>
                      )}
                    </Stack>
                  </Grid>
                </Grid>
              </form>
            )}
          </Formik>
        </MainCard>
      </Grid>
      <Grid item xs={12}>
        <MainCard title="Upload Avatar">
          <Formik
            initialValues={{ files: null }}
            onSubmit={(values: any) => {
              // submit form
            }}
            validationSchema={yup.object().shape({
              files: yup.mixed().required('Avatar is a required.')
            })}
          >
            {({ values, handleSubmit, setFieldValue, touched, errors }) => (
              <form onSubmit={handleSubmit}>
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <Stack spacing={1.5} alignItems="center">
                      <UploadAvatar setFieldValue={setFieldValue} file={values.files} error={touched.files && !!errors.files} />
                      {touched.files && errors.files && (
                        <FormHelperText error id="standard-weight-helper-text-password-login">
                          {errors.files}
                        </FormHelperText>
                      )}
                      <Stack spacing={0}>
                        <Typography align="center" variant="caption" color="secondary">
                          Allowed 'image/*'
                        </Typography>
                        <Typography align="center" variant="caption" color="secondary">
                          *.png, *.jpeg, *.jpg, *.gif
                        </Typography>
                      </Stack>
                    </Stack>
                  </Grid>
                  <Grid item xs={12}>
                    <Stack direction="row" justifyContent="flex-end" alignItems="center" spacing={2}>
                      <Button color="error" onClick={() => setFieldValue('files', null)}>
                        Cancel
                      </Button>
                      <Button type="submit" variant="contained">
                        Submit
                      </Button>
                    </Stack>
                  </Grid>
                </Grid>
              </form>
            )}
          </Formik>
        </MainCard>
      </Grid>
    </Grid>
  );
};

export default DropzonePage;
