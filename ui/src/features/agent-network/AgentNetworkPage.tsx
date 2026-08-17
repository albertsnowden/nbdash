import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import PoliciesTab from "./PoliciesTab";
import ProvidersTab from "./ProvidersTab";

export default function AgentNetworkPage() {
  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold text-nb-gray-50">Agent Network</h1>

      <Tabs defaultValue="providers">
        <TabsList>
          <TabsTrigger value="providers">Providers</TabsTrigger>
          <TabsTrigger value="policies">Policies</TabsTrigger>
        </TabsList>
        <TabsContent value="providers">
          <ProvidersTab />
        </TabsContent>
        <TabsContent value="policies">
          <PoliciesTab />
        </TabsContent>
      </Tabs>
    </section>
  );
}
