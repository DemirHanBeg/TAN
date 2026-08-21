import gercekVeri from "./gercekVeri.json";
import type {
  DepoBaglantisi,
  EtkinlikVerisi,
  HizliIslemVerisi,
  KaynakKullanimVerisi,
  Metrik,
  PerformansNoktasi,
  SistemDurumuVerisi,
  StatKartVerisi,
  VetaDurumKaynagi,
} from "./turler";

const GERCEK_VERI_KAYNAGI = "scripts/gercekVeriTopla.mjs (git + fs okuması, tan/ deposu)";

function gercek<T>(deger: T, kaynak: string = GERCEK_VERI_KAYNAGI): Metrik<T> {
  return { deger, bagli: true, kaynak };
}

function konsept<T>(deger: T): Metrik<T> {
  return { deger, bagli: false };
}

function statlarOlustur(): StatKartVerisi[] {
  const { git, dosyalar, tancElf } = gercekVeri;

  const commitTrend = git.sonHaftaCommitSayisi != null
    ? gercek(`↑ son 7 günde ${git.sonHaftaCommitSayisi} commit`)
    : konsept("");

  const tancElfKB = tancElf ? Math.round(tancElf.bayt / 1024) : 0;

  return [
    {
      id: "commit-sayisi",
      ad: gercek("Toplam Commit"),
      deger: gercek(git.toplamCommitSayisi ?? 0),
      trend: commitTrend,
      ikonAdi: "git-commit",
    },
    {
      id: "tan-dosya-sayisi",
      ad: gercek(".tan Dosyası"),
      deger: gercek(dosyalar.tanDosyaSayisi),
      trend: gercek("tan/ deposu geneli"),
      ikonAdi: "file-code",
    },
    {
      id: "tancelf-boyut",
      ad: gercek("TancElf İkili Boyutu"),
      deger: tancElf ? gercek(tancElfKB) : konsept(0),
      birim: " KB",
      ikonAdi: "package",
    },
    {
      id: "veta-md-sayisi",
      ad: gercek("VETA Dokümanı"),
      deger: gercek(dosyalar.vetaMdDosyaSayisi),
      trend: gercek("veta/*.md"),
      ikonAdi: "database",
    },
  ];
}

/** Sabit örnek etkinlikler — henüz canlı bir VETA sorgu motoru/DB yok, bu yüzden KONSEPT. */
function etkinliklerOlustur(): EtkinlikVerisi[] {
  return [
    { id: "e1", ikonAdi: "table2", baslik: "Tablo oluşturuldu", hedef: "kullanicilar", zaman: "2 dakika önce" },
    { id: "e2", ikonAdi: "terminal", baslik: "Sorgu çalıştırıldı", hedef: "satislar", zaman: "5 dakika önce" },
    { id: "e3", ikonAdi: "backup", baslik: "Veritabanı yedeği alındı", hedef: "musteri_veri", zaman: "15 dakika önce" },
    { id: "e4", ikonAdi: "login", baslik: "Kullanıcı girişi yapıldı", hedef: "admin", zaman: "20 dakika önce" },
    { id: "e5", ikonAdi: "index", baslik: "İndeks oluşturuldu", hedef: "urunler", zaman: "30 dakika önce" },
  ];
}

/** Örnek zaman serisi — canlı sistem izleme (metrics) borusu henüz yok, KONSEPT. */
function performansOlustur(): PerformansNoktasi[] {
  const temelDegerler = [30, 34, 32, 40, 45, 52, 58, 66, 63, 70, 68, 72, 67];
  return temelDegerler.map((v, i) => ({ t: `${String(i * 2).padStart(2, "0")}:00`, v }));
}

function kaynakKullanimiOlustur(): KaynakKullanimVerisi[] {
  return [
    { ad: "CPU Kullanımı", oran: konsept(32) },
    { ad: "Bellek Kullanımı", oran: konsept(58) },
    { ad: "Disk Kullanımı", oran: konsept(62) },
  ];
}

function hizliIslemlerOlustur(): HizliIslemVerisi[] {
  return [
    { id: "yeni-db", ad: "Yeni Veritabanı", ikonAdi: "plus" },
    { id: "yeni-tablo", ad: "Yeni Tablo", ikonAdi: "table" },
    { id: "sorgu", ad: "Sorgu Oluştur", ikonAdi: "terminal" },
    { id: "ice-aktar", ad: "Veri İçe Aktar", ikonAdi: "download" },
    { id: "yedek", ad: "Yedekleme Al", ikonAdi: "save" },
    { id: "rapor", ad: "Rapor Al", ikonAdi: "report" },
  ];
}

/** "Sistem Sağlıklı" iddiası canlı bir sağlık kontrolüne dayanmıyor — KONSEPT. */
function sistemDurumuOlustur(): SistemDurumuVerisi {
  return {
    saglikli: konsept(true),
    baslik: konsept("Sistem Sağlıklı"),
    aciklama: konsept("Tüm servisler sorunsuz çalışıyor."),
  };
}

/** Bağlantılar kartı: sahte sunucu/port yerine gerçek depo bilgisi. */
function baglantilarOlustur(): DepoBaglantisi[] {
  const { git, depoKoku } = gercekVeri;
  const kisaDepo = depoKoku.split(/[\\/]/).filter(Boolean).slice(-1)[0] ?? depoKoku;
  const uzakGosterim = git.uzakUrl ? git.uzakUrl.replace(/^https:\/\//, "").replace(/\.git$/, "") : null;
  const sonCommitGosterim = git.sonCommitHash
    ? `${git.sonCommitHash} — ${(git.sonCommitMesaji ?? "").slice(0, 28)}${(git.sonCommitMesaji ?? "").length > 28 ? "…" : ""}`
    : null;

  return [
    { ad: "Depo", deger: gercek(kisaDepo, "fs (script çalıştığı depo kökü)") },
    { ad: "Dal", deger: git.dal ? gercek(git.dal, "git branch --show-current") : konsept("bilinmiyor") },
    { ad: "Son Commit", deger: sonCommitGosterim ? gercek(sonCommitGosterim, "git log -1") : konsept("bilinmiyor") },
    { ad: "Uzak", deger: uzakGosterim ? gercek(uzakGosterim, "git remote get-url origin") : konsept("bağlanmadı") },
  ];
}

export function veriKaynagiOlustur(): VetaDurumKaynagi {
  return {
    statlar: statlarOlustur(),
    etkinlikler: etkinliklerOlustur(),
    performans: performansOlustur(),
    kaynakKullanimi: kaynakKullanimiOlustur(),
    hizliIslemler: hizliIslemlerOlustur(),
    sistemDurumu: sistemDurumuOlustur(),
    baglantilar: baglantilarOlustur(),
  };
}
