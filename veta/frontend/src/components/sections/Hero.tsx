import { Kabuk } from "../watermark/Kabuk";
import { Ebru } from "../watermark/Ebru";

type Props = {
  koyuMu: boolean;
  girisStili: React.CSSProperties;
};

export function Hero({ koyuMu, girisStili }: Props) {
  return (
    <div
      style={{
        position: "relative",
        overflow: "hidden",
        borderRadius: 20,
        marginBottom: 18,
        padding: "6px 4px 14px",
        ...girisStili,
      }}
    >
      <Kabuk opaklik={koyuMu ? 0.09 : 0.06} />
      <Ebru />
      <div style={{ position: "relative" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
          <h1 style={{ fontSize: 30, fontWeight: 700, letterSpacing: "-0.025em", margin: 0 }}>
            Hoş geldiniz, <span style={{ color: "var(--veta-tur)" }}>Yönetici</span>
          </h1>
          <span
            style={{
              fontSize: 10.5,
              letterSpacing: "0.1em",
              color: "var(--veta-tur-deep)",
              background: "var(--veta-tur-soft)",
              borderRadius: "var(--veta-r-pill)",
              padding: "3px 10px",
              alignSelf: "center",
            }}
          >
            KONSEPT
          </span>
        </div>
        <p style={{ color: "var(--veta-sub)", fontSize: 15, marginTop: 8 }}>
          Veritabanı sisteminin genel durumuna buradan göz atabilirsiniz. Alttaki depo metrikleri gerçek — bkz. kart
          üstü rozetler.
        </p>
      </div>
    </div>
  );
}
