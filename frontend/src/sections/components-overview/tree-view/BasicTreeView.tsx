// material-ui
import { TreeItem, TreeView } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// assets
import { DownOutlined, RightOutlined } from '@ant-design/icons';

// ==============================|| TREE VIEW - BASIC ||============================== //

export default function BasicTreeView() {
  const basicTreeviewCodeString = `<TreeView
  aria-label="file system navigator"
  defaultCollapseIcon={<DownOutlined />}
  defaultExpandIcon={<RightOutlined />}
  defaultExpanded={['5']}
  sx={{ height: 240, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
>
  <TreeItem nodeId="1" label="Applications">
    <TreeItem nodeId="2" label="Calendar" />
  </TreeItem>
  <TreeItem nodeId="5" label="Documents">
    <TreeItem nodeId="10" label="OSS" />
    <TreeItem nodeId="6" label="MUI">
      <TreeItem nodeId="8" label="index.js" />
    </TreeItem>
  </TreeItem>
</TreeView>`;

  return (
    <MainCard title="Basic" codeHighlight codeString={basicTreeviewCodeString}>
      <TreeView
        aria-label="file system navigator"
        defaultCollapseIcon={<DownOutlined />}
        defaultExpandIcon={<RightOutlined />}
        defaultExpanded={['5']}
        sx={{ height: 240, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
      >
        <TreeItem nodeId="1" label="Applications">
          <TreeItem nodeId="2" label="Calendar" />
        </TreeItem>
        <TreeItem nodeId="5" label="Documents">
          <TreeItem nodeId="10" label="OSS" />
          <TreeItem nodeId="6" label="MUI">
            <TreeItem nodeId="8" label="index.js" />
          </TreeItem>
        </TreeItem>
      </TreeView>
    </MainCard>
  );
}
