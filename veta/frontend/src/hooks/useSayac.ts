import { useEffect, useState } from "react";

const VARSAYILAN_SURE_MS = 950;

/**
 * Apple-tarzı count-up: cubic ease-out ile 0'dan hedefe sayar.
 * `azaltilmisHareket` true ise (prefers-reduced-motion) anında hedefe atlar.
 */
export function useSayac(
  hedef: number,
  calis: boolean,
  azaltilmisHareket: boolean,
  sureMs: number = VARSAYILAN_SURE_MS,
): number {
  const [deger, setDeger] = useState(azaltilmisHareket ? hedef : 0);

  useEffect(() => {
    if (!calis || azaltilmisHareket) {
      setDeger(hedef);
      return;
    }

    let rafId: number;
    let baslangic: number | undefined;

    const adim = (zaman: number) => {
      if (baslangic === undefined) baslangic = zaman;
      const ilerleme = Math.min(1, (zaman - baslangic) / sureMs);
      const easeOut = 1 - Math.pow(1 - ilerleme, 3);
      setDeger(hedef * easeOut);
      if (ilerleme < 1) rafId = requestAnimationFrame(adim);
    };

    rafId = requestAnimationFrame(adim);
    return () => cancelAnimationFrame(rafId);
  }, [hedef, calis, azaltilmisHareket, sureMs]);

  return deger;
}
