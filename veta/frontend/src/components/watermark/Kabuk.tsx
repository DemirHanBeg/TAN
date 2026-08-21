type Props = {
  renk?: string;
  opaklik?: number;
};

/**
 * Altıgen kabuk-scute tessellation — kaplumbağa kabuğu / çini karo / bellek
 * sayfası metaforunun subliminal watermark hali. YALNIZ hero zemininde,
 * çok düşük opaklıkta kullanılır — etiketlenmez, göze çarpmaz.
 */
export function Kabuk({ renk = "var(--veta-tur)", opaklik = 0.06 }: Props) {
  return (
    <svg
      width="100%"
      height="100%"
      style={{ position: "absolute", inset: 0, opacity: opaklik, pointerEvents: "none" }}
      aria-hidden="true"
    >
      <defs>
        <pattern id="veta-scute" width="56" height="48" patternUnits="userSpaceOnUse">
          <g fill="none" stroke={renk} strokeWidth="1.1">
            <path d="M14 0 L42 0 L56 24 L42 48 L14 48 L0 24 Z" />
          </g>
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill="url(#veta-scute)" />
    </svg>
  );
}
