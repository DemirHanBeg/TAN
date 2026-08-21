import { ShieldCheck } from "lucide-react";
import type { SistemDurumuVerisi } from "../../data/turler";
import { KonseptRozeti } from "../ui/KonseptRozeti";

type Props = {
  durum: SistemDurumuVerisi;
};

export function SistemDurumu({ durum }: Props) {
  return (
    <div className="veta-card veta-panel" style={{ padding: "18px 18px 20px", textAlign: "center" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, textAlign: "left", marginBottom: 12 }}>
        <div style={{ fontSize: 14.5, fontWeight: 700 }}>Sistem durumu</div>
        <KonseptRozeti />
      </div>
      <div
        style={{
          display: "grid",
          placeItems: "center",
          width: 62,
          height: 62,
          margin: "4px auto 0",
          borderRadius: 18,
          background: "var(--veta-tur-soft)",
          color: "var(--veta-tur)",
        }}
      >
        <ShieldCheck size={30} />
      </div>
      <div style={{ fontSize: 15, fontWeight: 700, color: "var(--veta-tur-deep)", marginTop: 12 }}>
        {durum.baslik.deger}
      </div>
      <div style={{ fontSize: 12.5, color: "var(--veta-sub)", marginTop: 4 }}>{durum.aciklama.deger}</div>
    </div>
  );
}
