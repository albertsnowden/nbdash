import { cva, type VariantProps } from "class-variance-authority";
import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium",
  {
    variants: {
      variant: {
        default: "border-nb-gray-800 bg-nb-gray-900 text-nb-gray-200",
        brand: "border-netbird-700 bg-netbird-950 text-netbird-400",
        success: "border-green-800 bg-green-900/40 text-green-400",
        warning: "border-yellow-800 bg-yellow-900/30 text-yellow-400",
        danger: "border-red-800 bg-red-900/30 text-red-400",
        beta: "border-nb-blue-700 bg-nb-blue-950 text-nb-blue-300",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

interface BadgeProps extends HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

export default function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
