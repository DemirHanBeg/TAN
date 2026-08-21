import { useEffect, useState, type CSSProperties } from "react";
import { Sidebar } from "./components/layout/Sidebar";
import { Topbar } from "./components/layout/Topbar";
import { Hero } from "./components/sections/Hero";
import { Etkinlikler } from "./components/sections/Etkinlikler";
import { Performans } from "./components/sections/Performans";
import { HizliIslemler } from "./components/sections/HizliIslemler";
import { SistemDurumu } from "./components/sections/SistemDurumu";
import { Baglantilar } from "./components/sections/Baglantilar";
import { StatKart } from "./components/ui/StatKart";
import { useReducedMotion } from "./hooks/useReducedMotion";
import { useTema } from "./hooks/useTema";
import { veriKaynagiOlustur } from "./data/veriKaynagi";

const DAR_ESIGI_PX = 900;

const veri = veriKaynagiOlustur();

function useGirisAnimasyonu(): boolean {
  const [gir, setGir] = useState(false);
  useEffect(() => {
    const rafId = requestAnimationFrame(() => setGir(true));
    return () => cancelAnimationFrame(rafId);
  }, []);
  return gir;
}

function useDarEkran(): boolean {
  const [dar, setDar] = useState(false);
  useEffect(() => {
    const oku = () => setDar(window.innerWidth < DAR_ESIGI_PX);
    oku();
    window.addEventListener("resize", oku);
    return () => window.removeEventListener("resize", oku);
  }, []);
  return dar;
}

function girisStili(gir: boolean, azaltilmisHareket: boolean, gecikmeSaniye = 0): CSSProperties {
  return {
    opacity: gir ? 1 : 0,
    transform: gir || azaltilmisHareket ? "none" : "translateY(12px)",
    transition: azaltilmisHareket
      ? "opacity 300ms"
      : `opacity 500ms ease ${gecikmeSaniye}s, transform 550ms cubic-bezier(.2,.8,.2,1) ${gecikmeSaniye}s`,
  };
}

export default function App() {
  const { koyuMu, temaDegistir } = useTema();
  const [aktifNav, setAktifNav] = useState("ana-sayfa");
  const [menuAcik, setMenuAcik] = useState(false);
  const dar = useDarEkran();
  const gir = useGirisAnimasyonu();
  const azaltilmisHareket = useReducedMotion();

  useEffect(() => {
    if (!dar) setMenuAcik(false);
  }, [dar]);

  return (
    <div style={{ background: "var(--veta-bg)", color: "var(--veta-ink)", minHeight: "100vh" }}>
      {menuAcik && <div className="veta-scrim" onClick={() => setMenuAcik(false)} />}
      <div className="veta-drawer" style={{ transform: menuAcik ? "translateX(0)" : "translateX(-105%)" }}>
        <Sidebar aktifId={aktifNav} onSec={(id) => { setAktifNav(id); setMenuAcik(false); }} drawer onKapat={() => setMenuAcik(false)} />
      </div>

      <div className="veta-shell">
        <Sidebar aktifId={aktifNav} onSec={setAktifNav} />

        <div style={{ minWidth: 0 }}>
          <Topbar darMi={dar} koyuMu={koyuMu} onMenuAc={() => setMenuAcik(true)} onTemaDegistir={temaDegistir} />

          <div style={{ maxWidth: 1180, margin: "0 auto", padding: "24px 20px 60px", position: "relative" }}>
            <Hero koyuMu={koyuMu} girisStili={girisStili(gir, azaltilmisHareket, 0)} />

            <div className="veta-stats" style={girisStili(gir, azaltilmisHareket, 0.05)}>
              {veri.statlar.map((s) => (
                <StatKart key={s.id} s={s} gir={gir} azaltilmisHareket={azaltilmisHareket} />
              ))}
            </div>

            <div className="veta-mid" style={{ marginTop: 14, ...girisStili(gir, azaltilmisHareket, 0.1) }}>
              <div>
                <Etkinlikler etkinlikler={veri.etkinlikler} />
                <Performans
                  veri={veri.performans}
                  kaynakKullanimi={veri.kaynakKullanimi}
                  koyuMu={koyuMu}
                  gir={gir}
                  azaltilmisHareket={azaltilmisHareket}
                />
              </div>

              <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
                <HizliIslemler islemler={veri.hizliIslemler} />
                <SistemDurumu durum={veri.sistemDurumu} />
                <Baglantilar baglantilar={veri.baglantilar} />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
