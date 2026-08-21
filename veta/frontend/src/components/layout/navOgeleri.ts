import {
  Activity,
  Database,
  HardDriveDownload,
  Home,
  KeyRound,
  ListTree,
  ScrollText,
  Settings,
  Table2,
  Terminal,
  Users,
  FileBarChart,
  type LucideIcon,
} from "lucide-react";

export type NavOgesi = {
  id: string;
  ad: string;
  Ikon: LucideIcon;
};

/** Sol navigasyon — sabit menü yapısı, veri kaynağı gerektirmez. */
export const NAV_OGELERI: NavOgesi[] = [
  { id: "ana-sayfa", ad: "Ana Sayfa", Ikon: Home },
  { id: "veritabanlari", ad: "Veritabanları", Ikon: Database },
  { id: "tablolar", ad: "Tablolar", Ikon: Table2 },
  { id: "sorgular", ad: "Sorgular", Ikon: Terminal },
  { id: "indeksler", ad: "İndeksler", Ikon: ListTree },
  { id: "kullanicilar", ad: "Kullanıcılar", Ikon: Users },
  { id: "yetkilendirme", ad: "Yetkilendirme", Ikon: KeyRound },
  { id: "yedeklemeler", ad: "Yedeklemeler", Ikon: HardDriveDownload },
  { id: "ayarlar", ad: "Ayarlar", Ikon: Settings },
  { id: "raporlar", ad: "Raporlar", Ikon: FileBarChart },
  { id: "loglar", ad: "Loglar", Ikon: ScrollText },
  { id: "izleme", ad: "İzleme", Ikon: Activity },
];
