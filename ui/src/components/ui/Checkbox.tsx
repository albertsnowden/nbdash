import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

export default function Checkbox({
  className,
  ...props
}: React.ComponentProps<typeof CheckboxPrimitive.Root>) {
  return (
    <CheckboxPrimitive.Root
      className={cn(
        "peer h-4 w-4 shrink-0 rounded border border-nb-gray-700 bg-nb-gray-940 transition-colors",
        "data-[state=checked]:border-netbird-500 data-[state=checked]:bg-netbird-500",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-netbird-500 focus-visible:ring-offset-2 focus-visible:ring-offset-nb-gray-950",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator className="flex items-center justify-center text-white">
        <Check size={12} strokeWidth={3} />
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}
