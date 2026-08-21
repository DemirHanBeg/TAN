import { X } from "lucide-react";
import { KaplumbagaLogo } from "../logo/KaplumbagaLogo";
import { NAV_OGELERI } from "./navOgeleri";

type Props = {
  aktifId: string;
  onSec: (id: string) => void;
  drawer?: boolean;
  onKapat?: () => void;
};

export function Sidebar({ aktifId, onSec, drawer, onKapat }: Props) {
  return (
    <div
      className={drawer ? undefined : "veta-side-static"}
      style={{
        background: "var(--veta-panel)",
        borderRight: "1px solid var(--veta-line)",
        display: "flex",
        flexDirection: "column",
        padding: "18px 14px",
        height: drawer ? "100%" : undefined,
        transition: "background-color var(--veta-sure-normal)",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 11, padding: "4px 8px 18px" }}>
        <KaplumbagaLogo />
        <div>
          <div style={{ fontSize: 20, fontWeight: 700, letterSpacing: "-0.02em", color: "var(--veta-ink)" }}>
            VETA
          </div>
          <div style={{ fontSize: 9, letterSpacing: "0.22em", color: "var(--veta-sub)", marginTop: -1 }}>
            VERİTABANI SİSTEMİ
          </div>
        </div>
        {drawer && (
          <button
            onClick={onKapat}
            aria-label="Menüyü kapat"
            style={{ marginLeft: "auto", background: "none", border: "none", color: "var(--veta-sub)", cursor: "pointer" }}
          >
            <X size={20} />
          </button>
        )}
      </div>

      <nav style={{ display: "flex", flexDirection: "column", gap: 2, overflowY: "auto" }}>
        {NAV_OGELERI.map(({ id, ad, Ikon }) => {
          const aktif = id === aktifId;
          return (
            <button
              key={id}
              className="veta-nav"
              onClick={() => onSec(id)}
              aria-current={aktif ? "page" : undefined}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 12,
                padding: "10px 12px",
                borderRadius: "var(--veta-r-ic)",
                border: "none",
                cursor: "pointer",
                textAlign: "left",
                background: aktif ? "var(--veta-tur-soft)" : "transparent",
                color: aktif ? "var(--veta-tur-deep)" : "var(--veta-sub)",
                fontSize: 14.5,
                fontWeight: aktif ? 600 : 500,
                position: "relative",
                transition: "background 200ms, color 200ms",
              }}
            >
              {aktif && (
                <span
                  style={{
                    position: "absolute",
                    left: 0,
                    top: 9,
                    bottom: 9,
                    width: 3,
                    borderRadius: 3,
                    background: "var(--veta-tur)",
                  }}
                />
              )}
              <Ikon size={18} strokeWidth={aktif ? 2.2 : 1.8} />
              {ad}
            </button>
          );
        })}
      </nav>

      <div
        style={{
          marginTop: "auto",
          display: "flex",
          alignItems: "center",
          gap: 10,
          padding: "12px 8px 2px",
          borderTop: "1px solid var(--veta-line)",
        }}
      >
        <div
          style={{
            width: 32,
            height: 32,
            borderRadius: "50%",
            background: "var(--veta-tur)",
            color: "#fff",
            display: "grid",
            placeItems: "center",
            fontSize: 13,
            fontWeight: 700,
          }}
        >
          Y
        </div>
        <div>
          <div style={{ fontSize: 13.5, fontWeight: 600, color: "var(--veta-ink)" }}>Yönetici</div>
          <div style={{ fontSize: 11, color: "var(--veta-sub)" }}>admin</div>
        </div>
      </div>
    </div>
  );
}
