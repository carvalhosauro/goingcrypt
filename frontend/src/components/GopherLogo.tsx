import type { FC } from 'react';

interface GopherLogoProps {
  size?: number;
}

const GopherLogo: FC<GopherLogoProps> = ({ size = 32 }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 40 40"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <defs>
      {/* Background gradient */}
      <linearGradient id="bgGrad" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stopColor="#0d1117" />
        <stop offset="100%" stopColor="#161b2e" />
      </linearGradient>

      {/* Lock body gradient */}
      <linearGradient id="lockGrad" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stopColor="#00d2ff" />
        <stop offset="100%" stopColor="#a855f7" />
      </linearGradient>

      {/* Glow filter */}
      <filter id="glow" x="-30%" y="-30%" width="160%" height="160%">
        <feGaussianBlur stdDeviation="1.8" result="blur" />
        <feMerge>
          <feMergeNode in="blur" />
          <feMergeNode in="SourceGraphic" />
        </feMerge>
      </filter>
    </defs>

    {/* Background rounded square */}
    <rect width="40" height="40" rx="10" fill="url(#bgGrad)" />

    {/* Shackle (arc) */}
    <path
      d="M13.5 19v-5.5a6.5 6.5 0 0 1 13 0V19"
      stroke="url(#lockGrad)"
      strokeWidth="2.2"
      strokeLinecap="round"
      fill="none"
      filter="url(#glow)"
    />

    {/* Lock body */}
    <rect
      x="10"
      y="19"
      width="20"
      height="14"
      rx="3.5"
      fill="url(#lockGrad)"
      filter="url(#glow)"
      opacity="0.95"
    />

    {/* Keyhole */}
    <circle cx="20" cy="25" r="2.2" fill="white" opacity="0.9" />
    <rect x="19.1" y="25.8" width="1.8" height="3" rx="0.9" fill="white" opacity="0.9" />
  </svg>
);

export default GopherLogo;
