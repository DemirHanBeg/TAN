import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { KaynakKullanimVerisi, PerformansNoktasi } from "../../data/turler";
import { TEMA_JS } from "../../styles/temaJS";
import { Ilerleme } from "../ui/Ilerleme";
import { KonseptRozeti } from "../ui/KonseptRozeti";

type Props = {
  veri: PerformansNoktasi[];
  kaynakKullanimi: KaynakKullanimVerisi[];
  koyuMu: boolean;
  gir: boolean;
  azaltilmisHareket: boolean;
};

export function Performans({ veri, kaynakKullanimi, koyuMu, gir, azaltilmisHareket }: Props) {
  const t = koyuMu ? TEMA_JS.koyu : TEMA_JS.acik;

  return (
    <div className="veta-card veta-panel" style={{ padding: "18px 20px", marginTop: 14 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 12 }}>
        <div style={{ fontSize: 15.5, fontWeight: 700 }}>Sistem performansı</div>
        <KonseptRozeti />
      </div>
      <div style={{ display: "flex", gap: 20, flexWrap: "wrap" }}>
        <div style={{ flex: "1 1 300px", minWidth: 260, height: 190 }}>
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={veri} margin={{ top: 6, right: 6, left: -18, bottom: 0 }}>
              <defs>
                <linearGradient id="veta-perf-gradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={t.tur} stopOpacity={0.35} />
                  <stop offset="100%" stopColor={t.tur} stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="t" tick={{ fontSize: 10, fill: t.sub }} axisLine={false} tickLine={false} interval={2} />
              <YAxis
                tick={{ fontSize: 10, fill: t.sub }}
                axisLine={false}
                tickLine={false}
                domain={[0, 100]}
                tickFormatter={(v) => `%${v}`}
              />
              <Tooltip
                contentStyle={{ background: t.panel, border: `1px solid ${t.line}`, borderRadius: 10, fontSize: 12 }}
                labelStyle={{ color: t.sub }}
                formatter={(v) => [`%${v}`, "yük"]}
              />
              <Area
                type="monotone"
                dataKey="v"
                stroke={t.tur}
                strokeWidth={2.4}
                fill="url(#veta-perf-gradient)"
                isAnimationActive={!azaltilmisHareket}
                animationDuration={900}
                dot={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
        <div style={{ flex: "1 1 200px", minWidth: 190, display: "flex", flexDirection: "column", justifyContent: "center", gap: 16 }}>
          {kaynakKullanimi.map((k) => (
            <div key={k.ad}>
              <div style={{ display: "flex", justifyContent: "space-between", fontSize: 13, marginBottom: 6 }}>
                <span style={{ color: "var(--veta-sub)" }}>{k.ad}</span>
                <span style={{ fontWeight: 600, fontVariantNumeric: "tabular-nums" }}>{k.oran.deger}%</span>
              </div>
              <Ilerleme oranYuzde={k.oran.deger} gir={gir} />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
