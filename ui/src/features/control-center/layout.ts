import type { Node } from "@xyflow/react";

// Every node carries its target column (0 = leftmost anchor) in
// data._col — set by whichever view builder constructs it. Positioning is
// then a pure "bucket by column, space each bucket's rows evenly and
// center it vertically on y=0" pass — no physics simulation needed, since
// every node in a given view has exactly one fixed role (anchor / source
// peer / policy / destination), unlike the reference dashboard's
// force-directed layout which also handles the free-floating network
// overview (not reimplemented here; see ControlCenterPage's Networks tab).
export const COLUMN_WIDTH = 300;
export const ROW_HEIGHT = 84;

export function colOf(node: Node): number {
  return (node.data as { _col?: number })?._col ?? 0;
}

export function layoutColumns(nodes: Node[]): Node[] {
  const byColumn = new Map<number, Node[]>();
  for (const node of nodes) {
    const col = (node.data as { _col?: number })?._col ?? 0;
    const bucket = byColumn.get(col);
    if (bucket) bucket.push(node);
    else byColumn.set(col, [node]);
  }

  const positioned: Node[] = [];
  for (const [col, bucket] of byColumn) {
    const totalHeight = (bucket.length - 1) * ROW_HEIGHT;
    const startY = -totalHeight / 2;
    bucket.forEach((node, i) => {
      positioned.push({ ...node, position: { x: col * COLUMN_WIDTH, y: startY + i * ROW_HEIGHT } });
    });
  }
  return positioned;
}

// Expansion nodes (a destination group's member peers) hang off that one
// specific group node rather than occupying a shared global column, so
// they're positioned relative to the parent instead of going through
// layoutColumns.
export function layoutExpansion(parent: Node, count: number, col: number) {
  const rowHeight = ROW_HEIGHT * 0.7;
  const totalHeight = (count - 1) * rowHeight;
  const startY = parent.position.y - totalHeight / 2;
  return { x: col * COLUMN_WIDTH, startY, rowHeight };
}
