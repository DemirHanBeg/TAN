# T2_T3_DEGISIKLIKLERI.md
# TancElf.tan'a uygulanacak T2+T3 değişiklikleri
# Durum: HENÜZ UYGULANMADI — gen1 (T1-only) onaylandıktan sonra uygulanacak.
# Strateji: T1 onay → T2+T3 uygula → gen2 derle → test → sabit nokta kabul

## T2: govdeDonusTipiCikar — Değişken-Dönüş Tipi Desteği

Mevcut durum: `döndür <AD>` (değişken) görüldüğünde hepsiMetinMi=0 → "tam".
Yeni: AD'nin fonksiyon gövdesindeki SON ataması metin-kanıtlıysa → "metin".

Eklenecek yardımcı fonksiyon (2957. satırdan sonra):

```
işlev degiskenMetinMi(tokenler, bas, bitis, poz, ad, islevTipKutu)
    # poz'dan geriye doğru bas'a kadar tara, ad'ın son atamasını bul
    i = poz - 1
    iken i >= bas
        t = alan(tokenler[i], 0)
        d = alan(tokenler[i], 1)
        eğer t == "AD" ve metinEsit(d, ad) ise
            eğer i + 1 < bitis ve alan(tokenler[i+1], 0) == "ESIT" ise
                rhs = i + 2
                eğer rhs < bitis ise
                    rt = alan(tokenler[rhs], 0)
                    rd = alan(tokenler[rhs], 1)
                    eğer rt == "METIN" ise
                        döndür 1
                    değilse eğer rt == "AD" ise
                        k = rhs + 1
                        eğer k < bitis ve alan(tokenler[k], 0) == "PARAC" ise
                            eğer yerlesikMetinDonerMi(rd) == 1 ise
                                döndür 1
                            son
                            eğer islevTipiAra(islevTipKutu, rd) == "metin" ise
                                döndür 1
                            son
                        son
                    son
                son
            son
        son
        i = i - 1
    son
    döndür 0
son
```

govdeDonusTipiCikar'da değişiklik (satır 3023-3024):
Mevcut:
```
                    değilse
                        hepsiMetinMi = 0
                    son
```
Yeni:
```
                    değilse
                        eğer degiskenMetinMi(tokenler, bas, bitis, j, jd, islevTipKutu) == 1 ise
                            x = 0
                        değilse
                            hepsiMetinMi = 0
                        son
                    son
```

## T3: argumanMetinMi — Değişken Argüman Desteği

Mevcut durum: Yalnız literal ve doğrudan yerleşik çağrıları tanır.
Yeni: Değişkenin aynı fonksiyon gövdesinde metin-kanıtlı ataması varsa tanır.

Not: argumanMetinMi'nin token aralığı (argBas, argBitis) fonksiyon gövdesi
dışındaki bir çağrının argümanlarını temsil eder — geriye dönük tarama için
fonksiyon gövdesinin bas/bitis gereklidir. Bu bilgi argumanMetinMi'ye
yeni parametre olarak eklenmeli VEYA parametreTipListesiOlustur'dan ÖNCE
tüm kaynak taranarak değişken tipleri çıkarılmalı (daha temiz yaklaşım).

Önerilen yaklaşım: parametreTipleriniTara'dan ÖNCE bir "degiskenTipiTara"
geçişi ekle — tüm kaynak boyunca AD ESIT METIN / AD ESIT <metinDoner> /
AD ESIT <kullaniciMetinDoner> kalıplarını topla, bir harita oluştur.
Sonra argumanMetinMi bu haritadan sorgulasın.

Bu daha kapsamlı bir değişiklik → FAZ 3, ADIM 2B olarak ayrı ele alınmalı.
