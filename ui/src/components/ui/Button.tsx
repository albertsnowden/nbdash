import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-netbird-500 focus-visible:ring-offset-2 focus-visible:ring-offset-nb-gray-950 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        primary: "bg-netbird-500 text-white hover:bg-netbird-600 active:bg-netbird-700",
        secondary:
          "bg-nb-gray-900 text-nb-gray-50 border border-nb-gray-800 hover:bg-nb-gray-850",
        outline:
          "border border-nb-gray-800 text-nb-gray-100 hover:bg-nb-gray-900",
        ghost: "text-nb-gray-200 hover:bg-nb-gray-900",
        danger: "bg-red-600 text-white hover:bg-red-700",
        link: "text-netbird-500 underline-offset-4 hover:underline",
      },
      size: {
        sm: "h-8 px-3 text-xs",
        md: "h-9 px-4",
        lg: "h-10 px-5",
        icon: "h-9 w-9 shrink-0",
      },
    },
    defaultVariants: {
      variant: "secondary",
      size: "md",
    },
  },
);

interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

export default function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : "button";
  return (
    <Comp className={cn(buttonVariants({ variant, size }), className)} {...props} />
  );
}
