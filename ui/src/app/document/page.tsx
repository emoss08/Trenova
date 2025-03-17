import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import {
  faFileAlt,
  faFileContract,
  faFileExcel,
  faFileImage,
  faFilePdf,
  faFileWord,
} from "@fortawesome/pro-regular-svg-icons";
import FolderList from "./_components/folder-list";

function getFileIcon(fileType: string) {
  const type = fileType.toLowerCase();
  if (type.includes("pdf")) return faFilePdf;
  if (
    type.includes("image") ||
    type.includes("jpg") ||
    type.includes("png") ||
    type.includes("jpeg")
  )
    return faFileImage;
  if (
    type.includes("excel") ||
    type.includes("spreadsheet") ||
    type.includes("csv") ||
    type.includes("xlsx")
  )
    return faFileExcel;
  if (type.includes("word") || type.includes("doc")) return faFileWord;
  if (type.includes("contract")) return faFileContract;
  return faFileAlt;
}

export function Document() {
  return (
    <>
      <MetaTags title="Document Studio" description="Document Studio" />
      <LazyComponent>
        {/* <DocumentUploadExample /> */}
        <FolderList />
      </LazyComponent>
    </>
  );
}
