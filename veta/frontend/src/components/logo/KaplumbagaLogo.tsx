type Props = {
  boyut?: number;
  renk?: string;
};

/**
 * VETA marka işareti: kaplumbağa — sade, geometrik, ince çizgi (hairline
 * stroke), gerçekçi bir çizim değil. Kabuk deseni, panodaki altıgen
 * kabuk-scute watermark'ıyla (bkz. watermark/Kabuk.tsx) aynı motifi taşır:
 * kabuk = çini karo = bellek sayfası metaforu.
 *
 * Not: Bu istisna SADECE logo için geçerli — referans dosyanın "literal
 * hayvan yasak" kuralı burada Demir'in açık kararıyla aşılıyor.
 */
export function KaplumbagaLogo({ boyut = 30, renk = "var(--veta-tur)" }: Props) {
  return (
    <svg width={boyut} height={boyut} viewBox="0 0 40 40" aria-hidden="true">
      <g fill="none" stroke={renk} strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
        {/* kabuk */}
        <ellipse cx="20" cy="23" rx="13.5" ry="10.5" />
        {/* kabuk-scute iç deseni: merkezi altıgen + kanatlara giden çizgiler */}
        <path d="M20 14.5 L26 18.5 L26 27 L20 31 L14 27 L14 18.5 Z" />
        <path d="M20 14.5 L20 31 M14 18.5 L6.5 20 M26 18.5 L33.5 20 M14 27 L7 26.5 M26 27 L33 26.5" />
        {/* baş + boyun */}
        <circle cx="20" cy="8.2" r="3" />
        <path d="M20 11.2 L20 14.5" />
        {/* ayaklar */}
        <path d="M9 15.5 L5.5 12.5 M31 15.5 L34.5 12.5 M9 31 L5.5 34.5 M31 31 L34.5 34.5" />
      </g>
    </svg>
  );
}
