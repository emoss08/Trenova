import { Fragment, type ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface CommentBodyMentionAttrs {
  id: string;
  label: string;
}

export interface CommentBodyEntityRefAttrs {
  entityType: string;
  entityId: string;
  label: string;
}

export interface CommentBodyRendererProps {
  body: Record<string, unknown> | null | undefined;
  comment: string;
  className?: string;
  renderMention?: (attrs: CommentBodyMentionAttrs, key: string) => ReactNode;
  renderEntityRef?: (attrs: CommentBodyEntityRefAttrs, key: string) => ReactNode;
  renderLink?: (href: string, children: ReactNode, key: string) => ReactNode;
}

interface BodyNode {
  type?: string;
  text?: string;
  content?: unknown[];
  attrs?: Record<string, unknown>;
  marks?: Array<{ type?: string; attrs?: Record<string, unknown> }>;
}

function asNode(value: unknown): BodyNode | null {
  return value && typeof value === "object" ? (value as BodyNode) : null;
}

function nodeChildren(node: BodyNode): BodyNode[] {
  if (!Array.isArray(node.content)) return [];
  return node.content.map(asNode).filter((child): child is BodyNode => child != null);
}

function attrString(node: BodyNode, key: string): string {
  const value = node.attrs?.[key];
  return typeof value === "string" ? value : "";
}

function defaultMention(attrs: CommentBodyMentionAttrs, key: string): ReactNode {
  return (
    <span key={key} className="rounded bg-blue-500/10 px-0.5 font-medium text-blue-500">
      @{attrs.label}
    </span>
  );
}

function defaultEntityRef(attrs: CommentBodyEntityRefAttrs, key: string): ReactNode {
  return (
    <span key={key} className="rounded bg-violet-500/10 px-0.5 font-medium text-violet-500">
      #{attrs.label}
    </span>
  );
}

function defaultLink(href: string, children: ReactNode, key: string): ReactNode {
  return (
    <a
      key={key}
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="text-blue-500 underline underline-offset-2 hover:text-blue-600"
    >
      {children}
    </a>
  );
}

function isSafeHref(href: string): boolean {
  const lower = href.toLowerCase();
  return lower.startsWith("http://") || lower.startsWith("https://");
}

export function CommentBodyRenderer({
  body,
  comment,
  className,
  renderMention = defaultMention,
  renderEntityRef = defaultEntityRef,
  renderLink = defaultLink,
}: CommentBodyRendererProps) {
  const content = renderBodyContent(body, { renderMention, renderEntityRef, renderLink });

  if (content == null) {
    return (
      <p className={cn("text-sm whitespace-pre-wrap text-foreground", className)}>{comment}</p>
    );
  }

  return <div className={cn("space-y-1.5 text-sm text-foreground", className)}>{content}</div>;
}

interface RenderContext {
  renderMention: (attrs: CommentBodyMentionAttrs, key: string) => ReactNode;
  renderEntityRef: (attrs: CommentBodyEntityRefAttrs, key: string) => ReactNode;
  renderLink: (href: string, children: ReactNode, key: string) => ReactNode;
}

function renderBodyContent(
  body: Record<string, unknown> | null | undefined,
  ctx: RenderContext,
): ReactNode | null {
  if (!body) return null;

  try {
    const root = asNode(body);
    if (!root || root.type !== "doc") return null;
    const blocks = nodeChildren(root);
    if (blocks.length === 0) return null;

    return blocks.map((block, index) => renderBlock(block, `b${index}`, ctx));
  } catch {
    return null;
  }
}

function renderBlock(node: BodyNode, key: string, ctx: RenderContext): ReactNode {
  switch (node.type) {
    case "paragraph": {
      const inlines = nodeChildren(node);
      if (inlines.length === 0) return <p key={key} className="min-h-[1.25rem]" />;
      return (
        <p key={key} className="wrap-break-word whitespace-pre-wrap">
          {inlines.map((inline, index) => renderInline(inline, `${key}-i${index}`, ctx))}
        </p>
      );
    }
    case "bulletList":
      return (
        <ul key={key} className="list-disc space-y-0.5 pl-5">
          {nodeChildren(node).map((item, index) => renderListItem(item, `${key}-li${index}`, ctx))}
        </ul>
      );
    case "orderedList": {
      const start = typeof node.attrs?.start === "number" ? node.attrs.start : undefined;
      return (
        <ol key={key} start={start} className="list-decimal space-y-0.5 pl-5">
          {nodeChildren(node).map((item, index) => renderListItem(item, `${key}-li${index}`, ctx))}
        </ol>
      );
    }
    default:
      return <Fragment key={key} />;
  }
}

function renderListItem(node: BodyNode, key: string, ctx: RenderContext): ReactNode {
  if (node.type !== "listItem") return <Fragment key={key} />;
  return (
    <li key={key}>
      {nodeChildren(node).map((child, index) => {
        if (child.type === "paragraph") {
          const inlines = nodeChildren(child);
          return (
            <Fragment key={`${key}-p${index}`}>
              {inlines.map((inline, inlineIdx) =>
                renderInline(inline, `${key}-p${index}-i${inlineIdx}`, ctx),
              )}
            </Fragment>
          );
        }
        return renderBlock(child, `${key}-c${index}`, ctx);
      })}
    </li>
  );
}

function renderInline(node: BodyNode, key: string, ctx: RenderContext): ReactNode {
  switch (node.type) {
    case "text":
      return renderTextNode(node, key, ctx);
    case "mention": {
      const label = attrString(node, "label");
      const id = attrString(node, "id");
      if (!label) return <Fragment key={key} />;
      return ctx.renderMention({ id, label }, key);
    }
    case "entityRef": {
      const label = attrString(node, "label");
      if (!label) return <Fragment key={key} />;
      return ctx.renderEntityRef(
        {
          entityType: attrString(node, "entityType"),
          entityId: attrString(node, "entityId"),
          label,
        },
        key,
      );
    }
    case "hardBreak":
      return <br key={key} />;
    default:
      return <Fragment key={key} />;
  }
}

function renderTextNode(node: BodyNode, key: string, ctx: RenderContext): ReactNode {
  const text = node.text ?? "";
  if (!text) return <Fragment key={key} />;

  let content: ReactNode = text;
  let linkHref: string | null = null;

  for (const mark of node.marks ?? []) {
    switch (mark.type) {
      case "bold":
        content = <strong>{content}</strong>;
        break;
      case "italic":
        content = <em>{content}</em>;
        break;
      case "underline":
        content = <u>{content}</u>;
        break;
      case "strike":
        content = <s>{content}</s>;
        break;
      case "link": {
        const href = typeof mark.attrs?.href === "string" ? mark.attrs.href : "";
        if (isSafeHref(href)) linkHref = href;
        break;
      }
      default:
        break;
    }
  }

  if (linkHref) {
    return ctx.renderLink(linkHref, content, key);
  }

  return <Fragment key={key}>{content}</Fragment>;
}

export function commentBodyToPlainText(body: Record<string, unknown> | null | undefined): string {
  const root = asNode(body ?? null);
  if (!root || root.type !== "doc") return "";

  const lines: string[] = [];
  for (const block of nodeChildren(root)) {
    collectBlockText(block, "", lines);
  }
  return lines.join("\n");
}

function collectBlockText(node: BodyNode, indent: string, lines: string[]): void {
  switch (node.type) {
    case "paragraph":
      lines.push(indent + collectInlineText(node));
      return;
    case "bulletList":
      collectListText(node, indent, lines, () => "- ");
      return;
    case "orderedList": {
      const start = typeof node.attrs?.start === "number" ? node.attrs.start : 1;
      collectListText(node, indent, lines, (index) => `${start + index}. `);
      return;
    }
    default:
  }
}

function collectListText(
  node: BodyNode,
  indent: string,
  lines: string[],
  marker: (index: number) => string,
): void {
  let itemIndex = 0;
  for (const item of nodeChildren(node)) {
    if (item.type !== "listItem") continue;
    let first = true;
    for (const child of nodeChildren(item)) {
      if (child.type === "paragraph") {
        lines.push((first ? indent + marker(itemIndex) : indent + "  ") + collectInlineText(child));
        first = false;
      } else {
        collectBlockText(child, indent + "  ", lines);
        first = false;
      }
    }
    if (first) lines.push(indent + marker(itemIndex));
    itemIndex++;
  }
}

function collectInlineText(node: BodyNode): string {
  let text = "";
  for (const inline of nodeChildren(node)) {
    switch (inline.type) {
      case "text":
        text += inline.text ?? "";
        break;
      case "mention":
        text += `@${attrString(inline, "label")}`;
        break;
      case "entityRef":
        text += `#${attrString(inline, "label")}`;
        break;
      case "hardBreak":
        text += "\n";
        break;
      default:
    }
  }
  return text;
}

export function extractMentionIdsFromBody(
  body: Record<string, unknown> | null | undefined,
): string[] {
  const root = asNode(body ?? null);
  if (!root) return [];

  const ids: string[] = [];
  const seen = new Set<string>();
  const walk = (node: BodyNode) => {
    if (node.type === "mention") {
      const id = attrString(node, "id");
      if (id && !seen.has(id)) {
        seen.add(id);
        ids.push(id);
      }
      return;
    }
    nodeChildren(node).forEach(walk);
  };
  walk(root);

  return ids;
}
