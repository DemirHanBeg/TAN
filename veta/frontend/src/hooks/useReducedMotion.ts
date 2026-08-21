import { useEffect, useState } from "react";

/**
 * `prefers-reduced-motion: reduce` tercihini izler. Değişince (kullanıcı
 * işletim sistemi ayarını değiştirirse) canlı günceller.
 */
export function useReducedMotion(): boolean {
  const [azaltilmis, setAzaltilmis] = useState(false);

  useEffect(() => {
    const sorgu = window.matchMedia("(prefers-reduced-motion: reduce)");
    setAzaltilmis(sorgu.matches);

    const dinle = (olay: MediaQueryListEvent) => setAzaltilmis(olay.matches);
    sorgu.addEventListener("change", dinle);
    return () => sorgu.removeEventListener("change", dinle);
  }, []);

  return azaltilmis;
}
