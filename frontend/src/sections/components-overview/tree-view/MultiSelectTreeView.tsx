// material-ui
import { TreeView, TreeItem } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// assets
import { DownOutlined, RightOutlined } from '@ant-design/icons';

// ==============================|| TREE VIEW - MULTI-SELECT ||============================== //

export default function MultiSelectTreeView() {
  const multiTreeviewCodeString = `<TreeView
  aria-label="multi-select"
  defaultCollapseIcon={<DownOutlined />}
  defaultExpandIcon={<RightOutlined />}
  multiSelect
  defaultExpanded={['1']}
  sx={{ height: 216, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
>
  <TreeItem nodeId="1" label="Applications">
    <TreeItem nodeId="2" label="Calendar" />
    <TreeItem nodeId="3" label="Chrome" />
    <TreeItem nodeId="4" label="Webstorm" />
  </TreeItem>
  <TreeItem nodeId="5" label="Documents">
    <TreeItem nodeId="6" label="MUI">
      <TreeItem nodeId="7" label="src">
        <TreeItem nodeId="8" label="index.js" />
        <TreeItem nodeId="9" label="tree-view.js" />
      </TreeItem>
    </TreeItem>
  </TreeItem>
</TreeView>`;

  return (
    <MainCard title="Multi-Select" codeString={multiTreeviewCodeString}>
      <TreeView
        aria-label="multi-select"
        defaultCollapseIcon={<DownOutlined />}
        defaultExpandIcon={<RightOutlined />}
        multiSelect
        defaultExpanded={['1']}
        sx={{ height: 216, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
      >
        <TreeItem nodeId="1" label="Applications">
          <TreeItem nodeId="2" label="Calendar" />
          <TreeItem nodeId="3" label="Chrome" />
          <TreeItem nodeId="4" label="Webstorm" />
        </TreeItem>
        <TreeItem nodeId="5" label="Documents">
          <TreeItem nodeId="6" label="MUI">
            <TreeItem nodeId="7" label="src">
              <TreeItem nodeId="8" label="index.js" />
              <TreeItem nodeId="9" label="tree-view.js" />
            </TreeItem>
          </TreeItem>
        </TreeItem>
      </TreeView>
    </MainCard>
  );
}
