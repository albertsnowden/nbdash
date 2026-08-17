import type { ReactNode } from "react";
import type { IdentityProviderType } from "@/api/identityProviders";

// Setup guidance shown next to the create/edit form, keyed by provider
// type. Verified against the NetBird management server's own source
// (idp/dex/connector.go, management/server/types/identity_provider.go) —
// every configured provider becomes a connector on NetBird's embedded Dex
// instance, and every connector shares one fixed redirect URI derived from
// the Management server's own OIDC issuer (idp/dex/connector.go's
// GetRedirectURI: "<issuer>/oauth2/callback" when the issuer doesn't
// already end in "/oauth2"). That URI is NOT this dashboard's own origin —
// it's wherever your NetBird Management server itself is publicly
// reachable, which is usually but not always the same domain.
export const REDIRECT_URI_HINT = "https://<your-netbird-management-domain>/oauth2/callback";

const GENERIC_OIDC_STEPS = (
  <>
    <li>Register a new OAuth2/OIDC application in your identity provider's admin console.</li>
    <li>
      Set its redirect/callback URI to <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">{REDIRECT_URI_HINT}</code>.
    </li>
    <li>Copy the app's issuer URL into the Issuer field above.</li>
    <li>Copy its Client ID and Client Secret into this form.</li>
  </>
);

export const IDENTITY_PROVIDER_HELP: Record<IdentityProviderType, { summary: string; steps: ReactNode }> = {
  google: {
    summary: "Uses Google's fixed OIDC endpoint — no Issuer needed.",
    steps: (
      <>
        <li>
          In <span className="text-nb-gray-300">Google Cloud Console</span>, go to APIs & Services → Credentials.
        </li>
        <li>Create Credentials → OAuth client ID → Application type: Web application.</li>
        <li>
          Under Authorized redirect URIs, add{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">{REDIRECT_URI_HINT}</code>.
        </li>
        <li>Copy the generated Client ID and Client Secret into this form. Leave Issuer blank.</li>
      </>
    ),
  },
  microsoft: {
    summary: "Uses Microsoft's fixed endpoint (tenant \"common\") — no Issuer needed.",
    steps: (
      <>
        <li>In the Azure Portal, go to Microsoft Entra ID → App registrations → New registration.</li>
        <li>
          Add a Web redirect URI:{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">{REDIRECT_URI_HINT}</code>.
        </li>
        <li>Create a client secret under Certificates & secrets.</li>
        <li>Copy the Application (client) ID and the secret's value into this form. Leave Issuer blank.</li>
        <li className="text-nb-gray-500">
          Note: this connector requests only the <code className="font-mono">user.read</code> scope by
          default — narrower than the other provider types.
        </li>
      </>
    ),
  },
  entra: {
    summary: "Tenant-specific — needs your own Issuer URL, unlike \"Microsoft\" above.",
    steps: (
      <>
        <li>In the Azure Portal, register an app under Microsoft Entra ID → App registrations.</li>
        <li>
          Add a Web redirect URI:{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">{REDIRECT_URI_HINT}</code>.
        </li>
        <li>
          Set Issuer to your tenant's endpoint, e.g.{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">
            https://login.microsoftonline.com/&lt;tenant-id&gt;/v2.0
          </code>
          .
        </li>
        <li>Copy the Application (client) ID and a client secret into this form.</li>
      </>
    ),
  },
  okta: {
    summary: "Needs your Okta domain's Issuer URL.",
    steps: (
      <>
        <li>In your Okta admin console, create an OIDC app integration (Web application).</li>
        <li>
          Set its sign-in redirect URI to{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">{REDIRECT_URI_HINT}</code>.
        </li>
        <li>
          Set Issuer to your Okta domain, e.g.{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">
            https://&lt;your-org&gt;.okta.com
          </code>
          .
        </li>
        <li>Copy the Client ID and Client Secret into this form.</li>
      </>
    ),
  },
  zitadel: { summary: "Generic OIDC — needs your Zitadel instance's Issuer URL.", steps: GENERIC_OIDC_STEPS },
  pocketid: { summary: "Generic OIDC — needs your PocketID instance's Issuer URL.", steps: GENERIC_OIDC_STEPS },
  authentik: { summary: "Generic OIDC — needs your Authentik instance's Issuer URL.", steps: GENERIC_OIDC_STEPS },
  keycloak: {
    summary: "Generic OIDC — Issuer is your realm's discovery endpoint.",
    steps: (
      <>
        <li>In Keycloak, create a client under your realm (Clients → Create client, OpenID Connect).</li>
        <li>
          Set its valid redirect URI to{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">{REDIRECT_URI_HINT}</code>.
        </li>
        <li>
          Set Issuer to{" "}
          <code className="rounded bg-nb-gray-900 px-1 py-0.5 font-mono text-[11px]">
            https://&lt;keycloak-host&gt;/realms/&lt;realm&gt;
          </code>
          .
        </li>
        <li>Copy the client's ID and secret into this form.</li>
      </>
    ),
  },
  adfs: { summary: "Generic OIDC — needs your AD FS Issuer URL.", steps: GENERIC_OIDC_STEPS },
  oidc: { summary: "Any standards-compliant OIDC provider not listed above.", steps: GENERIC_OIDC_STEPS },
};
