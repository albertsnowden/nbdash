import type { Edge, Node } from "@xyflow/react";
import type { AgentNetworkPolicy, AgentNetworkProvider } from "@/api/agentNetwork";
import type { Group } from "@/api/groups";
import type { Network, NetworkResource } from "@/api/networks";
import type { Peer } from "@/api/peers";
import type { Policy } from "@/api/policies";
import type { User } from "@/api/team";
import { colOf, layoutColumns, layoutExpansion } from "./layout";

export interface ViewResult {
  nodes: Node[];
  edges: Edge[];
}

export interface ViewContext {
  policies: Policy[];
  groupsById: Map<string, Group>;
  peersById: Map<string, Peer>;
}

interface PolicyMatch {
  policy: Policy;
  destinationGroupIds: Set<string>;
}

// A policy "matches" a set of source groups if any of its rules sources
// from one of them — collecting the union of that rule's destination
// groups. A policy with multiple rules can partially match (some rules
// relevant, others not), so this walks every rule rather than assuming
// rules[0] is the only one that matters.
function policiesForSourceGroups(policies: Policy[], sourceGroupIds: Set<string>): PolicyMatch[] {
  const matches: PolicyMatch[] = [];
  for (const policy of policies) {
    const destinationGroupIds = new Set<string>();
    for (const rule of policy.rules) {
      if ((rule.sources ?? []).some((s) => sourceGroupIds.has(s.id))) {
        for (const d of rule.destinations ?? []) destinationGroupIds.add(d.id);
      }
    }
    if (destinationGroupIds.size > 0) matches.push({ policy, destinationGroupIds });
  }
  return matches;
}

// Builds the policy + destination-group columns reachable from one or more
// source edges. `sources` is a list of (nodeId, groupIds) pairs so callers
// can fan edges out from several source nodes at once (the Users view,
// where each of a user's peers is its own source node) or from a single
// anchor (Groups/Peers views).
function buildPolicyAndDestinationColumns(
  sources: { nodeId: string; groupIds: string[] }[],
  ctx: ViewContext,
  policyCol: number,
): ViewResult {
  const destCol = policyCol + 1;
  const allSourceGroupIds = new Set(sources.flatMap((s) => s.groupIds));
  const matches = policiesForSourceGroups(ctx.policies, allSourceGroupIds);

  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const groupEnabled = new Map<string, boolean>();

  for (const { policy, destinationGroupIds } of matches) {
    nodes.push({
      id: `policy-${policy.id}`,
      type: "policyNode",
      position: { x: 0, y: 0 },
      data: { policy, enabled: policy.enabled, _col: policyCol },
    });

    const policyGroupIds = new Set<string>();
    for (const rule of policy.rules) {
      if ((rule.sources ?? []).some((s) => allSourceGroupIds.has(s.id))) {
        for (const s of rule.sources ?? []) policyGroupIds.add(s.id);
      }
    }
    for (const { nodeId, groupIds } of sources) {
      if (groupIds.some((id) => policyGroupIds.has(id))) {
        edges.push({
          id: `${nodeId}-policy-${policy.id}`,
          source: nodeId,
          target: `policy-${policy.id}`,
          type: "in",
          data: { enabled: policy.enabled },
        });
      }
    }

    for (const gid of destinationGroupIds) {
      groupEnabled.set(gid, (groupEnabled.get(gid) ?? false) || policy.enabled);
      edges.push({
        id: `policy-${policy.id}-group-${gid}`,
        source: `policy-${policy.id}`,
        target: `group-${gid}`,
        type: "in",
        data: { enabled: policy.enabled },
      });
    }
  }

  for (const [gid, enabled] of groupEnabled) {
    const group = ctx.groupsById.get(gid);
    if (!group) continue;
    nodes.push({
      id: `group-${gid}`,
      type: "groupNode",
      position: { x: 0, y: 0 },
      data: { group, enabled, _col: destCol },
    });
  }

  return { nodes, edges };
}

// The member peers of an expanded destination group, positioned relative
// to that one group node rather than through the shared column layout —
// mirrors the reference dashboard's "expand in place" interaction.
function buildGroupExpansion(groupNode: Node, group: Group, peersById: Map<string, Peer>): ViewResult {
  const memberIds = (group.peers ?? []).map((p) => p.id);
  const peers = memberIds.map((id) => peersById.get(id)).filter((p): p is Peer => Boolean(p));
  const { x, startY, rowHeight } = layoutExpansion(groupNode, peers.length, colOf(groupNode) + 1);

  const nodes: Node[] = peers.map((peer, i) => ({
    id: `peer-${peer.id}`,
    type: "peerNode",
    position: { x, y: startY + i * rowHeight },
    data: { peer, enabled: true },
  }));
  const edges: Edge[] = peers.map((peer) => ({
    id: `group-${group.id}-peer-${peer.id}`,
    source: groupNode.id,
    target: `peer-${peer.id}`,
    type: "simple",
  }));
  return { nodes, edges };
}

function withExpansion(
  base: ViewResult,
  expandedGroupId: string | null,
  ctx: ViewContext,
): ViewResult {
  const positioned = layoutColumns(base.nodes);
  if (!expandedGroupId) return { nodes: positioned, edges: base.edges };

  const groupNode = positioned.find((n) => n.id === `group-${expandedGroupId}`);
  const group = ctx.groupsById.get(expandedGroupId);
  if (!groupNode || !group) return { nodes: positioned, edges: base.edges };

  const expansion = buildGroupExpansion(groupNode, group, ctx.peersById);
  return {
    nodes: [...positioned, ...expansion.nodes],
    edges: [...base.edges, ...expansion.edges],
  };
}

export function buildGroupView(
  groupId: string,
  ctx: ViewContext,
  allGroups: Group[],
  expandedGroupId: string | null,
  onChangeGroup: (id: string) => void,
): ViewResult {
  const anchor: Node = {
    id: "anchor",
    type: "selectGroupNode",
    position: { x: 0, y: 0 },
    data: { current: groupId, groups: allGroups, onChange: onChangeGroup, _col: 0 },
  };
  const columns = buildPolicyAndDestinationColumns([{ nodeId: "anchor", groupIds: [groupId] }], ctx, 1);
  return withExpansion({ nodes: [anchor, ...columns.nodes], edges: columns.edges }, expandedGroupId, ctx);
}

export function buildPeerView(
  peerId: string,
  ctx: ViewContext,
  allPeers: Peer[],
  expandedGroupId: string | null,
  onChangePeer: (id: string) => void,
): ViewResult {
  const anchor: Node = {
    id: "anchor",
    type: "selectPeerNode",
    position: { x: 0, y: 0 },
    data: { current: peerId, peers: allPeers, onChange: onChangePeer, _col: 0 },
  };
  const peer = ctx.peersById.get(peerId);
  const groupIds = (peer?.groups ?? []).map((g) => g.id);
  const columns = buildPolicyAndDestinationColumns([{ nodeId: "anchor", groupIds }], ctx, 1);
  return withExpansion({ nodes: [anchor, ...columns.nodes], edges: columns.edges }, expandedGroupId, ctx);
}

export function buildUserView(
  userId: string,
  ctx: ViewContext,
  allUsers: User[],
  allPeers: Peer[],
  expandedGroupId: string | null,
  onChangeUser: (id: string) => void,
): ViewResult {
  const anchor: Node = {
    id: "anchor",
    type: "selectUserNode",
    position: { x: 0, y: 0 },
    data: { current: userId, users: allUsers, onChange: onChangeUser, _col: 0 },
  };

  const userPeers = allPeers.filter((p) => p.user_id === userId);
  const peerNodes: Node[] = userPeers.map((peer) => ({
    id: `source-peer-${peer.id}`,
    type: "peerNode",
    position: { x: 0, y: 0 },
    data: { peer, enabled: true, _col: 1 },
  }));
  const anchorEdges: Edge[] = userPeers.map((peer) => ({
    id: `anchor-peer-${peer.id}`,
    source: "anchor",
    target: `source-peer-${peer.id}`,
    type: "simple",
  }));

  const sources = userPeers.map((peer) => ({
    nodeId: `source-peer-${peer.id}`,
    groupIds: (peer.groups ?? []).map((g) => g.id),
  }));
  const columns = buildPolicyAndDestinationColumns(sources, ctx, 2);

  return withExpansion(
    { nodes: [anchor, ...peerNodes, ...columns.nodes], edges: [...anchorEdges, ...columns.edges] },
    expandedGroupId,
    ctx,
  );
}

// Networks and Agent Network don't fit buildPolicyAndDestinationColumns'
// shape — a network isn't a source group, it's a container whose policies
// name both the destination groups guarding its resources *and* the source
// groups allowed in. Anchor stays column 0 like every other view; the
// columns unfold "which resources does this network have, which groups
// guard them, which policies name those groups, which groups do those
// policies let in" — reachability runs right-to-left, but nothing in this
// graph draws arrowheads, so the layout direction carries no false claim.
export function buildNetworkView(
  networkId: string,
  network: Network | undefined,
  resources: NetworkResource[],
  ctx: ViewContext,
  allNetworks: Network[],
  expandedGroupId: string | null,
  onChangeNetwork: (id: string) => void,
): ViewResult {
  const anchor: Node = {
    id: "anchor",
    type: "selectNetworkNode",
    position: { x: 0, y: 0 },
    data: { current: networkId, networks: allNetworks, onChange: onChangeNetwork, _col: 0 },
  };

  const nodes: Node[] = [anchor];
  const edges: Edge[] = [];

  const resourceNodes: Node[] = resources.map((r) => ({
    id: `resource-${r.id}`,
    type: "resourceNode",
    position: { x: 0, y: 0 },
    data: { resource: r, _col: 1 },
  }));
  nodes.push(...resourceNodes);
  edges.push(
    ...resources.map((r) => ({
      id: `anchor-resource-${r.id}`,
      source: "anchor",
      target: `resource-${r.id}`,
      type: "simple",
    })),
  );

  const policies = (network?.policies ?? [])
    .map((id) => ctx.policies.find((p) => p.id === id))
    .filter((p): p is Policy => Boolean(p));

  // A group that shows up as both a destination (guards a resource) and a
  // source (let in by some other policy) is rendered once, on the
  // destination side — arbitrary but deterministic, and rare in practice.
  const destGroupIds = new Set<string>();
  const sourceGroupIds = new Set<string>();

  for (const policy of policies) {
    for (const rule of policy.rules) {
      for (const d of rule.destinations ?? []) destGroupIds.add(d.id);
    }
  }
  for (const policy of policies) {
    for (const rule of policy.rules) {
      for (const s of rule.sources ?? []) {
        if (!destGroupIds.has(s.id)) sourceGroupIds.add(s.id);
      }
    }
  }

  for (const gid of destGroupIds) {
    const group = ctx.groupsById.get(gid);
    if (!group) continue;
    nodes.push({ id: `group-${gid}`, type: "groupNode", position: { x: 0, y: 0 }, data: { group, enabled: true, _col: 2 } });
  }
  for (const gid of sourceGroupIds) {
    const group = ctx.groupsById.get(gid);
    if (!group) continue;
    nodes.push({ id: `group-${gid}`, type: "groupNode", position: { x: 0, y: 0 }, data: { group, enabled: true, _col: 4 } });
  }

  for (const r of resources) {
    for (const g of r.groups ?? []) {
      if (!destGroupIds.has(g.id)) continue;
      edges.push({ id: `resource-${r.id}-group-${g.id}`, source: `resource-${r.id}`, target: `group-${g.id}`, type: "simple" });
    }
  }

  for (const policy of policies) {
    nodes.push({
      id: `policy-${policy.id}`,
      type: "policyNode",
      position: { x: 0, y: 0 },
      data: { policy, enabled: policy.enabled, _col: 3 },
    });
    for (const rule of policy.rules) {
      for (const d of rule.destinations ?? []) {
        edges.push({
          id: `group-${d.id}-policy-${policy.id}`,
          source: `group-${d.id}`,
          target: `policy-${policy.id}`,
          type: "in",
          data: { enabled: policy.enabled },
        });
      }
      for (const s of rule.sources ?? []) {
        const sourceNodeExists = sourceGroupIds.has(s.id) || destGroupIds.has(s.id);
        if (!sourceNodeExists) continue;
        edges.push({
          id: `policy-${policy.id}-group-${s.id}`,
          source: `policy-${policy.id}`,
          target: `group-${s.id}`,
          type: "in",
          data: { enabled: policy.enabled },
        });
      }
    }
  }

  return withExpansion({ nodes, edges }, expandedGroupId, ctx);
}

// Agent Network mirrors the Network view's shape but is much flatter — a
// provider only has agent policies pointing at it, and each policy only has
// source groups (no destination groups of its own; the provider is the one
// and only destination, already fixed as the anchor).
export function buildAgentNetworkView(
  providerId: string,
  allProviders: AgentNetworkProvider[],
  agentPolicies: AgentNetworkPolicy[],
  ctx: ViewContext,
  expandedGroupId: string | null,
  onChangeProvider: (id: string) => void,
): ViewResult {
  const anchor: Node = {
    id: "anchor",
    type: "selectProviderNode",
    position: { x: 0, y: 0 },
    data: { current: providerId, providers: allProviders, onChange: onChangeProvider, _col: 0 },
  };

  const nodes: Node[] = [anchor];
  const edges: Edge[] = [];

  const policies = agentPolicies.filter((p) => p.destination_provider_ids.includes(providerId));
  const sourceGroupIds = new Set<string>();

  for (const policy of policies) {
    nodes.push({
      id: `agent-policy-${policy.id}`,
      type: "agentPolicyNode",
      position: { x: 0, y: 0 },
      data: { policy, _col: 1 },
    });
    edges.push({
      id: `anchor-agent-policy-${policy.id}`,
      source: "anchor",
      target: `agent-policy-${policy.id}`,
      type: "in",
      data: { enabled: policy.enabled },
    });
    for (const gid of policy.source_groups) {
      sourceGroupIds.add(gid);
      edges.push({
        id: `agent-policy-${policy.id}-group-${gid}`,
        source: `agent-policy-${policy.id}`,
        target: `group-${gid}`,
        type: "in",
        data: { enabled: policy.enabled },
      });
    }
  }

  for (const gid of sourceGroupIds) {
    const group = ctx.groupsById.get(gid);
    if (!group) continue;
    nodes.push({ id: `group-${gid}`, type: "groupNode", position: { x: 0, y: 0 }, data: { group, enabled: true, _col: 2 } });
  }

  return withExpansion({ nodes, edges }, expandedGroupId, ctx);
}
