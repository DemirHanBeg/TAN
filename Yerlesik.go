package main

import (
	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"strconv"
	"strings"
	"time"
)

func rastgeleTamSayi(n int) int {
	return rand.Intn(n)
}

// Yerleşik işlevler: Tan'ın kendi parçasıdır, köprüden ödünç DEĞİLDİR.
// Liste tipi Tan'ın kendi tipi olduğu için işlemleri de yerlidir.
var yerlesikler map[string]func(args []Deger, satir int) Deger

// tanScriptArgs: argv[0]=çalıştırılan .tan dosyasının yolu, argv[1:]=kullanıcı argümanları
// (C arka ucundaki _argv ile aynı 0-tabanlı kural — bkz. DerleC.go d_arg/d_argsay).
// MainNative.go tarafından normal dosya çalıştırma yolunda doldurulur.
var tanScriptArgs []string

func init() {
	yerlesikler = map[string]func(args []Deger, satir int) Deger{

		// uzunluk(x): liste ya da metnin uzunluğu
		"uzunluk": func(a []Deger, satir int) Deger {
			if len(a) < 1 {
				firlat(satir, "uzunluk() bir argüman ister")
			}
			switch v := a[0].(type) {
			case *TanListe:
				return int64(len(v.Elemanlar))
			case *TanSozluk:
				return int64(len(v.Sira))
			case string:
				return int64(len([]rune(v)))
			}
			firlat(satir, "uzunluk() liste, sözlük veya metin ister")
			return nil
		},

		// sistemDur(kod): programi verilen cikis koduyla sonlandirir.
		"sistemDur": func(a []Deger, satir int) Deger {
			kod := int64(0)
			if len(a) > 0 {
				kod, _ = tamAl(a[0])
			}
			os.Exit(int(kod))
			return int64(0)
		},

		// ekle(liste, x): sona ekler (yerinde), listeyi döndürür
		// listeYap(n, deger): n ogeli listeyi TEK SEFERDE ayirir.
		// ekle() ile n kez buyutmek O(n^2) ayirma demek; bu O(n).
		// Buyuyen tampon (doubling buffer) desenini mumkun kilar.
		"listeYap": func(a []Deger, satir int) Deger {
			if len(a) < 1 {
				firlat(satir, "listeYap(n, deger) argüman ister")
			}
			n, _ := tamAl(a[0])
			if n < 0 {
				firlat(satir, "listeYap() negatif uzunluk")
			}
			var baslangic Deger = int64(0)
			if len(a) > 1 {
				baslangic = a[1]
			}
			ogeler := make([]Deger, n)
			for i := range ogeler {
				ogeler[i] = baslangic
			}
			return &TanListe{Elemanlar: ogeler}
		},
		"ekle": func(a []Deger, satir int) Deger {
			if len(a) < 2 {
				firlat(satir, "ekle(liste, öge) iki argüman ister")
			}
			liste, ok := a[0].(*TanListe)
			if !ok {
				firlat(satir, "ekle() ilk argümanı liste olmalı")
			}
			// YENI liste dondur — yerinde DEGISTIRME.
			// ELF arka ucu (f_liste_ekle) yeni alan ayirip kopyaliyor;
			// yorumlayici yerinde ekleyip AYNI listeyi donduruyordu.
			// Sonuc: a = [1,2]; b = ekle(a,3) sonrasi yorumlayicida
			// uzunluk(a) 3, elf'te 2 idi. Ayni kaynak farkli sonuc.
			yeni := make([]Deger, len(liste.Elemanlar), len(liste.Elemanlar)+1)
			copy(yeni, liste.Elemanlar)
			yeni = append(yeni, a[1])
			return &TanListe{Elemanlar: yeni}
		},

		// çıkar(liste): son ögeyi siler ve döndürür
		"çıkar": func(a []Deger, satir int) Deger {
			if len(a) < 1 {
				firlat(satir, "çıkar(liste) bir argüman ister")
			}
			liste, ok := a[0].(*TanListe)
			if !ok {
				firlat(satir, "çıkar() argümanı liste olmalı")
			}
			n := len(liste.Elemanlar)
			if n == 0 {
				firlat(satir, "boş listeden çıkarılamaz")
			}
			son := liste.Elemanlar[n-1]
			liste.Elemanlar = liste.Elemanlar[:n-1]
			return son
		},

		// liste(): boş liste üretir  (liste() ya da liste(a, b, c))
		"liste": func(a []Deger, satir int) Deger {
			ogeler := make([]Deger, len(a))
			copy(ogeler, a)
			return &TanListe{Elemanlar: ogeler}
		},

		// sayı_mı(x), metin_mi(x), liste_mi(x): tip denetimi
		"liste_mi": func(a []Deger, satir int) Deger {
			_, ok := a[0].(*TanListe)
			return ok
		},

		// sözlük(): boş sözlük üretir
		"sözlük": func(a []Deger, satir int) Deger {
			return YeniSozluk()
		},

		// harfler(metin): metni tek tek harflere böler, liste döndürür
		"harfler": func(a []Deger, satir int) Deger {
			m, ok := a[0].(string)
			if !ok {
				m = metne(a[0])
			}
			var ogeler []Deger
			for _, r := range m {
				ogeler = append(ogeler, string(r))
			}
			return &TanListe{Elemanlar: ogeler}
		},

		// birleştir(liste): metin listesini tek metne birleştirir
		"birleştir": func(a []Deger, satir int) Deger {
			liste, ok := a[0].(*TanListe)
			if !ok {
				firlat(satir, "birleştir() argümanı liste olmalı")
			}
			// strings.Builder: O(n) -- önceki "sonuc += metne(o)" döngüsü her
			// eklemede TÜM önceki içeriği kopyalıyordu (O(n²)), büyük liste
			// birleştirmede (ör. LSM SSTable flush, binlerce satır) ciddi
			// yavaşlık kaynağıydı. Davranış AYNI, sadece asimptotik maliyet düzeldi.
			var b strings.Builder
			for _, o := range liste.Elemanlar {
				b.WriteString(metne(o))
			}
			return b.String()
		},

		// kod(h): bir karakterin sayısal kod noktasını (rune) döndürür
		"kod": func(a []Deger, satir int) Deger {
			m, ok := a[0].(string)
			if !ok {
				m = metne(a[0])
			}
			r := []rune(m)
			if len(r) == 0 {
				firlat(satir, "kod() boş olmayan bir karakter ister")
			}
			return int64(r[0])
		},

		// karakter(n): sayısal kod noktasından tek karakterlik metin üretir
		"karakter": func(a []Deger, satir int) Deger {
			n := int(sayiAl(a, 0))
			return string(rune(n))
		},

		// argsay(): program argüman sayısı (argv[0] dahil, C arka ucuyla aynı kural)
		"argsay": func(a []Deger, satir int) Deger {
			return int64(len(tanScriptArgs))
		},

		// arg(i): i. program argümanı; aralık dışıysa boş metin
		"arg": func(a []Deger, satir int) Deger {
			i := int(sayiAl(a, 0))
			if i < 0 || i >= len(tanScriptArgs) {
				return ""
			}
			return tanScriptArgs[i]
		},

		// rastgele(n): 0 ile n-1 arası rastgele tam sayı
		"rastgele": func(a []Deger, satir int) Deger {
			n := int(sayiAl(a, 0))
			if n <= 0 {
				return float64(0)
			}
			return int64(rastgeleTamSayi(n))
		},

		// sayı(metin): metni sayıya çevirir
		// metin(x): herhangi bir değeri metne çevirir (sayı, mantık, liste, sözlük)
		"metin": func(a []Deger, satir int) Deger {
			if len(a) < 1 {
				firlat(satir, "metin() bir argüman ister")
			}
			return metne(a[0])
		},
		// metinEsit/metinBirlestir: TancElf.tan (elf arka ucunda kendi kendini
		// derlerken) statik tip izlemeden BAĞIMSIZ metin işlemleri için
		// kullanıyor (bkz. DerleElf.go aynı isimli notlar). Yorumlayıcı zaten
		// dinamik tipli olduğundan burada sadece == / + ile aynı davranış.
		"metinEsit": func(a []Deger, satir int) Deger {
			if len(a) != 2 {
				firlat(satir, "metinEsit() iki argüman ister")
			}
			sa, ok1 := a[0].(string)
			sb, ok2 := a[1].(string)
			if ok1 && ok2 && sa == sb {
				return int64(1)
			}
			return int64(0)
		},
		"metinBirlestir": func(a []Deger, satir int) Deger {
			if len(a) != 2 {
				firlat(satir, "metinBirlestir() iki argüman ister")
			}
			sa, ok1 := a[0].(string)
			sb, ok2 := a[1].(string)
			if !ok1 || !ok2 {
				firlat(satir, "metinBirlestir() metin argüman ister")
			}
			return sa + sb
		},
		"metinAl": func(a []Deger, satir int) Deger {
			if len(a) != 2 {
				firlat(satir, "metinAl() iki argüman ister")
			}
			s, ok1 := a[0].(string)
			idx, ok2 := a[1].(int64)
			if !ok1 || !ok2 {
				firlat(satir, "metinAl() metin ve tam sayı argüman ister")
			}
			runes := []rune(s)
			if idx < 0 || int(idx) >= len(runes) {
				firlat(satir, "metinAl(): indeks sınır dışı")
			}
			return string(runes[idx])
		},
		"sayı": func(a []Deger, satir int) Deger {
			switch v := a[0].(type) {
			case int64:
				return v
			case float64:
				return v
			case string:
				// Nokta/üs yoksa TAM SAYI olarak çevir (hassasiyet korunsun)
				if !strings.ContainsAny(v, ".eE") {
					if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
						return n
					}
				}
				var f float64
				fmt.Sscanf(v, "%g", &f)
				return f
			}
			return int64(0)
		},

		// --- Dosya erişimi (işletim sisteminden istenir; her dil böyle yapar) ---

		// oku(dosya): dosya içeriğini metin olarak döndürür
		"oku": func(a []Deger, satir int) Deger {
			veri, err := os.ReadFile(metne(a[0]))
			if err != nil {
				firlat(satir, "dosya okunamadı: %v", err)
			}
			return string(veri)
		},

		// dosyaVarMi(yol): dosya var mı (oku() öncesi kontrol için — oku()
		// eksik dosyada hata fırlatır, dene/yakala elf'te yok)
		"dosyaVarMi": func(a []Deger, satir int) Deger {
			_, err := os.Stat(metne(a[0]))
			return err == nil
		},

		// dosyaSil(yol): dosyayı siler (yoksa sessizce geçer — çağıran
		// dosyaVarMi ile önceden kontrol edebilir ama zorunlu değil).
		// LSM depolama motorunun compaction sonrası eski segment dosyalarını
		// temizlemesi için eklendi (bkz. kutuphane/LsmDeposu.tan).
		"dosyaSil": func(a []Deger, satir int) Deger {
			err := os.Remove(metne(a[0]))
			if err != nil && !os.IsNotExist(err) {
				firlat(satir, "dosya silinemedi: %v", err)
			}
			return int64(0)
		},

		// yaz_dosya(dosya, metin): metni dosyaya yazar (üzerine)
		// yazBaytlar(yol, liste): liste ogelerinin DUSUK BAYTINI ham yazar.
		// UTF-8 kodlamasi YAPILMAZ — ikili dosya uretimi icin.
		// tamBol(a, b): HER ARKA UCTA AYNI davranan tam sayi bolmesi.
		// Sifira dogru keser (C/Go/idiv semantigi). Tasinabilir kod icin
		// "/" yerine bunu kullan — "/" arka uclar arasinda farkli davraniyor.
		"tamBol": func(a []Deger, satir int) Deger {
			if len(a) < 2 {
				firlat(satir, "tamBol() iki argüman ister")
			}
			x, _ := tamAl(a[0])
			y, _ := tamAl(a[1])
			if y == 0 {
				firlat(satir, "sıfıra bölme")
			}
			return x / y
		},
		"yazBaytlar": func(a []Deger, satir int) Deger {
			if len(a) < 2 {
				firlat(satir, "yazBaytlar() iki argüman ister")
			}
			yol := metne(a[0])
			liste, ok := a[1].(*TanListe)
			if !ok {
				firlat(satir, "yazBaytlar() ikinci argüman liste olmalı")
			}
			baytlar := make([]byte, len(liste.Elemanlar))
			for i, o := range liste.Elemanlar {
				v, _ := tamAl(o)
				baytlar[i] = byte(v)
			}
			if err := os.WriteFile(yol, baytlar, 0755); err != nil {
				firlat(satir, "dosya yazılamadı: %v", err)
			}
			return int64(0)
		},
		"yaz_dosya": func(a []Deger, satir int) Deger {
			err := os.WriteFile(metne(a[0]), []byte(metne(a[1])), 0644)
			if err != nil {
				firlat(satir, "dosyaya yazılamadı: %v", err)
			}
			return nil
		},

		// ekle_dosya(dosya, metin): dosyanın sonuna ekler
		"ekle_dosya": func(a []Deger, satir int) Deger {
			f, err := os.OpenFile(metne(a[0]), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				firlat(satir, "dosya açılamadı: %v", err)
			}
			defer f.Close()
			f.WriteString(metne(a[1]))
			return nil
		},

		// --- Rastgele erişimli (positional) dosya G/Ç ---
		// Sayfa-tabanlı depolama motorunun (B+Tree, buffer pool) tabanı.
		// oku()/yaz_dosya() tüm dosyayı işler; bunlar bir DOSYA TANITICISI (fd)
		// üzerinden çalışır: aç -> konumla -> oku/yaz -> senkron -> kapat.
		// dosyaAcOku(yol): salt-okuma açar, fd döndürür (O_RDONLY).
		"dosyaAcOku": func(a []Deger, satir int) Deger {
			fd, err := syscall.Open(metne(a[0]), syscall.O_RDONLY, 0644)
			if err != nil {
				firlat(satir, "dosyaAcOku() açılamadı: %v", err)
			}
			return int64(fd)
		},
		// dosyaAcYaz(yol): yazma için açar, VARSA SIFIRLAR (O_WRONLY|O_CREAT|O_TRUNC).
		"dosyaAcYaz": func(a []Deger, satir int) Deger {
			fd, err := syscall.Open(metne(a[0]), syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, 0644)
			if err != nil {
				firlat(satir, "dosyaAcYaz() açılamadı: %v", err)
			}
			return int64(fd)
		},
		// dosyaAcOkuYaz(yol): oku+yaz açar, YOKSA OLUŞTURUR, içeriği KORUR
		// (O_RDWR|O_CREAT). Rastgele erişimli güncelleme için asıl kullanılan.
		"dosyaAcOkuYaz": func(a []Deger, satir int) Deger {
			fd, err := syscall.Open(metne(a[0]), syscall.O_RDWR|syscall.O_CREAT, 0644)
			if err != nil {
				firlat(satir, "dosyaAcOkuYaz() açılamadı: %v", err)
			}
			return int64(fd)
		},
		// dosyaKonumla(fd, ofset): mutlak ofsete gider (SEEK_SET), yeni konumu döndürür.
		"dosyaKonumla": func(a []Deger, satir int) Deger {
			fd, _ := tamAl(a[0])
			ofset, _ := tamAl(a[1])
			yeni, err := syscall.Seek(int(fd), ofset, 0)
			if err != nil {
				firlat(satir, "dosyaKonumla() başarısız: %v", err)
			}
			return yeni
		},
		// dosyaOkuBlok(fd, uzunluk): GÜNCEL konumdan en çok uzunluk bayt okur,
		// metin döndürür (dosya sonuysa daha kısa olabilir).
		"dosyaOkuBlok": func(a []Deger, satir int) Deger {
			fd, _ := tamAl(a[0])
			uzunluk, _ := tamAl(a[1])
			if uzunluk < 0 {
				firlat(satir, "dosyaOkuBlok() uzunluk negatif olamaz")
			}
			ara := make([]byte, uzunluk)
			n, err := syscall.Read(int(fd), ara)
			if err != nil {
				firlat(satir, "dosyaOkuBlok() okuma hatası: %v", err)
			}
			return string(ara[:n])
		},
		// dosyaYazBlok(fd, metin): GÜNCEL konuma yazar, yazılan bayt sayısını döndürür.
		"dosyaYazBlok": func(a []Deger, satir int) Deger {
			fd, _ := tamAl(a[0])
			n, err := syscall.Write(int(fd), []byte(metne(a[1])))
			if err != nil {
				firlat(satir, "dosyaYazBlok() yazma hatası: %v", err)
			}
			return int64(n)
		},
		// dosyaSenkron(fd): fsync — tamponları diske ZORLAR (dayanıklılık/crash-safe).
		"dosyaSenkron": func(a []Deger, satir int) Deger {
			fd, _ := tamAl(a[0])
			if err := syscall.Fsync(int(fd)); err != nil {
				firlat(satir, "dosyaSenkron() fsync hatası: %v", err)
			}
			return int64(0)
		},
		// dosyaKapat(fd): dosya tanıtıcısını kapatır.
		"dosyaKapat": func(a []Deger, satir int) Deger {
			fd, _ := tamAl(a[0])
			if err := syscall.Close(int(fd)); err != nil {
				firlat(satir, "dosyaKapat() kapatma hatası: %v", err)
			}
			return int64(0)
		},

		// --- Metin işleme (saf, dilin kendi parçası) ---

		// satırlar(metin): metni satırlara böler, liste döndürür
		"satırlar": func(a []Deger, satir int) Deger {
			diziler := strings.Split(strings.ReplaceAll(metne(a[0]), "\r\n", "\n"), "\n")
			var ogeler []Deger
			for _, s := range diziler {
				if s != "" {
					ogeler = append(ogeler, s)
				}
			}
			return &TanListe{Elemanlar: ogeler}
		},

		// parçala(metin, ayraç): metni ayraca göre böler, liste döndürür
		"parçala": func(a []Deger, satir int) Deger {
			diziler := strings.Split(metne(a[0]), metne(a[1]))
			ogeler := make([]Deger, len(diziler))
			for i, s := range diziler {
				ogeler[i] = s
			}
			return &TanListe{Elemanlar: ogeler}
		},

		// kırp(metin): baştan/sondan boşlukları temizler
		"kırp": func(a []Deger, satir int) Deger {
			return strings.TrimSpace(metne(a[0]))
		},

		// --- Sayısal matematik (model/AI için temel) ---
		// Bunlar şimdi CPU'da çalışır; ileride ağır sürümü GPU köprüsüne devredilir.
		"log": func(a []Deger, satir int) Deger {
			return mathLog(sayiAl(a, 0))
		},
		"e_üssü": func(a []Deger, satir int) Deger {
			return mathExp(sayiAl(a, 0))
		},
		"taban": func(a []Deger, satir int) Deger {
			return mathFloor(sayiAl(a, 0))
		},
		"tavan": func(a []Deger, satir int) Deger {
			return mathCeil(sayiAl(a, 0))
		},
		"kök": func(a []Deger, satir int) Deger {
			return mathSqrt(sayiAl(a, 0))
		},
		// yuvarla(sayı, basamak): belirtilen ondalık basamağa yuvarlar
		// yuvarla(3.14159, 2) -> 3.14   |   yuvarla(2.7, 0) -> 3
		"yuvarla": func(a []Deger, satir int) Deger {
			sayı := sayiAl(a, 0)
			basamak := 0
			if len(a) > 1 {
				basamak = int(sayiAl(a, 1))
			}
			çarpan := mathPow(10, float64(basamak))
			return mathRound(sayı*çarpan) / çarpan
		},

		// zaman(): 1970'ten beri geçen saniye (Unix zamanı)
		"zaman": func(a []Deger, satir int) Deger {
			return int64(timeNow().Unix())
		},

		// --- İnternet: HTTP istemcisi ---
		// getir(url): GET isteği yapar, cevap gövdesini metin döndürür
		"getir": func(a []Deger, satir int) Deger {
			cevap, err := http.Get(metne(a[0]))
			if err != nil {
				firlat(satir, "getir hatası: %v", err)
			}
			defer cevap.Body.Close()
			govde, _ := io.ReadAll(cevap.Body)
			return string(govde)
		},

		// gönder(url, gövde): POST isteği yapar (JSON gövdesiyle), cevabı döndürür
		"gönder": func(a []Deger, satir int) Deger {
			url := metne(a[0])
			govde := ""
			if len(a) > 1 {
				govde = metne(a[1])
			}
			cevap, err := http.Post(url, "application/json", strings.NewReader(govde))
			if err != nil {
				firlat(satir, "gönder hatası: %v", err)
			}
			defer cevap.Body.Close()
			sonuc, _ := io.ReadAll(cevap.Body)
			return string(sonuc)
		},

		// --- JSON: dillerarası ortak veri dili ---
		// json_çöz(metin): JSON metnini Tan sözlüğüne/listesine çevirir
		"json_çöz": func(a []Deger, satir int) Deger {
			var ham interface{}
			if err := json.Unmarshal([]byte(metne(a[0])), &ham); err != nil {
				firlat(satir, "json_çöz hatası: %v", err)
			}
			return goDegeriTana(ham)
		},

		// json_yap(değer): Tan değerini JSON metnine çevirir
		"json_yap": func(a []Deger, satir int) Deger {
			ham := tanDegeriGoya(a[0])
			bayt, err := json.Marshal(ham)
			if err != nil {
				firlat(satir, "json_yap hatası: %v", err)
			}
			return string(bayt)
		},

		// --- Web sunucusu ---
		// sun(port, işleyici): HTTP sunucusu başlatır. Her istek geldiğinde
		// işleyici(yol) çağrılır; döndürdüğü metin tarayıcıya gönderilir.
		"sun": func(a []Deger, satir int) Deger {
			port := int(sayiAl(a, 0))
			isleyici, ok := a[1].(IslevDeger)
			if !ok {
				firlat(satir, "sun() ikinci argümanı bir işlev olmalı")
			}
			if kuresel_yorumlayici == nil {
				firlat(satir, "sun() için yorumlayıcı hazır değil")
			}
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				sonuc := kuresel_yorumlayici.islevCagir(isleyici, []Deger{r.URL.Path})
				icerik := metne(sonuc)
				if strings.HasPrefix(strings.TrimSpace(icerik), "<") {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
				}
				fmt.Fprint(w, icerik)
			})
			fmt.Printf("Tan sunucusu çalışıyor: http://localhost:%d\n", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
				firlat(satir, "sun hatası: %v", err)
			}
			return nil
		},

		// --- Dillerarası iletişim: dış program çalıştırma ---
		// çalıştır(komut, arg1, arg2, ...): başka bir programı çağırır,
		// çıktısını metin olarak döndürür. Python, Go, node — ne olursa.
		// Örn: çalıştır("python3", "-c", "print(2+2)")  -> "4\n"
		"çalıştır": func(a []Deger, satir int) Deger {
			if len(a) < 1 {
				firlat(satir, "çalıştır() en az komut adı ister")
			}
			ad := metne(a[0])
			var args []string
			for _, x := range a[1:] {
				args = append(args, metne(x))
			}
			çıktı, err := exec.Command(ad, args...).CombinedOutput()
			if err != nil {
				// Hata olsa bile çıktıyı döndür (stderr dahil), programı durdurma
				return string(çıktı) + "\n[çalıştır hatası: " + err.Error() + "]"
			}
			return string(çıktı)
		},

		// anahtarlar(sözlük): anahtarları liste olarak döndürür
		"anahtarlar": func(a []Deger, satir int) Deger {
			s, ok := a[0].(*TanSozluk)
			if !ok {
				firlat(satir, "anahtarlar() argümanı sözlük olmalı")
			}
			ogeler := make([]Deger, len(s.Sira))
			for i, an := range s.Sira {
				ogeler[i] = an
			}
			return &TanListe{Elemanlar: ogeler}
		},

		// var_mı(sözlük, anahtar): anahtar var mı
		"var_mı": func(a []Deger, satir int) Deger {
			s, ok := a[0].(*TanSozluk)
			if !ok {
				firlat(satir, "var_mı() ilk argümanı sözlük olmalı")
			}
			_, bulundu := s.Cift[metne(a[1])]
			return bulundu
		},

		// sil(sözlük, anahtar): anahtarı siler
		"sil": func(a []Deger, satir int) Deger {
			s, ok := a[0].(*TanSozluk)
			if !ok {
				firlat(satir, "sil() ilk argümanı sözlük olmalı")
			}
			anahtar := metne(a[1])
			if _, bulundu := s.Cift[anahtar]; bulundu {
				delete(s.Cift, anahtar)
				for i, an := range s.Sira {
					if an == anahtar {
						s.Sira = append(s.Sira[:i], s.Sira[i+1:]...)
						break
					}
				}
			}
			return nil
		},
	}

	// --- camelCase takma adlar (eski alt cizgili adlar da calisir) ---
	yerlesikler["yazDosya"] = yerlesikler["yaz_dosya"]
	yerlesikler["ekleDosya"] = yerlesikler["ekle_dosya"]
	yerlesikler["jsonYap"] = yerlesikler["json_yap"]
	yerlesikler["jsonÇöz"] = yerlesikler["json_çöz"]
	yerlesikler["listeMi"] = yerlesikler["liste_mi"]
	yerlesikler["varMı"] = yerlesikler["var_mı"]
	yerlesikler["eÜssü"] = yerlesikler["e_üssü"]

	// --- sabit genişlikli tamsayı kurucuları (u8/u16/u32/u64/i8/i16/i32/i64) ---
	// Taşma tanımlı davranış: verilen değer Genislik bite maskelenir (sarma).
	sabitKurucu := func(genislik int, imzali bool) func([]Deger, int) Deger {
		return func(a []Deger, satir int) Deger {
			if len(a) < 1 {
				firlat(satir, "sabit genişlikli tamsayı kurucusu bir argüman ister")
			}
			deger, ok := sabitDegerCikar(a[0])
			if !ok {
				firlat(satir, "sabit genişlikli tamsayı kurucusu sayı bekliyor")
			}
			return sabitOlustur(genislik, imzali, deger)
		}
	}
	yerlesikler["u8"] = sabitKurucu(8, false)
	yerlesikler["u16"] = sabitKurucu(16, false)
	yerlesikler["u32"] = sabitKurucu(32, false)
	yerlesikler["u64"] = sabitKurucu(64, false)
	yerlesikler["i8"] = sabitKurucu(8, true)
	yerlesikler["i16"] = sabitKurucu(16, true)
	yerlesikler["i32"] = sabitKurucu(32, true)
	yerlesikler["i64"] = sabitKurucu(64, true)

	// --- ham bellek erişimi (simüle edilmiş düz bellek, bkz. Yorumlayici.hamBellek) ---

	// hamAyir(bayt): bump allocator ile yeni alan ayırır, başlangıç işaretçisini döndürür.
	yerlesikler["hamAyir"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "hamAyir(bayt) bir argüman ister")
		}
		boyut, ok := tamAl(a[0])
		if !ok || boyut < 0 {
			firlat(satir, "hamAyir() negatif olmayan tam sayı bekliyor")
		}
		isaretci := int64(len(kuresel_yorumlayici.hamBellek))
		kuresel_yorumlayici.hamBellek = append(kuresel_yorumlayici.hamBellek, make([]byte, boyut)...)
		return isaretci
	}

	// hamOku(isaretci): belirtilen adresten tek bayt okur (0-255 tam sayı olarak).
	yerlesikler["hamOku"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "hamOku(isaretci) bir argüman ister")
		}
		p, ok := tamAl(a[0])
		if !ok || p < 0 || p >= int64(len(kuresel_yorumlayici.hamBellek)) {
			firlat(satir, "hamOku(): sınır dışı adres")
		}
		return int64(kuresel_yorumlayici.hamBellek[p])
	}

	// hamYaz(isaretci, deger): belirtilen adrese tek bayt yazar (deger mod 256).
	yerlesikler["hamYaz"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "hamYaz(isaretci, deger) iki argüman ister")
		}
		p, ok := tamAl(a[0])
		if !ok || p < 0 || p >= int64(len(kuresel_yorumlayici.hamBellek)) {
			firlat(satir, "hamYaz(): sınır dışı adres")
		}
		v, ok := sabitDegerCikar(a[1])
		if !ok {
			firlat(satir, "hamYaz() değeri tam sayı olmalı")
		}
		kuresel_yorumlayici.hamBellek[p] = byte(v)
		return nil
	}

	// hamOku8(adres): adresten 8 baytlık (word) tam sayı okur (little-endian).
	// Sayfa-tabanlı depolama: bellekEsle/arenaAyir'dan dönen bloklara word erişimi.
	yerlesikler["hamOku8"] = func(a []Deger, satir int) Deger {
		p, _ := tamAl(a[0])
		hb := kuresel_yorumlayici.hamBellek
		if p < 0 || p+8 > int64(len(hb)) {
			firlat(satir, "hamOku8(): sınır dışı adres")
		}
		var v int64
		for i := int64(0); i < 8; i++ {
			v |= int64(hb[p+i]) << uint(8*i)
		}
		return v
	}

	// hamYaz8(adres, deger): adrese 8 baytlık tam sayı yazar (little-endian).
	yerlesikler["hamYaz8"] = func(a []Deger, satir int) Deger {
		p, _ := tamAl(a[0])
		v, _ := tamAl(a[1])
		hb := kuresel_yorumlayici.hamBellek
		if p < 0 || p+8 > int64(len(hb)) {
			firlat(satir, "hamYaz8(): sınır dışı adres")
		}
		for i := int64(0); i < 8; i++ {
			hb[p+i] = byte(v >> uint(8*i))
		}
		return int64(0)
	}

	// bellekEsle(boyut): boyut baytlık bellek bloğu ayırır, başlangıç adresini
	// döndürür. Native'de gerçek mmap (anonim); yorumlayıcıda simüle düz bellek.
	// İki modda da: dönen adrese hamOku8/hamYaz8 ile erişilir (round-trip aynı).
	yerlesikler["bellekEsle"] = func(a []Deger, satir int) Deger {
		boyut, ok := tamAl(a[0])
		if !ok || boyut < 0 {
			firlat(satir, "bellekEsle(boyut) negatif olmayan tam sayı ister")
		}
		p := int64(len(kuresel_yorumlayici.hamBellek))
		kuresel_yorumlayici.hamBellek = append(kuresel_yorumlayici.hamBellek, make([]byte, boyut)...)
		return p
	}

	// bellekCoz(adres, boyut): native'de munmap; yorumlayıcıda simüle bellek
	// geri verilmez (no-op) — API tutarlılığı için var.
	yerlesikler["bellekCoz"] = func(a []Deger, satir int) Deger {
		return int64(0)
	}

	// hamOku4(adres): adresten 4 baytlık (32-bit) tam sayı okur (little-endian,
	// işaretsiz — üst bitler sıfır). Sayfa başlığı/uzunluk alanları için.
	yerlesikler["hamOku4"] = func(a []Deger, satir int) Deger {
		p, _ := tamAl(a[0])
		hb := kuresel_yorumlayici.hamBellek
		if p < 0 || p+4 > int64(len(hb)) {
			firlat(satir, "hamOku4(): sınır dışı adres")
		}
		var v int64
		for i := int64(0); i < 4; i++ {
			v |= int64(hb[p+i]) << uint(8*i)
		}
		return v
	}

	// hamYaz4(adres, deger): adrese 4 baytlık tam sayı yazar (little-endian,
	// deger 2^32 mod alınır).
	yerlesikler["hamYaz4"] = func(a []Deger, satir int) Deger {
		p, _ := tamAl(a[0])
		v, _ := tamAl(a[1])
		hb := kuresel_yorumlayici.hamBellek
		if p < 0 || p+4 > int64(len(hb)) {
			firlat(satir, "hamYaz4(): sınır dışı adres")
		}
		for i := int64(0); i < 4; i++ {
			hb[p+i] = byte(v >> uint(8*i))
		}
		return int64(0)
	}

	// hamOkuBayt(adres): tek bayt okur (0-255). hamOku ile aynı — bellekEsle
	// ailesiyle (hamOku4/hamOku8) tutarlı adlandırma için.
	yerlesikler["hamOkuBayt"] = func(a []Deger, satir int) Deger {
		p, ok := tamAl(a[0])
		if !ok || p < 0 || p >= int64(len(kuresel_yorumlayici.hamBellek)) {
			firlat(satir, "hamOkuBayt(): sınır dışı adres")
		}
		return int64(kuresel_yorumlayici.hamBellek[p])
	}

	// hamYazBayt(adres, deger): tek bayt yazar (deger mod 256). hamYaz ile
	// aynı — bellekEsle ailesiyle tutarlı adlandırma için.
	yerlesikler["hamYazBayt"] = func(a []Deger, satir int) Deger {
		p, ok := tamAl(a[0])
		if !ok || p < 0 || p >= int64(len(kuresel_yorumlayici.hamBellek)) {
			firlat(satir, "hamYazBayt(): sınır dışı adres")
		}
		v, ok := sabitDegerCikar(a[1])
		if !ok {
			firlat(satir, "hamYazBayt() değeri tam sayı olmalı")
		}
		kuresel_yorumlayici.hamBellek[p] = byte(v)
		return int64(0)
	}

	// bellekKopyala(hedef, kaynak, uzunluk): memmove — hedef/kaynak aralıkları
	// çakışsa bile doğru sonuç verir (Go'nun copy() ilkeli çakışma-güvenlidir).
	yerlesikler["bellekKopyala"] = func(a []Deger, satir int) Deger {
		hedef, _ := tamAl(a[0])
		kaynak, _ := tamAl(a[1])
		uzunluk, ok := tamAl(a[2])
		if !ok || uzunluk < 0 {
			firlat(satir, "bellekKopyala() uzunluk negatif olamaz")
		}
		hb := kuresel_yorumlayici.hamBellek
		if hedef < 0 || hedef+uzunluk > int64(len(hb)) || kaynak < 0 || kaynak+uzunluk > int64(len(hb)) {
			firlat(satir, "bellekKopyala(): sınır dışı adres")
		}
		copy(hb[hedef:hedef+uzunluk], hb[kaynak:kaynak+uzunluk])
		return int64(0)
	}

	// bellekDoldur(adres, deger, uzunluk): memset — uzunluk baytı deger mod 256
	// ile doldurur.
	yerlesikler["bellekDoldur"] = func(a []Deger, satir int) Deger {
		adres, _ := tamAl(a[0])
		deger, _ := tamAl(a[1])
		uzunluk, ok := tamAl(a[2])
		if !ok || uzunluk < 0 {
			firlat(satir, "bellekDoldur() uzunluk negatif olamaz")
		}
		hb := kuresel_yorumlayici.hamBellek
		if adres < 0 || adres+uzunluk > int64(len(hb)) {
			firlat(satir, "bellekDoldur(): sınır dışı adres")
		}
		b := byte(deger)
		for i := int64(0); i < uzunluk; i++ {
			hb[adres+i] = b
		}
		return int64(0)
	}

	// ---- Kripto (Faz A #1 — bkz. NexusCore/FazA_Kapsam.md) ----
	// Kendi kripto yazma tuzağına düşülmedi: Go'nun crypto/sha256 ve
	// crypto/aes+cipher (AES-256-GCM, NIST standardı, kimlik doğrulamalı
	// şifreleme) stdlib'i sarılıyor — diğer TÜM yerleşiklerin (oku/yaz_dosya
	// os.* sarması, kök/log math.* sarması gibi) izlediği AYNI desen.

	// sha256Ozet(metin): SHA-256 özetini hex metin olarak döndürür.
	yerlesikler["sha256Ozet"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "sha256Ozet() bir argüman ister")
		}
		toplam := sha256.Sum256([]byte(metne(a[0])))
		return hex.EncodeToString(toplam[:])
	}

	// aesSifrele(anahtar, açıkMetin): AES-256-GCM ile şifreler, hex metin
	// döndürür (rastgele nonce + şifreli metin + kimlik doğrulama etiketi
	// birlikte, tek parça). Anahtar dizesi SHA-256 ile 32 bayta türetilir —
	// BASİT türetme, gerçek parola-tabanlı KDF (tuzlu PBKDF2/Argon2) DEĞİL,
	// bilinen v1 sınırı (bkz. FazA_Kapsam.md).
	yerlesikler["aesSifrele"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "aesSifrele(anahtar, açıkMetin) iki argüman ister")
		}
		anahtarHam := sha256.Sum256([]byte(metne(a[0])))
		blok, err := aes.NewCipher(anahtarHam[:])
		if err != nil {
			firlat(satir, "aesSifrele(): %v", err)
		}
		gcm, err := cipher.NewGCM(blok)
		if err != nil {
			firlat(satir, "aesSifrele(): %v", err)
		}
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(cryptoRand.Reader, nonce); err != nil {
			firlat(satir, "aesSifrele(): rastgele nonce üretilemedi: %v", err)
		}
		sifreli := gcm.Seal(nonce, nonce, []byte(metne(a[1])), nil)
		return hex.EncodeToString(sifreli)
	}

	// aesCoz(anahtar, şifreliHex): aesSifrele()'nin tersi. Yanlış anahtar ya
	// da bozulmuş/değiştirilmiş veri kimlik doğrulamayı (GCM tag) başarısız
	// kılar, firlat() ile hata verir (dene/yakala ile yakalanabilir).
	yerlesikler["aesCoz"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "aesCoz(anahtar, şifreliMetin) iki argüman ister")
		}
		anahtarHam := sha256.Sum256([]byte(metne(a[0])))
		ham, err := hex.DecodeString(metne(a[1]))
		if err != nil {
			firlat(satir, "aesCoz(): geçersiz hex metin")
		}
		blok, err := aes.NewCipher(anahtarHam[:])
		if err != nil {
			firlat(satir, "aesCoz(): %v", err)
		}
		gcm, err := cipher.NewGCM(blok)
		if err != nil {
			firlat(satir, "aesCoz(): %v", err)
		}
		boyut := gcm.NonceSize()
		if len(ham) < boyut {
			firlat(satir, "aesCoz(): şifreli metin çok kısa")
		}
		nonce, govde := ham[:boyut], ham[boyut:]
		acik, err := gcm.Open(nil, nonce, govde, nil)
		if err != nil {
			firlat(satir, "aesCoz(): kimlik doğrulama başarısız (yanlış anahtar ya da bozulmuş veri)")
		}
		return string(acik)
	}

	// ---- Soket / TCP (Faz A #2 — bkz. NexusCore/FazA_Kapsam.md) ----
	// Go'nun net paketi sarılıyor (elf yolunda YOK — OS ağ yığınına erişim
	// için gerçek syscall arayüzü gerekir, kapsam dışı, bkz. TanSoket notu).

	// soketDinle(port): TCP dinleyici açar (sunucu tarafı), tutamaç döndürür.
	yerlesikler["soketDinle"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "soketDinle(port) bir argüman ister")
		}
		port, ok := tamAl(a[0])
		if !ok {
			firlat(satir, "soketDinle() port tam sayı olmalı")
		}
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			firlat(satir, "soketDinle(): %v", err)
		}
		return &TanSoket{Dinleyici: l}
	}

	// soketKabulEt(dinleyici): bir istemci bağlanana kadar BLOKLAR, bağlantı
	// tutamacı döndürür.
	yerlesikler["soketKabulEt"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "soketKabulEt() bir argüman ister")
		}
		s, ok := a[0].(*TanSoket)
		if !ok || s.Dinleyici == nil {
			firlat(satir, "soketKabulEt() bir dinleyici soketi (soketDinle) bekliyor")
		}
		baglanti, err := s.Dinleyici.Accept()
		if err != nil {
			firlat(satir, "soketKabulEt(): %v", err)
		}
		return &TanSoket{Baglanti: baglanti}
	}

	// soketBaglan(adres, port): istemci tarafı — uzak sunucuya bağlanır.
	yerlesikler["soketBaglan"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "soketBaglan(adres, port) iki argüman ister")
		}
		adres := metne(a[0])
		port, ok := tamAl(a[1])
		if !ok {
			firlat(satir, "soketBaglan() port tam sayı olmalı")
		}
		baglanti, err := net.Dial("tcp", fmt.Sprintf("%s:%d", adres, port))
		if err != nil {
			firlat(satir, "soketBaglan(): %v", err)
		}
		return &TanSoket{Baglanti: baglanti}
	}

	// soketYaz(baglanti, metin): metni bağlantıya yazar, yazılan bayt sayısını döndürür.
	yerlesikler["soketYaz"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "soketYaz(baglanti, metin) iki argüman ister")
		}
		s, ok := a[0].(*TanSoket)
		if !ok || s.Baglanti == nil {
			firlat(satir, "soketYaz() bir bağlantı soketi bekliyor")
		}
		n, err := s.Baglanti.Write([]byte(metne(a[1])))
		if err != nil {
			firlat(satir, "soketYaz(): %v", err)
		}
		return int64(n)
	}

	// soketOku(baglanti, [maxBayt]): TEK bir okuma çağrısı yapar (en fazla
	// maxBayt, varsayılan 4096), okunan metni döndürür. Karşı taraf bağlantıyı
	// kapattıysa boş metin döner (EOF sessizce, hata değil).
	yerlesikler["soketOku"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "soketOku(baglanti, [maxBayt]) en az bir argüman ister")
		}
		s, ok := a[0].(*TanSoket)
		if !ok || s.Baglanti == nil {
			firlat(satir, "soketOku() bir bağlantı soketi bekliyor")
		}
		boyut := 4096
		if len(a) > 1 {
			b, ok2 := tamAl(a[1])
			if ok2 {
				boyut = int(b)
			}
		}
		tampon := make([]byte, boyut)
		n, err := s.Baglanti.Read(tampon)
		if err != nil && err != io.EOF {
			firlat(satir, "soketOku(): %v", err)
		}
		return string(tampon[:n])
	}

	// soketKapat(soket): dinleyici ya da bağlantı — hangisiyse onu kapatır.
	yerlesikler["soketKapat"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "soketKapat() bir argüman ister")
		}
		s, ok := a[0].(*TanSoket)
		if !ok {
			firlat(satir, "soketKapat() bir soket bekliyor")
		}
		if s.Baglanti != nil {
			s.Baglanti.Close()
		}
		if s.Dinleyici != nil {
			s.Dinleyici.Close()
		}
		return int64(0)
	}

	// ---- Thread / mutex / kanal (Faz A #4 — bkz. NexusCore/FazA_Kapsam.md) ----
	// Go'nun goroutine/sync.Mutex/channel'ı sarılıyor. TanIplik/TanKilit/
	// TanKanal tipleri (Yorumlayici.go) + Kapsam'ın kilitlenmesi bkz. o
	// dosyadaki tasarım notu.

	// içParcaLat(islev, ...args): islev'i YENİ bir goroutine'de çalıştırır,
	// hemen bir iplik tutamacı döndürür (BLOKLAMAZ). Sonucu almak için
	// iplikBekle() kullan.
	yerlesikler["içParcaLat"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "içParcaLat(islev, ...args) en az bir argüman ister")
		}
		islev, ok := a[0].(IslevDeger)
		if !ok {
			firlat(satir, "içParcaLat() ilk argüman bir işlev olmalı")
		}
		args := append([]Deger{}, a[1:]...)
		sonucKanali := make(chan Deger, 1)
		y := kuresel_yorumlayici
		go func() {
			sonucKanali <- y.islevCagir(islev, args)
		}()
		return &TanIplik{Bitti: sonucKanali}
	}

	// iplikBekle(iplik): iş parçacığı bitene kadar bloklar, döndürdüğü
	// değeri verir (döndür yoksa yok).
	yerlesikler["iplikBekle"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "iplikBekle() bir argüman ister")
		}
		s, ok := a[0].(*TanIplik)
		if !ok {
			firlat(satir, "iplikBekle() bir iplik tutamacı (içParcaLat sonucu) bekliyor")
		}
		return <-s.Bitti
	}

	// kilitOlustur(): yeni bir mutex (kilit) döndürür.
	yerlesikler["kilitOlustur"] = func(a []Deger, satir int) Deger {
		return &TanKilit{}
	}

	// kilitle(kilit): kilidi al — başka bir iş parçacığı zaten tutuyorsa
	// serbest kalana kadar BLOKLAR.
	yerlesikler["kilitle"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "kilitle() bir argüman ister")
		}
		k, ok := a[0].(*TanKilit)
		if !ok {
			firlat(satir, "kilitle() bir kilit (kilitOlustur sonucu) bekliyor")
		}
		k.Mu.Lock()
		return int64(0)
	}

	// kilidiAc(kilit): daha önce kilitle() ile alınmış kilidi serbest bırakır.
	yerlesikler["kilidiAc"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "kilidiAc() bir argüman ister")
		}
		k, ok := a[0].(*TanKilit)
		if !ok {
			firlat(satir, "kilidiAc() bir kilit bekliyor")
		}
		k.Mu.Unlock()
		return int64(0)
	}

	// kanalOlustur([tamponBoyu]): iş parçacıkları arası mesajlaşma kanalı.
	// tamponBoyu verilmezse (ya da 0) SENKRON kanal (gönderen, alan hazır
	// olana kadar bloklanır).
	yerlesikler["kanalOlustur"] = func(a []Deger, satir int) Deger {
		tampon := 0
		if len(a) > 0 {
			b, ok := tamAl(a[0])
			if ok {
				tampon = int(b)
			}
		}
		return &TanKanal{Ch: make(chan Deger, tampon)}
	}

	// kanalGonder(kanal, deger): kanala değer gönderir (tampon doluysa/
	// senkron kanalda BLOKLAR).
	yerlesikler["kanalGonder"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "kanalGonder(kanal, deger) iki argüman ister")
		}
		k, ok := a[0].(*TanKanal)
		if !ok {
			firlat(satir, "kanalGonder() bir kanal (kanalOlustur sonucu) bekliyor")
		}
		k.Ch <- a[1]
		return int64(0)
	}

	// kanalAl(kanal): kanaldan bir değer okur (hazır olana kadar BLOKLAR).
	// Kanal kanalKapat() ile kapatılmışsa ve boşsa yok döner.
	yerlesikler["kanalAl"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "kanalAl() bir argüman ister")
		}
		k, ok := a[0].(*TanKanal)
		if !ok {
			firlat(satir, "kanalAl() bir kanal bekliyor")
		}
		v, acikMi := <-k.Ch
		if !acikMi {
			return nil
		}
		return v
	}

	// kanalKapat(kanal): kanalı kapatır — bekleyen kanalAl() çağrıları yok
	// döner, kapalı kanala kanalGonder() ise panikler (Go'nun kendi kuralı).
	yerlesikler["kanalKapat"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "kanalKapat() bir argüman ister")
		}
		k, ok := a[0].(*TanKanal)
		if !ok {
			firlat(satir, "kanalKapat() bir kanal bekliyor")
		}
		close(k.Ch)
		return int64(0)
	}

	// ---- Atomik (Faz A #6 — bkz. NexusCore/FazA_Kapsam.md) ----
	// Go'nun sync/atomic'i (atomic.Int64) sarılıyor — kilit'e göre daha
	// ucuz, TEK bir sayı için lock-free paylaşımlı durum.

	// atomikOlustur([baslangic]): yeni atomik sayaç (varsayılan 0).
	yerlesikler["atomikOlustur"] = func(a []Deger, satir int) Deger {
		at := &TanAtomik{}
		if len(a) > 0 {
			v, ok := tamAl(a[0])
			if ok {
				at.Deger.Store(v)
			}
		}
		return at
	}

	// atomikEkle(atomik, miktar): miktarı atomik olarak ekler, YENİ değeri döndürür.
	yerlesikler["atomikEkle"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "atomikEkle(atomik, miktar) iki argüman ister")
		}
		at, ok := a[0].(*TanAtomik)
		if !ok {
			firlat(satir, "atomikEkle() bir atomik sayaç (atomikOlustur sonucu) bekliyor")
		}
		miktar, ok := tamAl(a[1])
		if !ok {
			firlat(satir, "atomikEkle() miktar tam sayı olmalı")
		}
		return at.Deger.Add(miktar)
	}

	// atomikOku(atomik): mevcut değeri okur.
	yerlesikler["atomikOku"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "atomikOku() bir argüman ister")
		}
		at, ok := a[0].(*TanAtomik)
		if !ok {
			firlat(satir, "atomikOku() bir atomik sayaç bekliyor")
		}
		return at.Deger.Load()
	}

	// atomikAyarla(atomik, deger): değeri atomik olarak yazar, ESKİ değeri döndürür.
	yerlesikler["atomikAyarla"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "atomikAyarla(atomik, deger) iki argüman ister")
		}
		at, ok := a[0].(*TanAtomik)
		if !ok {
			firlat(satir, "atomikAyarla() bir atomik sayaç bekliyor")
		}
		deger, ok := tamAl(a[1])
		if !ok {
			firlat(satir, "atomikAyarla() değer tam sayı olmalı")
		}
		return at.Deger.Swap(deger)
	}

	// atomikKarsilastirDegistir(atomik, beklenen, yeni): mevcut değer
	// beklenen İSE yeni değere değiştirir (compare-and-swap). Başarılıysa
	// doğru, değilse (başka bir iş parçacığı araya girmişse) yanlış döner.
	yerlesikler["atomikKarsilastirDegistir"] = func(a []Deger, satir int) Deger {
		if len(a) < 3 {
			firlat(satir, "atomikKarsilastirDegistir(atomik, beklenen, yeni) üç argüman ister")
		}
		at, ok := a[0].(*TanAtomik)
		if !ok {
			firlat(satir, "atomikKarsilastirDegistir() bir atomik sayaç bekliyor")
		}
		beklenen, ok1 := tamAl(a[1])
		yeni, ok2 := tamAl(a[2])
		if !ok1 || !ok2 {
			firlat(satir, "atomikKarsilastirDegistir() beklenen/yeni tam sayı olmalı")
		}
		return at.Deger.CompareAndSwap(beklenen, yeni)
	}

	// FFI / dış kütüphane yerleşikleri (disKutuphaneAc/disIslevBul/
	// disIslevCagir/disKutuphaneKapat) — bkz. YerlesikFFI_linux.go /
	// YerlesikFFI_diger.go. Platforma göre AYRI dosyada (purego'nun
	// dlopen/dlsym API'si SADECE unix benzeri sistemlerde var, Windows'ta
	// derleme hatası veriyordu — bkz. o dosyalardaki not).
}

// goDegeriTana: json_çöz'ün ürettiği Go değerini Tan değerine çevirir
func goDegeriTana(v interface{}) Deger {
	switch t := v.(type) {
	case nil:
		return nil
	case bool:
		return t
	case float64:
		return t
	case string:
		return t
	case []interface{}:
		ogeler := make([]Deger, len(t))
		for i, e := range t {
			ogeler[i] = goDegeriTana(e)
		}
		return &TanListe{Elemanlar: ogeler}
	case map[string]interface{}:
		s := YeniSozluk()
		for anahtar, deger := range t {
			s.koy(anahtar, goDegeriTana(deger))
		}
		return s
	}
	return nil
}

// tanDegeriGoya: json_yap için Tan değerini Go değerine çevirir
func tanDegeriGoya(d Deger) interface{} {
	switch t := d.(type) {
	case nil:
		return nil
	case bool:
		return t
	case float64:
		return t
	case string:
		return t
	case *TanListe:
		ogeler := make([]interface{}, len(t.Elemanlar))
		for i, e := range t.Elemanlar {
			ogeler[i] = tanDegeriGoya(e)
		}
		return ogeler
	case *TanSozluk:
		m := map[string]interface{}{}
		for _, anahtar := range t.Sira {
			m[anahtar] = tanDegeriGoya(t.Cift[anahtar])
		}
		return m
	}
	return nil
}

// Matematik sarmalayıcıları (math paketine köprü)
func mathLog(x float64) float64   { return math.Log(x) }
func mathExp(x float64) float64   { return math.Exp(x) }
func mathFloor(x float64) float64 { return math.Floor(x) }
func mathCeil(x float64) float64  { return math.Ceil(x) }
func mathSqrt(x float64) float64  { return math.Sqrt(x) }

// timeNow: time.Now sarmalayıcısı (zaman yerleşiği için)
func timeNow() time.Time           { return time.Now() }
func mathPow(x, y float64) float64 { return math.Pow(x, y) }
func mathRound(x float64) float64  { return math.Round(x) }
