// material-ui
import { styled } from '@mui/material/styles';
import { Box, Button, Stack } from '@mui/material';

// third-party
import { useDropzone } from 'react-dropzone';

// project import
import RejectionFiles from './RejectionFiles';
import PlaceholderContent from './PlaceholderContent';
import FilesPreview from './FilesPreview';

// types
import { CustomFile, DropzopType, UploadMultiFileProps } from 'types/dropzone';

const DropzoneWrapper = styled('div')(({ theme }) => ({
  outline: 'none',
  padding: theme.spacing(5, 1),
  borderRadius: theme.shape.borderRadius,
  backgroundColor: theme.palette.background.paper,
  border: `1px dashed ${theme.palette.secondary.main}`,
  '&:hover': { opacity: 0.72, cursor: 'pointer' }
}));

// ==============================|| UPLOAD - MULTIPLE FILE ||============================== //

const MultiFileUpload = ({ error, showList = false, files, type, setFieldValue, sx, onUpload, ...other }: UploadMultiFileProps) => {
  const { getRootProps, getInputProps, isDragActive, isDragReject, fileRejections } = useDropzone({
    multiple: true,
    onDrop: (acceptedFiles: CustomFile[]) => {
      if (files) {
        setFieldValue('files', [
          ...files,
          ...acceptedFiles.map((file: CustomFile) =>
            Object.assign(file, {
              preview: URL.createObjectURL(file)
            })
          )
        ]);
      } else {
        setFieldValue(
          'files',
          acceptedFiles.map((file: CustomFile) =>
            Object.assign(file, {
              preview: URL.createObjectURL(file)
            })
          )
        );
      }
    }
  });

  const onRemoveAll = () => {
    setFieldValue('files', null);
  };

  const onRemove = (file: File | string) => {
    const filteredItems = files && files.filter((_file) => _file !== file);
    setFieldValue('files', filteredItems);
  };

  return (
    <>
      <Box
        sx={{
          width: '100%',
          ...(type === DropzopType.standard && { width: 'auto', display: 'flex' }),
          ...sx
        }}
      >
        <Stack {...(type === DropzopType.standard && { alignItems: 'center' })}>
          <DropzoneWrapper
            {...getRootProps()}
            sx={{
              ...(type === DropzopType.standard && {
                p: 0,
                m: 1,
                width: 64,
                height: 64
              }),
              ...(isDragActive && { opacity: 0.72 }),
              ...((isDragReject || error) && {
                color: 'error.main',
                borderColor: 'error.light',
                bgcolor: 'error.lighter'
              })
            }}
          >
            <input {...getInputProps()} />
            <PlaceholderContent type={type} />
          </DropzoneWrapper>
          {type === DropzopType.standard && files && files.length > 1 && (
            <Button variant="contained" color="error" size="extraSmall" onClick={onRemoveAll}>
              Remove all
            </Button>
          )}
        </Stack>
        {fileRejections.length > 0 && <RejectionFiles fileRejections={fileRejections} />}
        {files && files.length > 0 && <FilesPreview files={files} showList={showList} onRemove={onRemove} type={type} />}
      </Box>

      {type !== DropzopType.standard && files && files.length > 0 && (
        <Stack direction="row" justifyContent="flex-end" spacing={1.5} sx={{ mt: 1.5 }}>
          <Button color="inherit" size="small" onClick={onRemoveAll}>
            Remove all
          </Button>
          <Button size="small" variant="contained" onClick={onUpload}>
            Upload files
          </Button>
        </Stack>
      )}
    </>
  );
};

export default MultiFileUpload;
