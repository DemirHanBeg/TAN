import { ChevronRight } from "lucide-react";
import type { HizliIslemVerisi } from "../../data/turler";
import { IKON_HARITASI } from "../ui/ikonEsle";

type Props = {
  islemler: HizliIslemVerisi[];
};

/**
 * Hızlı işlem butonları veri iddiası taşımıyor (birer navigasyon/eylem
 * kısayolu) — bu yüzden KONSEPT rozeti gerekmiyor. Ama arkalarında henüz
 * gerçek bir VETA sunucusu olmadığından tıklanamaz durumdalar; bunu
 * başlığın altındaki nota açıkça yazıyoruz (Potemkin buton istemiyoruz).
 */
export function HizliIslemler({ islemler }: Props) {
  return (
    <div className="veta-card veta-panel" style={{ padding: "16px 18px" }}>
      <div style={{ fontSize: 14.5, fontWeight: 700, marginBottom: 2 }}>Hızlı işlemler</div>
      <div style={{ fontSize: 11, color: "var(--veta-sub)", marginBottom: 8 }}>
        VETA arka ucu henüz yok — bu düğmeler devre dışı.
      </div>
      {islemler.map((islem) => {
        const Ik = IKON_HARITASI[islem.ikonAdi];
        return (
          <button
            key={islem.id}
            disabled
            title="VETA arka ucu henüz yok — TODO: bağla"
            className="veta-act"
            style={{
              width: "100%",
              display: "flex",
              alignItems: "center",
              gap: 12,
              padding: "11px 10px",
              borderRadius: "var(--veta-r-ic)",
              border: "none",
              background: "transparent",
              color: "var(--veta-ink)",
              fontSize: 14,
              cursor: "not-allowed",
              opacity: 0.72,
              transition: "background 200ms",
            }}
          >
            <span className="veta-ikon-rozet-kucuk">
              <Ik size={16} />
            </span>
            {islem.ad}
            <ChevronRight size={16} color="var(--veta-sub)" style={{ marginLeft: "auto" }} />
          </button>
        );
      })}
    </div>
  );
}
