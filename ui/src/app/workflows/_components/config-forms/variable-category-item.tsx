import { Variable, VariableCategory } from "@/types/workflow";

export function VariableCategoryItem({
  category,
  onChange,
}: {
  category: VariableCategory;
  onChange: (value: string) => void;
}) {
  return (
    <div className="flex flex-col gap-1 px-2 py-2">
      <VariableCategoryHeader category={category} />
      <div className="space-y-1">
        {category.variables.map((variable) => (
          <VariableItem
            key={variable.value}
            variable={variable}
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
  variable,
  onChange,
}: {
  variable: Variable;
  onChange: (value: string) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onChange(variable.value)}
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
