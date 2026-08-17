import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import ClustersTab from "./ClustersTab";
import DomainsTab from "./DomainsTab";
import ProxyTokensTab from "./ProxyTokensTab";
import ServicesTab from "./ServicesTab";

export default function ReverseProxyPage() {
  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold text-nb-gray-50">Reverse Proxy</h1>

      <Tabs defaultValue="services">
        <TabsList>
          <TabsTrigger value="services">Services</TabsTrigger>
          <TabsTrigger value="domains">Custom Domains</TabsTrigger>
          <TabsTrigger value="proxy-tokens">Proxy Tokens</TabsTrigger>
          <TabsTrigger value="clusters">Clusters</TabsTrigger>
        </TabsList>
        <TabsContent value="services">
          <ServicesTab />
        </TabsContent>
        <TabsContent value="domains">
          <DomainsTab />
        </TabsContent>
        <TabsContent value="proxy-tokens">
          <ProxyTokensTab />
        </TabsContent>
        <TabsContent value="clusters">
          <ClustersTab />
        </TabsContent>
      </Tabs>
    </section>
  );
}
