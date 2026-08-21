type Props = {
  oranYuzde: number;
  gir: boolean;
  gecikmeSaniye?: number;
  yukseklik?: number;
};

/** Dolum çubuğu — giriş animasyonu tetiklendiğinde 0'dan orana genişler. */
export function Ilerleme({ oranYuzde, gir, gecikmeSaniye = 0.2, yukseklik = 7 }: Props) {
  return (
    <div
      style={{
        height: yukseklik,
        background: "var(--veta-raf)",
        borderRadius: 5,
        overflow: "hidden",
      }}
    >
      <div
        style={{
          width: "100%",
          height: "100%",
          background: "linear-gradient(90deg, var(--veta-tur), var(--veta-tur-deep))",
          borderRadius: 5,
          transform: `scaleX(${gir ? oranYuzde / 100 : 0})`,
          transformOrigin: "left",
          transition: `transform 800ms var(--veta-ease) ${gecikmeSaniye}s`,
        }}
      />
    </div>
  );
}
