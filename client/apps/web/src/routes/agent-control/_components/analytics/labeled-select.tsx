import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";

type LabeledSelectProps<T extends string> = {
  label: string;
  items: readonly { value: T; label: string }[];
  value: T;
  /** Narrowed to something other than everything: the trigger is marked. */
  active?: boolean;
  onChange: (value: T) => void;
};

/** A select whose trigger reads as its label and choice together: "Agent  Every agent". */
export function LabeledSelect<T extends string>({
  label,
  items,
  value,
  active = false,
  onChange,
}: LabeledSelectProps<T>) {
  const chosen = items.find((item) => item.value === value) ?? items[0];

  return (
    <div className={active ? "an-pick on" : "an-pick"}>
      <Select
        items={items}
        value={chosen?.value ?? null}
        onValueChange={(next) => next && onChange(next as T)}
      >
        <SelectTrigger aria-label={label}>
          <SelectValue>
            <span className="k">{label}</span>
            <span className="v">{chosen?.label}</span>
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  );
}
