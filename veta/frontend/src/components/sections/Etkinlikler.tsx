import { ChevronRight } from "lucide-react";
import type { EtkinlikVerisi } from "../../data/turler";
import { IKON_HARITASI } from "../ui/ikonEsle";
import { KonseptRozeti } from "../ui/KonseptRozeti";

type Props = {
  etkinlikler: EtkinlikVerisi[];
};

export function Etkinlikler({ etkinlikler }: Props) {
  return (
    <div className="veta-card veta-panel" style={{ padding: "18px 20px" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 4 }}>
        <div style={{ fontSize: 15.5, fontWeight: 700 }}>Son etkinlikler</div>
        <KonseptRozeti />
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr auto auto",
          gap: "0 16px",
          fontSize: 11.5,
          letterSpacing: "0.05em",
          textTransform: "uppercase",
          color: "var(--veta-sub)",
          padding: "10px 0 8px",
          borderBottom: "1px solid var(--veta-line)",
        }}
      >
        <span>Etkinlik</span>
        <span>Veritabanı</span>
        <span>Zaman</span>
      </div>
      {etkinlikler.map((e, i) => {
        const Ik = IKON_HARITASI[e.ikonAdi];
        return (
          <div
            key={e.id}
            className="veta-row"
            style={{
              display: "grid",
              gridTemplateColumns: "1fr auto auto",
              gap: "0 16px",
              alignItems: "center",
              padding: "12px 8px",
              margin: "0 -8px",
              borderRadius: 10,
              borderBottom: i < etkinlikler.length - 1 ? "1px solid var(--veta-line)" : "none",
              transition: "background 200ms",
              cursor: "pointer",
            }}
          >
            <span style={{ display: "flex", alignItems: "center", gap: 11, fontSize: 14 }}>
              <span className="veta-ikon-rozet-kucuk">
                <Ik size={15} />
              </span>
              {e.baslik}
            </span>
            <span style={{ fontSize: 13, color: "var(--veta-sub)", fontFamily: "ui-monospace, monospace" }}>
              {e.hedef}
            </span>
            <span style={{ fontSize: 13, color: "var(--veta-sub)", display: "flex", alignItems: "center", gap: 8 }}>
              {e.zaman}
              <ChevronRight size={15} />
            </span>
          </div>
        );
      })}
    </div>
  );
}
