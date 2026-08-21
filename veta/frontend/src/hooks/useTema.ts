import { useCallback, useEffect, useState } from "react";

export type Tema = "acik" | "koyu";

const DEPOLAMA_ANAHTARI = "veta-tema";

function sistemTercihi(): Tema {
  if (typeof window === "undefined") return "acik";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "koyu" : "acik";
}

function kayitliTema(): Tema | null {
  try {
    const deger = localStorage.getItem(DEPOLAMA_ANAHTARI);
    return deger === "acik" || deger === "koyu" ? deger : null;
  } catch {
    return null;
  }
}

/**
 * Karanlık mod durumu: kullanıcı elle seçtiyse localStorage'dan okunur,
 * seçmediyse sistem tercihine düşer. `data-tema` özniteliğini <html>
 * üzerine yazar — tüm token'lar bundan besleniyor (bkz. styles/tokens.css).
 */
export function useTema(): { tema: Tema; koyuMu: boolean; temaDegistir: () => void } {
  const [tema, setTema] = useState<Tema>(() => kayitliTema() ?? sistemTercihi());

  useEffect(() => {
    document.documentElement.setAttribute("data-tema", tema);
  }, [tema]);

  const temaDegistir = useCallback(() => {
    setTema((onceki) => {
      const yeni: Tema = onceki === "koyu" ? "acik" : "koyu";
      try {
        localStorage.setItem(DEPOLAMA_ANAHTARI, yeni);
      } catch {
        // localStorage kapalıysa (gizli sekme vb.) sessizce devam et
      }
      return yeni;
    });
  }, []);

  return { tema, koyuMu: tema === "koyu", temaDegistir };
}
