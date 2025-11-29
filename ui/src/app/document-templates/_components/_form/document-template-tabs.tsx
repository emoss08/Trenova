import { documentTemplateEditorParser } from "@/app/workers/_components/pto/use-document-template-state";
import { CodeEditorField } from "@/components/fields/code-editor-field";
import { DocumentTemplateSchema } from "@/lib/schemas/document-template-schema";
import { useQueryStates } from "nuqs";
import React, { Activity } from "react";
import { useFormContext } from "react-hook-form";
import { DocumentTemplateVariableList } from "../_variables/document-template-variable-list";

export function DocumentTemplateTabContent() {
  const [searchParams] = useQueryStates(documentTemplateEditorParser);

  return (
    <DocumentTemplateTabOuter>
      <DocumentTemplateTabInner>
        <Activity
          mode={searchParams.editorTab === "html" ? "visible" : "hidden"}
        >
          <DocumentTemplateHTMLTab />
        </Activity>
        <Activity
          mode={searchParams.editorTab === "css" ? "visible" : "hidden"}
        >
          <DocumentTemplateCSSTab />
        </Activity>
        <Activity
          mode={searchParams.editorTab === "header" ? "visible" : "hidden"}
        >
          <DocumentTemplateHeaderTab />
        </Activity>
        <Activity
          mode={searchParams.editorTab === "footer" ? "visible" : "hidden"}
        >
          <DocumentTemplateFooterTab />
        </Activity>
      </DocumentTemplateTabInner>
      <Activity mode={searchParams.showVariables ? "visible" : "hidden"}>
        <DocumentTemplateVariableList />
      </Activity>
    </DocumentTemplateTabOuter>
  );
}

function DocumentTemplateTabOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex size-full flex-row">{children}</div>;
}

function DocumentTemplateTabInner({ children }: { children: React.ReactNode }) {
  return <div className="flex w-full flex-col">{children}</div>;
}

// Calculate editor height based on viewport, accounting for header/toolbar space
// Uses clamp() to ensure minimum 300px, preferred calc, and maximum 1200px
const EDITOR_HEIGHT = "clamp(300px, calc(100vh - 280px), 1200px)";

export function DocumentTemplateHTMLTab() {
  const { control } = useFormContext<DocumentTemplateSchema>();

  return (
    <div className="flex-1 p-4">
      <CodeEditorField
        control={control}
        name="htmlContent"
        language="html"
        height={EDITOR_HEIGHT}
        placeholder="<!-- Enter your HTML template content here -->
<div class='document'>
  <h1>{{ .DocumentTitle }}</h1>
  <p>Date: {{ formatDate .DocumentDate }}</p>

  <!-- Use the variable palette on the right to insert template variables -->
</div>"
        rules={{ required: "HTML content is required" }}
      />
    </div>
  );
}

export function DocumentTemplateCSSTab() {
  const { control } = useFormContext<DocumentTemplateSchema>();

  return (
    <div className="min-h-0 flex-1 p-4">
      <CodeEditorField
        control={control}
        name="cssContent"
        language="css"
        height={EDITOR_HEIGHT}
        placeholder="/* Custom styles for your template */

.document {
  font-family: 'Inter', system-ui, sans-serif;
  max-width: 800px;
  margin: 0 auto;
  padding: 40px;
  color: #1f2937;
}

h1 {
  font-size: 28px;
  font-weight: 700;
  color: #111827;
  margin-bottom: 16px;
}"
      />
    </div>
  );
}

export function DocumentTemplateHeaderTab() {
  const { control } = useFormContext<DocumentTemplateSchema>();

  return (
    <div className="min-h-0 flex-1 p-4">
      <CodeEditorField
        control={control}
        name="headerHtml"
        language="html"
        height={EDITOR_HEIGHT}
        placeholder="<!-- Page header (appears on every page) -->
<div class='header'>
  <img src='{{ .CompanyLogo }}' alt='Logo' class='logo' />
  <span>{{ .CompanyName }}</span>
  <span class='page-info'>Page {{ .PageNumber }} of {{ .TotalPages }}</span>
</div>"
      />
    </div>
  );
}

export function DocumentTemplateFooterTab() {
  const { control } = useFormContext<DocumentTemplateSchema>();

  return (
    <div className="min-h-0 flex-1 p-4">
      <CodeEditorField
        control={control}
        name="footerHtml"
        language="html"
        height={EDITOR_HEIGHT}
        placeholder="<!-- Page footer (appears on every page) -->
<div class='footer'>
  <p>{{ .CompanyName }} | {{ .CompanyPhone }} | {{ .CompanyEmail }}</p>
  <p>Generated on {{ formatDate .GeneratedAt }}</p>
</div>"
      />
    </div>
  );
}
