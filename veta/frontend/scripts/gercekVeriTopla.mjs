#!/usr/bin/env node
/**
 * gercekVeriTopla.mjs
 *
 * VETA yönetim paneli, üzerinde çalıştığı gerçek TAN deposundan (repo)
 * gerçek metrikler okur: son commit, commit sayısı, .tan dosya sayısı,
 * TancElf ikili (derleyici) boyutu vb. Bu script `npm run dev` / `npm run build`
 * öncesinde çalışır ve src/data/gercekVeri.json dosyasını üretir.
 *
 * Amaç: panoda "gerçek" gibi görünen ama uydurma olan sayı OLMASIN — bu
 * dosyadaki her alan, altında çalıştırılabilir bir okuma/komuta dayanır.
 */
import { execSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, resolve, join, extname } from "node:path";
import { readdirSync, statSync, writeFileSync, mkdirSync } from "node:fs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
// scripts/ -> frontend/ -> veta/ -> tan/  (TAN deposunun kökü)
const REPO_KOKU = resolve(SCRIPT_DIR, "..", "..", "..");
const CIKTI_YOLU = resolve(SCRIPT_DIR, "..", "src", "data", "gercekVeri.json");

const YOKSAY_DIZINLER = new Set([
  ".git",
  "node_modules",
  "dist",
  "veta" + "/frontend", // kendi frontend'imizi tara-dışı bırak (aşağıda ayrıca kontrol var)
]);

function komutCalistir(komut) {
  try {
    return execSync(komut, { cwd: REPO_KOKU, encoding: "utf-8" }).trim();
  } catch (hata) {
    return null;
  }
}

function dosyalariSay(baslangicDizini, uzanti) {
  let sayac = 0;
  const yigin = [baslangicDizini];
  while (yigin.length > 0) {
    const guncelDizin = yigin.pop();
    let girdiler;
    try {
      girdiler = readdirSync(guncelDizin, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const girdi of girdiler) {
      const tamYol = join(guncelDizin, girdi.name);
      if (girdi.isDirectory()) {
        if (girdi.name === ".git" || girdi.name === "node_modules" || girdi.name === "dist" || girdi.name === "frontend") {
          continue;
        }
        yigin.push(tamYol);
      } else if (girdi.isFile() && extname(girdi.name) === uzanti) {
        sayac += 1;
      }
    }
  }
  return sayac;
}

function tancElfBilgisi() {
  try {
    const yol = join(REPO_KOKU, "TancElf");
    const bilgi = statSync(yol);
    return { bayt: bilgi.size, degistirilmeTarihi: bilgi.mtime.toISOString() };
  } catch {
    return null;
  }
}

function main() {
  const sonCommitHash = komutCalistir("git rev-parse --short HEAD");
  const sonCommitMesaji = komutCalistir("git log -1 --format=%s");
  const sonCommitTarihi = komutCalistir("git log -1 --format=%cI");
  const sonCommitYazari = komutCalistir("git log -1 --format=%an");
  const toplamCommitSayisiStr = komutCalistir("git rev-list --count HEAD");
  const dal = komutCalistir("git branch --show-current");
  const uzakUrl = komutCalistir("git remote get-url origin");
  const sonHaftaCommitStr = komutCalistir('git log --since="7 days ago" --oneline');

  const tanDosyaSayisi = dosyalariSay(REPO_KOKU, ".tan");
  const vetaMdDosyaSayisi = dosyalariSay(join(REPO_KOKU, "veta"), ".md");
  const tancElf = tancElfBilgisi();

  const veri = {
    uretildi: new Date().toISOString(),
    depoKoku: REPO_KOKU,
    git: {
      sonCommitHash,
      sonCommitMesaji,
      sonCommitTarihi,
      sonCommitYazari,
      toplamCommitSayisi: toplamCommitSayisiStr ? Number(toplamCommitSayisiStr) : null,
      sonHaftaCommitSayisi: sonHaftaCommitStr ? sonHaftaCommitStr.split("\n").filter(Boolean).length : null,
      dal,
      uzakUrl,
    },
    dosyalar: {
      tanDosyaSayisi,
      vetaMdDosyaSayisi,
    },
    tancElf,
  };

  mkdirSync(dirname(CIKTI_YOLU), { recursive: true });
  writeFileSync(CIKTI_YOLU, JSON.stringify(veri, null, 2) + "\n", "utf-8");
  console.log(`[gercekVeriTopla] yazıldı -> ${CIKTI_YOLU}`);
  console.log(`[gercekVeriTopla] son commit: ${sonCommitHash} — "${sonCommitMesaji}"`);
  console.log(`[gercekVeriTopla] .tan dosya sayısı: ${tanDosyaSayisi}, TancElf: ${tancElf ? (tancElf.bayt / 1024).toFixed(0) + " KB" : "bulunamadı"}`);
}

main();
