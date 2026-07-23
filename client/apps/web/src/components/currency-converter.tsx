import { Button } from "@trenova/shared/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { cn } from "@trenova/shared/lib/utils";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useQuery, useMutation } from "@tanstack/react-query";
import type { RateConversionResult } from "@/services/exchange-rate";
import { Check, ChevronsUpDown, Repeat } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

const CURRENCIES = [
  { code: "USD", name: "US Dollar" },
  { code: "EUR", name: "Euro" },
  { code: "GBP", name: "British Pound" },
  { code: "CAD", name: "Canadian Dollar" },
  { code: "MXN", name: "Mexican Peso" },
  { code: "AUD", name: "Australian Dollar" },
  { code: "JPY", name: "Japanese Yen" },
  { code: "CNY", name: "Chinese Yuan" },
  { code: "CHF", name: "Swiss Franc" },
  { code: "BRL", name: "Brazilian Real" },
  { code: "INR", name: "Indian Rupee" },
  { code: "ARS", name: "Argentine Peso" },
  { code: "COP", name: "Colombian Peso" },
  { code: "CLP", name: "Chilean Peso" },
  { code: "PEN", name: "Peruvian Sol" },
  { code: "DOP", name: "Dominican Peso" },
  { code: "CRC", name: "Costa Rican Colon" },
  { code: "GTQ", name: "Guatemalan Quetzal" },
  { code: "HNL", name: "Honduran Lempira" },
  { code: "NIO", name: "Nicaraguan Cordoba" },
  { code: "PAB", name: "Panamanian Balboa" },
  { code: "UYU", name: "Uruguayan Peso" },
  { code: "VES", name: "Venezuelan Bolivar" },
  { code: "BMD", name: "Bermudian Dollar" },
  { code: "BSD", name: "Bahamian Dollar" },
  { code: "BBD", name: "Barbadian Dollar" },
  { code: "BZD", name: "Belize Dollar" },
  { code: "XCD", name: "East Caribbean Dollar" },
  { code: "HTG", name: "Haitian Gourde" },
  { code: "JMD", name: "Jamaican Dollar" },
  { code: "TTD", name: "Trinidad & Tobago Dollar" },
].sort((a, b) => a.code.localeCompare(b.code));

interface CurrencySelectProps {
  value: string;
  onChange: (value: string) => void;
  label: string;
}

function CurrencySelect({ value, onChange, label }: CurrencySelectProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="space-y-1.5">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger className="w-full">
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className="w-full justify-between font-mono text-sm"
          >
            {value ? (
              <span>
                <span className="font-semibold">{value}</span>
                <span className="ml-2 text-muted-foreground">
                  {CURRENCIES.find((c) => c.code === value)?.name}
                </span>
              </span>
            ) : (
              "Select currency..."
            )}
            <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[300px] p-0" align="start">
          <Command>
            <CommandInput placeholder="Search currency..." />
            <CommandList>
              <CommandEmpty>No currency found.</CommandEmpty>
              <CommandGroup>
                {CURRENCIES.map((currency) => (
                  <CommandItem
                    key={currency.code}
                    value={currency.code}
                    onSelect={(currentValue) => {
                      onChange(currentValue === value ? "" : currentValue);
                      setOpen(false);
                    }}
                  >
                    <Check
                      className={cn(
                        "mr-2 size-4",
                        value === currency.code ? "opacity-100" : "opacity-0",
                      )}
                    />
                    <span className="font-mono font-medium">{currency.code}</span>
                    <span className="ml-2 text-muted-foreground">{currency.name}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}

export function CurrencyConverter() {
  const [fromCurrency, setFromCurrency] = useState("USD");
  const [toCurrency, setToCurrency] = useState("EUR");
  const [amount, setAmount] = useState("100.00");

  const convertQuery = useQuery({
    ...queries.exchangeRate.convert(fromCurrency, toCurrency, Number.parseFloat(amount || "0")),
    enabled: !!fromCurrency && !!toCurrency && !!amount && Number.parseFloat(amount) > 0,
  });

  const swapCurrencies = useCallback(() => {
    setFromCurrency(toCurrency);
    setToCurrency(fromCurrency);
  }, [fromCurrency, toCurrency]);

  const refreshMutation = useMutation({
    mutationFn: () => apiService.exchangeRateService.refresh(fromCurrency),
    onSuccess: () => {
      toast.success("Exchange rates refreshed");
    },
  });

  const result = convertQuery.data as RateConversionResult | undefined;
  const rate = result ? Number.parseFloat(result.rate) : null;
  const converted = result ? Number.parseFloat(result.converted) : null;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-[1fr_auto_1fr] items-end gap-2">
        <CurrencySelect value={fromCurrency} onChange={setFromCurrency} label="From" />
        <div className="flex items-center pb-2">
          <Button
            variant="ghost"
            size="icon"
            className="size-8"
            onClick={swapCurrencies}
            title="Swap currencies"
          >
            <Repeat className="size-4" />
          </Button>
        </div>
        <CurrencySelect value={toCurrency} onChange={setToCurrency} label="To" />
      </div>

      <div className="space-y-1.5">
        <Label className="text-xs text-muted-foreground">Amount</Label>
        <Input
          type="number"
          step="0.01"
          min="0"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          className="font-mono text-lg"
          placeholder="0.00"
        />
      </div>

      <div className="rounded-lg border border-border bg-muted/30 p-4">
        {convertQuery.isLoading ? (
          <div className="flex items-center justify-center py-4">
            <Spinner className="size-5" />
          </div>
        ) : converted !== null && rate !== null ? (
          <div className="space-y-2 text-center">
            <div className="font-mono text-2xl font-bold tracking-tight">
              {converted.toLocaleString(undefined, {
                minimumFractionDigits: 2,
                maximumFractionDigits: 2,
              })}{" "}
              {toCurrency}
            </div>
            <div className="text-xs text-muted-foreground">
              1 {fromCurrency} ={" "}
              {rate.toLocaleString(undefined, {
                minimumFractionDigits: 6,
                maximumFractionDigits: 6,
              })}{" "}
              {toCurrency}
            </div>
            {result?.date && (
              <div className="text-xs text-muted-foreground">Rate as of {result.date}</div>
            )}
            {result?.provider && (
              <div className="text-xs text-muted-foreground">
                {result.provider} {result.rateType ?? "mid"} rate - not a settlement quote
              </div>
            )}
          </div>
        ) : (
          <div className="text-center text-sm text-muted-foreground">
            Enter an amount to convert
          </div>
        )}
      </div>

      <Button
        variant="outline"
        size="sm"
        className="w-full"
        onClick={() => refreshMutation.mutateAsync()}
        isLoading={refreshMutation.isPending}
        loadingText="Refreshing..."
      >
        Refresh Rates
      </Button>
    </div>
  );
}
