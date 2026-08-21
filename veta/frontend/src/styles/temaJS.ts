/**
 * recharts SVG içindeki bazı öznitelikler (stroke/fill gradient stopColor)
 * CSS custom property'leri her tarayıcıda güvenilir çözümlemez — bu yüzden
 * SADECE recharts'a geçirilen ham renkler için bu küçük JS aynası var.
 * Tek doğruluk kaynağı yine styles/tokens.css'tir; buradaki değerler onunla
 * BİREBİR aynı kalmalı (bkz. Performans.tsx kullanım noktası).
 */
export const TEMA_JS = {
  acik: {
    tur: "#2E9CAD",
    turDeep: "#1C7A88",
    sub: "#7C8A89",
    panel: "#FFFFFF",
    line: "rgba(46,156,173,0.14)",
  },
  koyu: {
    tur: "#43B8C6",
    turDeep: "#79D6E0",
    sub: "#7E9695",
    panel: "#12201F",
    line: "rgba(67,184,198,0.16)",
  },
} as const;
