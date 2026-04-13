import Image from 'next/image';

export function Logo({ size = 28 }: { size?: number }) {
  return <Image src="/logo.svg" alt="Feature Bacon" width={size} height={size} priority />;
}
