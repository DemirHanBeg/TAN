import { useSayac } from "../../hooks/useSayac";
import type { StatKartVerisi } from "../../data/turler";
import { IKON_HARITASI } from "./ikonEsle";
import { Ilerleme } from "./Ilerleme";
import { KonseptRozeti } from "./KonseptRozeti";

type Props = {
  s: StatKartVerisi;
  gir: boolean;
  azaltilmisHareket: boolean;
};

export function StatKart({ s, gir, azaltilmisHareket }: Props) {
  const v = useSayac(s.deger.deger, gir, azaltilmisHareket);
  const Ikon = IKON_HARITASI[s.ikonAdi];
  const gosterilecek = s.deger.deger >= 1000 ? Math.round(v).toLocaleString("tr") : Math.round(v);
  const buKartBagliMi = s.ad.bagli && s.deger.bagli;

  return (
    <div className="veta-card veta-panel" style={{ padding: "18px 18px 16px" }}>
      <div style={{ display: "flex", alignItems: "flex-start", gap: 13 }}>
        <span className="veta-ikon-rozet">
          <Ikon size={21} />
        </span>
        <div style={{ minWidth: 0, flex: 1 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <div style={{ fontSize: 12.5, color: "var(--veta-sub)" }}>{s.ad.deger}</div>
            {!buKartBagliMi && <KonseptRozeti />}
          </div>
          <div
            style={{
              fontSize: 27,
              fontWeight: 700,
              letterSpacing: "-0.02em",
              marginTop: 2,
              fontVariantNumeric: "tabular-nums",
            }}
          >
            {gosterilecek}
            {s.birim}
          </div>
        </div>
      </div>
      {s.trend && (
        <div style={{ fontSize: 12.5, color: "var(--veta-tur-deep)", marginTop: 12, fontWeight: 500 }}>
          {s.trend.deger}
        </div>
      )}
      {s.oranYuzde && (
        <div style={{ marginTop: 12 }}>
          <Ilerleme oranYuzde={s.oranYuzde.deger} gir={gir} gecikmeSaniye={0.25} yukseklik={6} />
          <div style={{ fontSize: 12, color: "var(--veta-sub)", marginTop: 6, textAlign: "right" }}>
            %{s.oranYuzde.deger} kullanıldı
          </div>
        </div>
      )}
    </div>
  );
}
