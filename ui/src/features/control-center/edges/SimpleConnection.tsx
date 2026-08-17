import { BaseEdge, getSimpleBezierPath, type EdgeProps } from "@xyflow/react";

// Plain membership edges — a group node to one of its expanded member
// peers. No animation; these represent static structure, not live traffic.
export default function SimpleConnection({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
}: EdgeProps) {
  const [path] = getSimpleBezierPath({ sourceX, sourceY, sourcePosition, targetX, targetY, targetPosition });

  return <BaseEdge id={id} path={path} style={{ strokeWidth: 1.5, stroke: "#595959" }} />;
}
