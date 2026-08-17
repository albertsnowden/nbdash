import { Info } from "lucide-react";
import type { ReactNode } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";

// A small inline (i) icon that reveals a hint on hover/focus — for a short
// explanation next to a field label that would otherwise need a full
// paragraph of helper text under the input.
export default function InfoTooltip({ children }: { children: ReactNode }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className="inline-flex text-nb-gray-500 hover:text-nb-gray-200"
          aria-label="More information"
        >
          <Info size={13} />
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-64">{children}</TooltipContent>
    </Tooltip>
  );
}
