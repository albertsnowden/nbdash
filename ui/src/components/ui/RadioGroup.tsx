import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import { Circle } from "lucide-react";
import { cn } from "@/lib/utils";

export const RadioGroup = RadioGroupPrimitive.Root;

export function RadioGroupItem({
  className,
  ...props
}: React.ComponentProps<typeof RadioGroupPrimitive.Item>) {
  return (
    <RadioGroupPrimitive.Item
      className={cn(
        "flex h-4 w-4 shrink-0 items-center justify-center rounded-full border border-nb-gray-700 bg-nb-gray-940 transition-colors",
        "data-[state=checked]:border-netbird-500",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-netbird-500 focus-visible:ring-offset-2 focus-visible:ring-offset-nb-gray-950",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    >
      <RadioGroupPrimitive.Indicator>
        <Circle size={8} className="fill-netbird-500 text-netbird-500" />
      </RadioGroupPrimitive.Indicator>
    </RadioGroupPrimitive.Item>
  );
}
