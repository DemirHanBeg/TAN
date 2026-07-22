package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
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

		// ekle(liste, x): sona ekler (yerinde), listeyi döndürür
		"ekle": func(a []Deger, satir int) Deger {
			if len(a) < 2 {
				firlat(satir, "ekle(liste, öge) iki argüman ister")
			}
			liste, ok := a[0].(*TanListe)
			if !ok {
				firlat(satir, "ekle() ilk argümanı liste olmalı")
			}
			liste.Elemanlar = append(liste.Elemanlar, a[1])
			return liste
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
			sonuc := ""
			for _, o := range liste.Elemanlar {
				sonuc += metne(o)
			}
			return sonuc
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

		// yaz_dosya(dosya, metin): metni dosyaya yazar (üzerine)
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
