type Props = {
  className?: string;
};

/**
 * Bağlanmamış (uydurma/örnek) her veri veya karta uygulanan görünür rozet.
 * Dürüstlük kısıtı: kaynağı olmayan hiçbir sayı "gerçek" gibi gösterilemez.
 */
export function KonseptRozeti({ className }: Props) {
  return (
    <span
      className={className}
      title="Bu veri henüz canlı bir kaynağa bağlı değil — tasarım vizyonunu göstermek için örnek."
      style={{
        fontSize: 10.5,
        letterSpacing: "0.1em",
        fontWeight: 600,
        color: "var(--veta-tur-deep)",
        background: "var(--veta-tur-soft)",
        borderRadius: "var(--veta-r-pill)",
        padding: "3px 10px",
        whiteSpace: "nowrap",
      }}
    >
      KONSEPT
    </span>
  );
}
