import { GitBranch, GitCommitHorizontal, FolderGit2, Link2 } from "lucide-react";
import type { DepoBaglantisi } from "../../data/turler";

type Props = {
  baglantilar: DepoBaglantisi[];
};

const IKONLAR: Record<string, typeof FolderGit2> = {
  Depo: FolderGit2,
  Dal: GitBranch,
  "Son Commit": GitCommitHorizontal,
  Uzak: Link2,
};

/**
 * Referans dosyadaki sahte "Sunucu/Port/Sürüm" listesi yerine gerçek depo
 * bilgisi — bkz. src/data/veriKaynagi.ts:baglantilarOlustur. Canlı bir VETA
 * sunucusu olmadığından "Durum: Bağlı" gibi bir iddia burada YOK.
 */
export function Baglantilar({ baglantilar }: Props) {
  return (
    <div className="veta-card veta-panel" style={{ padding: "16px 18px" }}>
      <div style={{ fontSize: 14.5, fontWeight: 700, marginBottom: 8 }}>Depo bağlantısı</div>
      {baglantilar.map((b, i) => {
        const Ik = IKONLAR[b.ad] ?? Link2;
        return (
          <div
            key={b.ad}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 11,
              padding: "10px 0",
              borderTop: i ? "1px solid var(--veta-line)" : "none",
            }}
          >
            <Ik size={16} color="var(--veta-sub)" />
            <span style={{ fontSize: 13.5, color: "var(--veta-sub)" }}>{b.ad}</span>
            <span
              title={b.deger.bagli ? b.deger.kaynak : undefined}
              style={{
                marginLeft: "auto",
                fontSize: 13,
                fontWeight: 600,
                fontFamily: "ui-monospace, monospace",
                textAlign: "right",
                maxWidth: 170,
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
            >
              {b.deger.deger}
            </span>
          </div>
        );
      })}
    </div>
  );
}
