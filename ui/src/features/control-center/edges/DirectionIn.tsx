import { BaseEdge, getSimpleBezierPath, type EdgeProps } from "@xyflow/react";

// Policy edges: an animated dashed green line when the policy/rule is
// enabled, a static gray one when disabled — the graph's primary visual
// signal for "is traffic actually flowing here".
export default function DirectionIn({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}: EdgeProps) {
  const enabled = Boolean((data as { enabled?: boolean } | undefined)?.enabled);
  const [path] = getSimpleBezierPath({ sourceX, sourceY, sourcePosition, targetX, targetY, targetPosition });

  return (
    <BaseEdge
      id={id}
      path={path}
      style={{
        strokeWidth: 2,
        stroke: enabled ? "#0e9f6e" : "#595959",
        strokeDasharray: "5, 5",
        opacity: enabled ? 1 : 0.6,
      }}
    >
      {enabled && (
        <animate attributeName="stroke-dashoffset" from="20" to="0" dur="0.5s" repeatCount="indefinite" />
      )}
    </BaseEdge>
  );
}
