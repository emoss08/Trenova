import { Variable, VariableCategory } from "@/types/workflow";

export function VariableCategoryItem({
  value,
  category,
  inputRef,
  onChange,
}: {
  value: string;
  inputRef: HTMLInputElement | null;
  category: VariableCategory;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-1">
      <VariableCategoryHeader category={category} />
      <div className="space-y-1">
        {category.variables.map((variable) => (
          <VariableItem
            key={variable.value}
            value={value}
            variable={variable}
            inputRef={inputRef}
            onChange={onChange}
          />
        ))}
      </div>
    </div>
  );
}

function VariableCategoryHeader({ category }: { category: VariableCategory }) {
  return (
    <div>
      <p className="text-xs font-medium text-foreground uppercase">
        {category.label}
      </p>
      <p className="text-2xs text-muted-foreground">{category.description}</p>
    </div>
  );
}

function VariableItem({
  value,
  variable,
  inputRef,
  onChange,
}: {
  value: string;
  variable: Variable;
  inputRef: HTMLInputElement | null;
  onChange: (value: string) => void;
}) {
  const insertVariable = (variable: string) => {
    if (!inputRef) return;

    const start = inputRef.selectionStart || 0;
    const end = inputRef.selectionEnd || 0;
    const newValue =
      value.substring(0, start) + variable + value.substring(end);

    onChange(newValue);

    setTimeout(() => {
      inputRef.focus();
      const newCursorPos = start + variable.length;
      inputRef.setSelectionRange(newCursorPos, newCursorPos);
    }, 0);
  };

  return (
    <button
      type="button"
      onClick={() => insertVariable(variable.value)}
      className="block w-full rounded-md p-2 text-left hover:bg-accent"
    >
      <div className="flex items-start justify-between">
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium">{variable.label}</div>
          <div className="font-mono text-xs text-muted-foreground">
            {variable.value}
          </div>
          <div className="text-xs text-muted-foreground">
            {variable.description}
          </div>
        </div>
      </div>
    </button>
  );
}
