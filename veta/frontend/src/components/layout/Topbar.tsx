import { Bell, ChevronDown, Menu, Moon, Search, Sun } from "lucide-react";

type Props = {
  darMi: boolean;
  koyuMu: boolean;
  onMenuAc: () => void;
  onTemaDegistir: () => void;
};

export function Topbar({ darMi, koyuMu, onMenuAc, onTemaDegistir }: Props) {
  return (
    <div
      style={{
        position: "sticky",
        top: 0,
        zIndex: 20,
        background: "var(--veta-chrome)",
        backdropFilter: "blur(16px) saturate(160%)",
        WebkitBackdropFilter: "blur(16px) saturate(160%)",
        borderBottom: "1px solid var(--veta-line)",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 12, padding: "12px 20px", maxWidth: 1180, margin: "0 auto" }}>
        {darMi && (
          <button
            onClick={onMenuAc}
            aria-label="Menüyü aç"
            style={{
              display: "grid",
              placeItems: "center",
              background: "none",
              border: "1px solid var(--veta-line)",
              borderRadius: 10,
              width: 38,
              height: 38,
              color: "var(--veta-sub)",
              cursor: "pointer",
            }}
          >
            <Menu size={18} />
          </button>
        )}

        <div
          style={{
            flex: 1,
            maxWidth: 460,
            display: "flex",
            alignItems: "center",
            gap: 9,
            background: "var(--veta-panel)",
            border: "1px solid var(--veta-line)",
            borderRadius: 12,
            padding: "9px 13px",
          }}
        >
          <Search size={17} color="var(--veta-sub)" />
          <input
            placeholder="Ara…"
            style={{ border: "none", background: "transparent", color: "var(--veta-ink)", fontSize: 14, width: "100%" }}
          />
        </div>

        <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 6 }}>
          <button
            onClick={onTemaDegistir}
            aria-label="Temayı değiştir"
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--veta-sub)",
              padding: 8,
              borderRadius: 10,
              display: "grid",
              placeItems: "center",
            }}
          >
            {koyuMu ? <Sun size={19} /> : <Moon size={19} />}
          </button>
          <button
            style={{ background: "none", border: "none", cursor: "pointer", color: "var(--veta-sub)", padding: 8, position: "relative" }}
            aria-label="Bildirimler"
          >
            <Bell size={19} />
            <span
              style={{
                position: "absolute",
                top: 7,
                right: 8,
                width: 7,
                height: 7,
                background: "var(--veta-tur)",
                borderRadius: "50%",
                border: "2px solid var(--veta-chrome)",
              }}
            />
          </button>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 9,
              background: "var(--veta-panel)",
              border: "1px solid var(--veta-line)",
              borderRadius: 30,
              padding: "5px 10px 5px 6px",
            }}
          >
            <div
              style={{
                width: 28,
                height: 28,
                borderRadius: "50%",
                background: "var(--veta-tur)",
                color: "#fff",
                display: "grid",
                placeItems: "center",
                fontSize: 12,
                fontWeight: 700,
              }}
            >
              Y
            </div>
            <div style={{ lineHeight: 1.1 }}>
              <div style={{ fontSize: 12.5, fontWeight: 600 }}>Yönetici</div>
              <div style={{ fontSize: 10.5, color: "var(--veta-sub)" }}>admin</div>
            </div>
            <ChevronDown size={15} color="var(--veta-sub)" />
          </div>
        </div>
      </div>
    </div>
  );
}
