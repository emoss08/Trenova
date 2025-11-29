import { IconProp } from "@/components/ui/icons";
import {
  faCode,
  faFile,
  faPalette,
  faTableLayout,
} from "@fortawesome/pro-regular-svg-icons";
import { parseAsBoolean, parseAsString, parseAsStringLiteral } from "nuqs";

export type EditorTab = "html" | "css" | "header" | "footer";

export const editorTabChoices = ["html", "css", "header", "footer"] as const;

export const documentTemplateEditorParser = {
  editorTab: parseAsStringLiteral(editorTabChoices)
    .withDefault("html")
    .withOptions({
      shallow: true,
    }),
  showVariables: parseAsBoolean.withDefault(true).withOptions({}),
  showPreview: parseAsBoolean.withDefault(true).withOptions({}),
  isFullscreen: parseAsBoolean.withDefault(false).withOptions({}),
  variableSearchQuery: parseAsString.withDefault("").withOptions({}),
};

type EditorTabs = {
  id: (typeof editorTabChoices)[number];
  label: string;
  icon: IconProp;
  description: string;
  rotate?: boolean;
};

export const editorTabs: EditorTabs[] = [
  {
    id: "html",
    label: "Content",
    icon: faCode,
    description: "Main template HTML",
  },
  {
    id: "css" as const,
    label: "Styles",
    icon: faPalette,
    description: "Custom CSS",
  },
  {
    id: "header" as const,
    label: "Header",
    icon: faTableLayout,
    description: "Page header",
  },
  {
    id: "footer" as const,
    label: "Footer",
    icon: faFile,
    description: "Page footer",
    rotate: true,
  },
];
