import { useContainerLogStore } from "@/stores/docker-store";
import {
  ContainerDetailHeader,
  CopyIcon,
  KV,
  Mono,
} from "./container-detail-components";

export function NetworkTabContent({
  handleCopy,
  copiedKey,
}: {
  handleCopy: (text: string, key: string) => void;
  copiedKey: string | null;
}) {
  const selectedContainer = useContainerLogStore.get("selectedContainer");

  return (
    <div>
      <div className="p-4">
        <ContainerDetailHeader
          title="Port Mappings"
          description="Published container ports"
        />

        <div className="space-y-2 text-sm">
          {selectedContainer?.Ports?.length ? (
            <div className="space-y-2">
              {selectedContainer?.Ports.map((p, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between text-sm rounded-md border p-2"
                >
                  <span>
                    {p.PrivatePort}/{p.Type}
                  </span>
                  {p.PublicPort ? (
                    <div className="flex items-center gap-2">
                      <span className="text-muted-foreground">→</span>
                      <Mono>{`${p.IP || "0.0.0.0"}:${p.PublicPort}`}</Mono>
                      <CopyIcon
                        ariaLabel="Copy mapping"
                        onClick={() =>
                          handleCopy(
                            `${p.IP || "0.0.0.0"}:${p.PublicPort}`,
                            `port-${idx}`,
                          )
                        }
                        active={copiedKey === `port-${idx}`}
                      />
                    </div>
                  ) : (
                    <span className="text-muted-foreground">not published</span>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">No port mappings</p>
          )}
        </div>
      </div>
      <div className="p-4">
        <ContainerDetailHeader
          title="Networks"
          description="Connected Docker networks"
        />
        <div className="space-y-2">
          {Object.entries(
            selectedContainer?.NetworkSettings?.Networks || {},
          ).map(([name, network]: [string, any]) => (
            <div key={name} className="text-sm rounded-md border p-2 space-y-1">
              <div className="font-semibold">{name}</div>
              {network.IPAddress && (
                <KV label="IP Address">
                  <Mono>{network.IPAddress}</Mono>
                  <CopyIcon
                    ariaLabel="Copy IP"
                    onClick={() => handleCopy(network.IPAddress, `ip-${name}`)}
                    active={copiedKey === `ip-${name}`}
                  />
                </KV>
              )}
              {network.MacAddress && (
                <KV label="MAC">
                  <Mono>{network.MacAddress}</Mono>
                </KV>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
