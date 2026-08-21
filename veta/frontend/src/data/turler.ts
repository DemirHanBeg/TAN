/**
 * VETA yönetim panosunun sunum katmanı, veriyi HİÇBİR ZAMAN doğrudan
 * üretmez — her zaman bu `VetaDurumKaynagi` arayüzünden okur. Böylece
 * "gerçek" ve "KONSEPT" (henüz bağlanmamış / uydurma) veri birbirinden
 * ayrılır ve bileşenler hangi sayının kaynağa dayandığını bilir.
 *
 * `bagli: true`  -> gerçek bir okuma/komuta dayanır (bkz. scripts/gercekVeriTopla.mjs)
 * `bagli: false` -> KONSEPT: henüz canlı bir VETA sunucusu/DB'si yok, bu sayı
 *                   yalnızca tasarımın nasıl görüneceğini göstermek için var.
 */

export interface Metrik<T> {
  deger: T;
  bagli: boolean;
  /** bagli:true ise verinin nereden geldiğinin kısa açıklaması */
  kaynak?: string;
}

/** IKON_HARITASI (bkz. components/ui/ikonEsle.tsx) içindeki tüm anahtarları kapsar. */
export type PanoIkonAdi =
  | "database"
  | "table"
  | "table2"
  | "activity"
  | "server"
  | "git-commit"
  | "file-code"
  | "package"
  | "terminal"
  | "backup"
  | "login"
  | "index"
  | "plus"
  | "download"
  | "save"
  | "report";

/** @deprecated kullanım yerine `PanoIkonAdi` tercih edilir; geriye dönük uyumluluk için tutuluyor. */
export type StatIkonAdi = PanoIkonAdi;

export interface StatKartVerisi {
  id: string;
  ad: Metrik<string>;
  deger: Metrik<number>;
  birim?: string;
  trend?: Metrik<string>;
  oranYuzde?: Metrik<number>;
  ikonAdi: PanoIkonAdi;
}

export interface EtkinlikVerisi {
  id: string;
  ikonAdi: PanoIkonAdi;
  baslik: string;
  hedef: string;
  zaman: string;
}

export interface PerformansNoktasi {
  t: string;
  v: number;
}

export interface KaynakKullanimVerisi {
  ad: string;
  oran: Metrik<number>;
}

export interface DepoBaglantisi {
  ad: string;
  deger: Metrik<string>;
}

export interface SistemDurumuVerisi {
  saglikli: Metrik<boolean>;
  baslik: Metrik<string>;
  aciklama: Metrik<string>;
}

export interface HizliIslemVerisi {
  id: string;
  ad: string;
  ikonAdi: PanoIkonAdi;
}

export interface VetaDurumKaynagi {
  statlar: StatKartVerisi[];
  etkinlikler: EtkinlikVerisi[];
  performans: PerformansNoktasi[];
  kaynakKullanimi: KaynakKullanimVerisi[];
  hizliIslemler: HizliIslemVerisi[];
  sistemDurumu: SistemDurumuVerisi;
  baglantilar: DepoBaglantisi[];
}
