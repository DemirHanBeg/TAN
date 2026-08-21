type Props = {
  renk?: string;
};

const CIZGI_SAYISI = 7;

/** Ambient ebru izi — sağ-üst köşede kısık, katmanlı bezier iz. Dominant değil. */
export function Ebru({ renk = "var(--veta-tur)" }: Props) {
  return (
    <svg
      width="360"
      height="300"
      viewBox="0 0 360 300"
      style={{ position: "absolute", top: -20, right: -10, opacity: 0.5, pointerEvents: "none" }}
      aria-hidden="true"
    >
      <g fill="none" stroke={renk} strokeWidth="1" strokeLinecap="round">
        {Array.from({ length: CIZGI_SAYISI }).map((_, i) => (
          <path
            key={i}
            d={`M${360 - i * 12} ${10 + i * 6} C ${240 - i * 14} ${40 + i * 10}, ${300 - i * 10} ${140 + i * 8}, ${140 - i * 12} ${200 + i * 6} S ${40 - i * 6} 280, 10 300`}
            opacity={0.5 - i * 0.05}
          />
        ))}
      </g>
    </svg>
  );
}
