import { Eye, EyeOff } from "lucide-react";
import { useState, type InputHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "prefix"> {
  error?: boolean;
  prefix?: ReactNode;
  suffix?: ReactNode;
}

export default function Input({
  className,
  type = "text",
  error,
  prefix,
  suffix,
  disabled,
  ...props
}: InputProps) {
  const [reveal, setReveal] = useState(false);
  const isPassword = type === "password";
  const resolvedType = isPassword ? (reveal ? "text" : "password") : type;

  return (
    <div
      className={cn(
        "flex h-9 items-center gap-2 rounded-md border border-nb-gray-800 bg-nb-gray-940 px-3 text-sm text-nb-gray-50 transition-colors",
        "focus-within:border-netbird-500 focus-within:ring-1 focus-within:ring-netbird-500",
        error && "border-red-600 focus-within:border-red-600 focus-within:ring-red-600",
        disabled && "cursor-not-allowed opacity-50",
        className,
      )}
    >
      {prefix && <span className="shrink-0 text-nb-gray-400">{prefix}</span>}
      <input
        type={resolvedType}
        disabled={disabled}
        className="h-full w-full min-w-0 bg-transparent placeholder:text-nb-gray-500 focus:outline-none disabled:cursor-not-allowed"
        {...props}
      />
      {isPassword && (
        <button
          type="button"
          tabIndex={-1}
          onClick={() => setReveal((v) => !v)}
          className="shrink-0 text-nb-gray-400 hover:text-nb-gray-100"
          aria-label={reveal ? "Hide value" : "Show value"}
        >
          {reveal ? <EyeOff size={16} /> : <Eye size={16} />}
        </button>
      )}
      {suffix && <span className="shrink-0 text-nb-gray-400">{suffix}</span>}
    </div>
  );
}
