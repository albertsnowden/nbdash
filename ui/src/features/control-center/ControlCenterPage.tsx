import {
  Background,
  Controls,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge,
  type EdgeTypes,
  type Node,
  type NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Bot, FolderGit2, MonitorSmartphone, Network as NetworkIcon, Users as UsersIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useAgentNetworkPolicies, useAgentNetworkProviders } from "@/api/agentNetwork";
import { PROTECTED_GROUP_NAME, useGroups } from "@/api/groups";
import { useNetworks, useResources } from "@/api/networks";
import { usePeers } from "@/api/peers";
import { usePolicies } from "@/api/policies";
import { useUsers } from "@/api/team";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import AgentPolicyNode from "./nodes/AgentPolicyNode";
import DirectionIn from "./edges/DirectionIn";
import SimpleConnection from "./edges/SimpleConnection";
import GroupNode from "./nodes/GroupNode";
import PeerNode from "./nodes/PeerNode";
import PolicyNode from "./nodes/PolicyNode";
import ResourceNode from "./nodes/ResourceNode";
import SelectGroupNode from "./nodes/SelectGroupNode";
import SelectNetworkNode from "./nodes/SelectNetworkNode";
import SelectPeerNode from "./nodes/SelectPeerNode";
import SelectProviderNode from "./nodes/SelectProviderNode";
import SelectUserNode from "./nodes/SelectUserNode";
import { buildAgentNetworkView, buildGroupView, buildNetworkView, buildPeerView, buildUserView, type ViewContext } from "./views";

const NODE_TYPES: NodeTypes = {
  groupNode: GroupNode,
  peerNode: PeerNode,
  policyNode: PolicyNode,
  resourceNode: ResourceNode,
  agentPolicyNode: AgentPolicyNode,
  selectGroupNode: SelectGroupNode,
  selectPeerNode: SelectPeerNode,
  selectUserNode: SelectUserNode,
  selectNetworkNode: SelectNetworkNode,
  selectProviderNode: SelectProviderNode,
} as unknown as NodeTypes;

const EDGE_TYPES: EdgeTypes = {
  in: DirectionIn,
  simple: SimpleConnection,
} as unknown as EdgeTypes;

type ViewKind = "groups" | "peers" | "users" | "networks" | "agentNetwork";

export default function ControlCenterPage() {
  return (
    <ReactFlowProvider>
      <ControlCenterView />
    </ReactFlowProvider>
  );
}

function ControlCenterView() {
  const { canReadAny } = usePermissionsContext();
  const canReadNetworks = canReadAny([MODULE.networks]);
  const canReadAgentNetwork = canReadAny([MODULE.agentNetworkProviders, MODULE.agentNetworkPolicies]);

  const { data: groupsData, isLoading: groupsLoading } = useGroups();
  const { data: policiesData, isLoading: policiesLoading } = usePolicies();
  const { data: peersData, isLoading: peersLoading } = usePeers("");
  const { data: usersData, isLoading: usersLoading } = useUsers();
  const { data: networksData, isLoading: networksDataLoading } = useNetworks(canReadNetworks);
  const { data: agentProvidersData, isLoading: agentProvidersLoading } = useAgentNetworkProviders(canReadAgentNetwork);
  const { data: agentPoliciesData, isLoading: agentPoliciesLoading } = useAgentNetworkPolicies(canReadAgentNetwork);
  const reactFlow = useReactFlow();

  const isLoading = groupsLoading || policiesLoading || peersLoading || usersLoading;
  const networksLoading = canReadNetworks && networksDataLoading;
  const agentNetworkLoading = canReadAgentNetwork && (agentProvidersLoading || agentPoliciesLoading);

  const groups = groupsData?.groups ?? [];
  const policies = policiesData?.policies ?? [];
  const peers = peersData?.peers ?? [];
  const users = useMemo(() => (usersData?.users ?? []).filter((u) => !u.is_service_user), [usersData]);
  const networks = networksData?.networks ?? [];
  const agentProviders = agentProvidersData?.providers ?? [];
  const agentPolicies = agentPoliciesData?.policies ?? [];

  const ctx: ViewContext = useMemo(
    () => ({
      policies,
      groupsById: new Map(groups.map((g) => [g.id, g])),
      peersById: new Map(peers.map((p) => [p.id, p])),
    }),
    [policies, groups, peers],
  );

  const [view, setView] = useState<ViewKind>("groups");
  const [selectedGroupId, setSelectedGroupId] = useState("");
  const [selectedPeerId, setSelectedPeerId] = useState("");
  const [selectedUserId, setSelectedUserId] = useState("");
  const [selectedNetworkId, setSelectedNetworkId] = useState("");
  const [selectedProviderId, setSelectedProviderId] = useState("");
  const [expandedGroupId, setExpandedGroupId] = useState<string | null>(null);

  const { data: resourcesData } = useResources(selectedNetworkId);
  const resources = useMemo(() => resourcesData?.resources ?? [], [resourcesData]);

  // Pick a sensible default per view once its data has loaded: for Groups,
  // the largest group that's actually a policy source (falling back to the
  // largest non-"All" group) — an empty graph on first load would look
  // broken even though nothing is actually wrong.
  useEffect(() => {
    if (isLoading) return;
    if (selectedGroupId === "" && groups.length > 0) {
      const nonAll = groups.filter((g) => g.name !== PROTECTED_GROUP_NAME);
      const withPolicy = nonAll.find((g) =>
        policies.some((p) => p.rules.some((r) => (r.sources ?? []).some((s) => s.id === g.id))),
      );
      setSelectedGroupId((withPolicy ?? nonAll[0] ?? groups[0]).id);
    }
    if (selectedPeerId === "" && peers.length > 0) {
      setSelectedPeerId(peers[0].id);
    }
    if (selectedUserId === "" && users.length > 0) {
      const withPeer = users.find((u) => peers.some((p) => p.user_id === u.id));
      setSelectedUserId((withPeer ?? users[0]).id);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading]);

  useEffect(() => {
    if (networksLoading) return;
    if (selectedNetworkId === "" && networks.length > 0) {
      setSelectedNetworkId(networks[0].id);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [networksLoading]);

  useEffect(() => {
    if (agentNetworkLoading) return;
    if (selectedProviderId === "" && agentProviders.length > 0) {
      setSelectedProviderId(agentProviders[0].id);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentNetworkLoading]);

  useEffect(() => {
    setExpandedGroupId(null);
  }, [view, selectedGroupId, selectedPeerId, selectedUserId, selectedNetworkId, selectedProviderId]);

  const onChangeGroup = (id: string) => {
    setSelectedGroupId(id);
    setExpandedGroupId(null);
  };
  const onToggleGroup = (id: string) => {
    setExpandedGroupId((prev) => (prev === id ? null : id));
  };

  const { nodes, edges } = useMemo<{ nodes: Node[]; edges: Edge[] }>(() => {
    const withGroupClicks = (result: { nodes: Node[]; edges: Edge[] }) => ({
      nodes: result.nodes.map((n) =>
        n.type === "groupNode"
          ? {
              ...n,
              data: {
                ...n.data,
                expanded: n.id === `group-${expandedGroupId}`,
                onClick: () => onToggleGroup(n.id.replace("group-", "")),
              },
            }
          : n,
      ),
      edges: result.edges,
    });

    if (view === "groups" && !isLoading && selectedGroupId) {
      return withGroupClicks(buildGroupView(selectedGroupId, ctx, groups, expandedGroupId, onChangeGroup));
    }
    if (view === "peers" && !isLoading && selectedPeerId) {
      return withGroupClicks(
        buildPeerView(selectedPeerId, ctx, peers, expandedGroupId, (id) => {
          setSelectedPeerId(id);
          setExpandedGroupId(null);
        }),
      );
    }
    if (view === "users" && !isLoading && selectedUserId) {
      return withGroupClicks(
        buildUserView(selectedUserId, ctx, users, peers, expandedGroupId, (id) => {
          setSelectedUserId(id);
          setExpandedGroupId(null);
        }),
      );
    }
    if (view === "networks" && !isLoading && !networksLoading && selectedNetworkId) {
      const network = networks.find((n) => n.id === selectedNetworkId);
      return withGroupClicks(
        buildNetworkView(selectedNetworkId, network, resources, ctx, networks, expandedGroupId, (id) => {
          setSelectedNetworkId(id);
          setExpandedGroupId(null);
        }),
      );
    }
    if (view === "agentNetwork" && !isLoading && !agentNetworkLoading && selectedProviderId) {
      return withGroupClicks(
        buildAgentNetworkView(selectedProviderId, agentProviders, agentPolicies, ctx, expandedGroupId, (id) => {
          setSelectedProviderId(id);
          setExpandedGroupId(null);
        }),
      );
    }
    return { nodes: [], edges: [] };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    view,
    selectedGroupId,
    selectedPeerId,
    selectedUserId,
    selectedNetworkId,
    selectedProviderId,
    expandedGroupId,
    ctx,
    groups,
    peers,
    users,
    networks,
    resources,
    agentProviders,
    agentPolicies,
    isLoading,
    networksLoading,
    agentNetworkLoading,
  ]);

  useEffect(() => {
    if (nodes.length === 0) return;
    window.requestAnimationFrame(() => reactFlow.fitView({ padding: 0.2, duration: 400, maxZoom: 1 }));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view, selectedGroupId, selectedPeerId, selectedUserId, selectedNetworkId, selectedProviderId, expandedGroupId]);

  const viewLoading =
    isLoading || (view === "networks" ? networksLoading : view === "agentNetwork" ? agentNetworkLoading : false);

  const emptyMessage =
    view === "groups" && groups.length === 0
      ? "No groups yet — create one to see it here."
      : view === "peers" && peers.length === 0
        ? "No peers connected yet."
        : view === "users" && users.length === 0
          ? "No users yet."
          : view === "networks" && networks.length === 0
            ? "No networks yet — create one to see it here."
            : view === "agentNetwork" && agentProviders.length === 0
              ? "No AI providers connected yet."
              : null;

  return (
    <section className="flex h-[calc(100vh-8rem)] flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">Control Center</h1>
          <p className="text-sm text-nb-gray-500">
            Pick a group, peer, user, network, or AI provider to see what it can reach.
          </p>
        </div>
        <Tabs value={view} onValueChange={(v) => setView(v as ViewKind)}>
          <TabsList>
            <TabsTrigger value="peers" className="inline-flex items-center gap-1.5">
              <MonitorSmartphone size={14} /> Peer
            </TabsTrigger>
            <TabsTrigger value="users" className="inline-flex items-center gap-1.5">
              <UsersIcon size={14} /> User
            </TabsTrigger>
            <TabsTrigger value="groups" className="inline-flex items-center gap-1.5">
              <FolderGit2 size={14} /> Group
            </TabsTrigger>
            {canReadNetworks && (
              <TabsTrigger value="networks" className="inline-flex items-center gap-1.5">
                <NetworkIcon size={14} /> Network
              </TabsTrigger>
            )}
            {canReadAgentNetwork && (
              <TabsTrigger value="agentNetwork" className="inline-flex items-center gap-1.5">
                <Bot size={14} /> Agent Network
              </TabsTrigger>
            )}
          </TabsList>
        </Tabs>
      </div>

      <div className="flex-1 overflow-hidden rounded-lg border border-nb-gray-900 bg-nb-gray-950">
        {viewLoading ? (
          <div className="flex h-full items-center justify-center text-sm text-nb-gray-500">Loading…</div>
        ) : emptyMessage ? (
          <div className="flex h-full items-center justify-center text-sm text-nb-gray-500">{emptyMessage}</div>
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={NODE_TYPES}
            edgeTypes={EDGE_TYPES}
            fitView
            colorMode="dark"
            proOptions={{ hideAttribution: true }}
            minZoom={0.2}
            maxZoom={1.6}
          >
            <Background color="#292b2f" gap={24} />
            <Controls />
          </ReactFlow>
        )}
      </div>
    </section>
  );
}
