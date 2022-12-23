import { useDispatch } from 'react-redux';

// material-ui
import { Button, Grid } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import { openSnackbar } from 'store/reducers/snackbar';

// ==============================|| COMPONENTS - SNACKBAR ||============================== //

const ComponentSnackbar = () => {
  const dispatch = useDispatch();

  const basicSnackbarCodeString = `<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is default message',
        variant: 'alert',
        close: false
      })
    )
  }
>
  Default
</Button>
<Button
  variant="contained"
  color="secondary"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is secondary message',
        variant: 'alert',
        alert: { color: 'secondary' },
        close: false
      })
    )
  }
>
  Secondary
</Button>
<Button
  variant="contained"
  color="success"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is success message',
        variant: 'alert',
        alert: {
          color: 'success'
        },
        close: false
      })
    )
  }
>
  Success
</Button>
<Button
  variant="contained"
  color="warning"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is warning message',
        variant: 'alert',
        alert: {
          color: 'warning'
        },
        close: false
      })
    )
  }
>
  Warning
</Button>
<Button
  variant="contained"
  color="info"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is info message',
        variant: 'alert',
        alert: {
          color: 'info'
        },
        close: false
      })
    )
  }
>
  Info
</Button>
<Button
  variant="contained"
  color="error"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is error message',
        variant: 'alert',
        alert: {
          color: 'error'
        },
        close: false
      })
    )
  }
>
  Error
</Button>`;

  const outlinedSnackbarCodeString = `<Button
  variant="outlined"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is default message',
        variant: 'alert',
        alert: {
          variant: 'outlined'
        },
        close: false
      })
    )
  }
>
  Default
</Button>
<Button
  variant="outlined"
  color="secondary"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is secondary message',
        variant: 'alert',
        alert: {
          variant: 'outlined',
          color: 'secondary'
        },
        close: false
      })
    )
  }
>
  Secondary
</Button>
<Button
  variant="outlined"
  color="success"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is success message',
        variant: 'alert',
        alert: {
          variant: 'outlined',
          color: 'success'
        },
        close: false
      })
    )
  }
>
  Success
</Button>
<Button
  variant="outlined"
  color="warning"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is warning message',
        variant: 'alert',
        alert: {
          variant: 'outlined',
          color: 'warning'
        },
        close: false
      })
    )
  }
>
  Warning
</Button>
<Button
  variant="outlined"
  color="info"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is info message',
        variant: 'alert',
        alert: {
          variant: 'outlined',
          color: 'info'
        },
        close: false
      })
    )
  }
>
  Info
</Button>
<Button
  variant="outlined"
  color="error"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is error message',
        variant: 'alert',
        alert: {
          variant: 'outlined',
          color: 'error'
        },
        close: false
      })
    )
  }
>
  Error
</Button>`;

  const closeSnackbarCodeString = `<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is default message',
        variant: 'alert'
      })
    )
  }
>
  Default
</Button>
<Button
  variant="contained"
  color="secondary"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is secondary message',
        variant: 'alert',
        alert: { color: 'secondary' }
      })
    )
  }
>
  Secondary
</Button>
<Button
  variant="contained"
  color="success"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is success message',
        variant: 'alert',
        alert: {
          color: 'success'
        }
      })
    )
  }
>
  Success
</Button>
<Button
  variant="contained"
  color="warning"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is warning message',
        variant: 'alert',
        alert: {
          color: 'warning'
        }
      })
    )
  }
>
  Warning
</Button>
<Button
  variant="contained"
  color="info"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is info message',
        variant: 'alert',
        alert: {
          color: 'info'
        }
      })
    )
  }
>
  Info
</Button>
<Button
  variant="contained"
  color="error"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is error message',
        variant: 'alert',
        alert: {
          color: 'error'
        }
      })
    )
  }
>
  Error
</Button>`;

  const actionSnackbarCodeString = `<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is default message',
        variant: 'alert',
        actionButton: true
      })
    )
  }
>
  Default
</Button>
<Button
  variant="contained"
  color="secondary"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is secondary message',
        variant: 'alert',
        alert: { color: 'secondary' },
        actionButton: true
      })
    )
  }
>
  Secondary
</Button>
<Button
  variant="contained"
  color="success"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is success message',
        variant: 'alert',
        alert: {
          color: 'success'
        },
        actionButton: true
      })
    )
  }
>
  Success
</Button>
<Button
  variant="contained"
  color="warning"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is warning message',
        variant: 'alert',
        alert: {
          color: 'warning'
        },
        actionButton: true
      })
    )
  }
>
  Warning
</Button>
<Button
  variant="contained"
  color="info"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is info message',
        variant: 'alert',
        alert: {
          color: 'info'
        },
        actionButton: true
      })
    )
  }
>
  Info
</Button>
<Button
  variant="contained"
  color="error"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is error message',
        variant: 'alert',
        alert: {
          color: 'error'
        },
        actionButton: true
      })
    )
  }
>
  Error
</Button>`;

  const positionSnackbarCodeString = `<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        anchorOrigin: { vertical: 'top', horizontal: 'left' },
        message: 'This is an top-left message!'
      })
    )
  }
>
  Top-Left
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        anchorOrigin: { vertical: 'top', horizontal: 'center' },
        message: 'This is an top-center message!'
      })
    )
  }
>
  Top-Center
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        anchorOrigin: { vertical: 'top', horizontal: 'right' },
        message: 'This is an top-right message!'
      })
    )
  }
>
  Top-Right
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        anchorOrigin: { vertical: 'bottom', horizontal: 'right' },
        message: 'This is an bottom-right message!'
      })
    )
  }
>
  Bottom-Right
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        anchorOrigin: { vertical: 'bottom', horizontal: 'center' },
        message: 'This is an bottom-center message!'
      })
    )
  }
>
  Bottom-Center
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        anchorOrigin: { vertical: 'bottom', horizontal: 'left' },
        message: 'This is an bottom-left message!'
      })
    )
  }
>
  Bottom-Left
</Button>`;

  const transitionsSnackbarCodeString = `<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is an fade message!',
        transition: 'Fade'
      })
    )
  }
>
  Default/Fade
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is an slide-left message!',
        transition: 'SlideLeft'
      })
    )
  }
>
  Slide Left
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is an slide-up message!',
        transition: 'SlideUp'
      })
    )
  }
>
  Slide Up
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is an slide-right message!',
        transition: 'SlideRight'
      })
    )
  }
>
  Slide Right
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is an slide-down message!',
        transition: 'SlideDown'
      })
    )
  }
>
  Slide Down
</Button>
<Button
  variant="contained"
  onClick={() =>
    dispatch(
      openSnackbar({
        open: true,
        message: 'This is an grow message!',
        transition: 'Grow'
      })
    )
  }
>
  Grow
</Button>`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Snackbar"
        caption="Snackbars provide brief notifications. The component is also known as a toast."
        directory="src/pages/components-overview/snackbar"
        link="https://mui.com/material-ui/react-snackbar/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12} lg={6}>
            <MainCard title="Basic" codeString={basicSnackbarCodeString}>
              <Grid container spacing={2}>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is default message',
                          variant: 'alert',
                          close: false
                        })
                      )
                    }
                  >
                    Default
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="secondary"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is secondary message',
                          variant: 'alert',
                          alert: { color: 'secondary' },
                          close: false
                        })
                      )
                    }
                  >
                    Secondary
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="success"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is success message',
                          variant: 'alert',
                          alert: {
                            color: 'success'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Success
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="warning"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is warning message',
                          variant: 'alert',
                          alert: {
                            color: 'warning'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Warning
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="info"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is info message',
                          variant: 'alert',
                          alert: {
                            color: 'info'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Info
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="error"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is error message',
                          variant: 'alert',
                          alert: {
                            color: 'error'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Error
                  </Button>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Outlined" codeString={outlinedSnackbarCodeString}>
              <Grid container spacing={2}>
                <Grid item>
                  <Button
                    variant="outlined"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is default message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Default
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="secondary"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is secondary message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'secondary'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Secondary
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="success"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is success message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'success'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Success
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="warning"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is warning message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'warning'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Warning
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="info"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is info message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'info'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Info
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="error"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is error message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'error'
                          },
                          close: false
                        })
                      )
                    }
                  >
                    Error
                  </Button>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="With Close" codeString={closeSnackbarCodeString}>
              <Grid container spacing={2}>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is default message',
                          variant: 'alert'
                        })
                      )
                    }
                  >
                    Default
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="secondary"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is secondary message',
                          variant: 'alert',
                          alert: {
                            color: 'secondary'
                          }
                        })
                      )
                    }
                  >
                    Secondary
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="success"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is success message',
                          variant: 'alert',
                          alert: {
                            color: 'success'
                          }
                        })
                      )
                    }
                  >
                    Success
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="warning"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is warning message',
                          variant: 'alert',
                          alert: {
                            color: 'warning'
                          }
                        })
                      )
                    }
                  >
                    Warning
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="info"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is info message',
                          variant: 'alert',
                          alert: {
                            color: 'info'
                          }
                        })
                      )
                    }
                  >
                    Info
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    color="error"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is error message',
                          variant: 'alert',
                          alert: {
                            color: 'error'
                          }
                        })
                      )
                    }
                  >
                    Error
                  </Button>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="With Close + Action" codeString={actionSnackbarCodeString}>
              <Grid container spacing={2}>
                <Grid item>
                  <Button
                    variant="outlined"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is default message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined'
                          },
                          actionButton: true
                        })
                      )
                    }
                  >
                    Default
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="secondary"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is secondary message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'secondary'
                          },
                          actionButton: true
                        })
                      )
                    }
                  >
                    Secondary
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="success"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is success message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'success'
                          },
                          actionButton: true
                        })
                      )
                    }
                  >
                    Success
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="warning"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is warning message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'warning'
                          },
                          actionButton: true
                        })
                      )
                    }
                  >
                    Warning
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="info"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is info message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'info'
                          },
                          actionButton: true
                        })
                      )
                    }
                  >
                    Info
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="outlined"
                    color="error"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is error message',
                          variant: 'alert',
                          alert: {
                            variant: 'outlined',
                            color: 'error'
                          },
                          actionButton: true
                        })
                      )
                    }
                  >
                    Error
                  </Button>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Position" codeString={positionSnackbarCodeString}>
              <Grid container spacing={2}>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          anchorOrigin: { vertical: 'top', horizontal: 'left' },
                          message: 'This is an top-left message!'
                        })
                      )
                    }
                  >
                    Top-Left
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          anchorOrigin: { vertical: 'top', horizontal: 'center' },
                          message: 'This is an top-center message!'
                        })
                      )
                    }
                  >
                    Top-Center
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          anchorOrigin: { vertical: 'top', horizontal: 'right' },
                          message: 'This is an top-right message!'
                        })
                      )
                    }
                  >
                    Top-Right
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          anchorOrigin: { vertical: 'bottom', horizontal: 'right' },
                          message: 'This is an bottom-right message!'
                        })
                      )
                    }
                  >
                    Bottom-Right
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          anchorOrigin: { vertical: 'bottom', horizontal: 'center' },
                          message: 'This is an bottom-center message!'
                        })
                      )
                    }
                  >
                    Bottom-Center
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          anchorOrigin: { vertical: 'bottom', horizontal: 'left' },
                          message: 'This is an bottom-left message!'
                        })
                      )
                    }
                  >
                    Bottom-Left
                  </Button>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Transitions" codeString={transitionsSnackbarCodeString}>
              <Grid container spacing={2}>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is an fade message!',
                          transition: 'Fade'
                        })
                      )
                    }
                  >
                    Default/Fade
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is an slide-left message!',
                          transition: 'SlideLeft'
                        })
                      )
                    }
                  >
                    Slide Left
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is an slide-up message!',
                          transition: 'SlideUp'
                        })
                      )
                    }
                  >
                    Slide Up
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is an slide-right message!',
                          transition: 'SlideRight'
                        })
                      )
                    }
                  >
                    Slide Right
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is an slide-down message!',
                          transition: 'SlideDown'
                        })
                      )
                    }
                  >
                    Slide Down
                  </Button>
                </Grid>
                <Grid item>
                  <Button
                    variant="contained"
                    onClick={() =>
                      dispatch(
                        openSnackbar({
                          open: true,
                          message: 'This is an grow message!',
                          transition: 'Grow'
                        })
                      )
                    }
                  >
                    Grow
                  </Button>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentSnackbar;
